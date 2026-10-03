package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// A non-streamed request is asked of the model server as a stream, and its
// answer is assembled here into the one object the model server would have
// returned unstreamed (iss-2610030919536329).
//
// The reason is the pinned mlx-lm 0.31.3's timing. Unstreamed, it sends no
// header until the whole answer is generated, and its token loop writes
// nothing before the end, so it never learns that the client has gone: the
// gateway's wait for headers then bounded prefill AND the whole answer, and a
// client that gave up left the server generating to max_tokens for nobody.
// Streamed, the headers go out once the request is admitted and every chunk is
// a write, so the wait bounds admission alone and a hang-up stops generation
// at the next chunk.
//
// The assembled object is the unstreamed one field for field, read off
// server.py's two builders side by side (generate_response for the chunk and
// for the answer, completion_usage_response for the counts):
//
//   - id, system_fingerprint, model and created are the same request
//     constants in every event and in the answer, and are taken from the
//     first event that carries a choice;
//   - object is the chunk's with its ".chunk" suffix removed:
//     "chat.completion.chunk" answers as "chat.completion", and
//     "text_completion" is the same streamed and not;
//   - each choice keeps its index; its finish_reason is the last one the
//     stream gave (every chunk but the last carries null); a chat choice's
//     delta becomes its message — role as the first delta gave it, content
//     and reasoning joined in order, tool_calls appended in order with the
//     "index" the streaming formatter adds removed; a text completion's text
//     is joined in order;
//   - usage is the counts-only event's, which the server builds from the same
//     three figures as the unstreamed answer's (prompt, completion, and the
//     prompt-cache detail when it has one).
//
// What a stream cannot carry is logprobs: the pinned server writes them only
// into an unstreamed answer. A request asking for them is therefore relayed
// unstreamed as before (see assembles).
//
// Nothing of the answer is logged, kept or counted. It is held only while it
// is assembled, as an unstreamed answer has always been held whole while its
// model field is rewritten, and under the same cap.

// assembles reports whether a request is one whose answer the gateway
// assembles: the client asked for no stream — "stream" absent or false — and
// for nothing a stream cannot carry.
//
// Any other value of "stream" is relayed as it came, and that includes the
// values Python calls false. The pinned server reads the field without
// testing its truth — it validates it first, with
// _validate("stream", bool), and a 0, null, "", [] or {} fails that check
// and is refused before anything is generated. Asking for a stream in its
// place would answer a request the model server refuses; true is a stream
// already.
func assembles(payload map[string]json.RawMessage) bool {
	return streamOff(payload) && !asksForLogprobs(payload)
}

// streamOff reports whether the client asked for no stream in a form the
// model server accepts: "stream" absent, or false.
func streamOff(payload map[string]json.RawMessage) bool {
	raw, ok := payload["stream"]
	return !ok || string(bytes.TrimSpace(raw)) == "false"
}

// asksForLogprobs reports whether the request asks for logprobs, which the
// pinned server writes only into an unstreamed answer.
func asksForLogprobs(payload map[string]json.RawMessage) bool {
	if raw, ok := payload["logprobs"]; ok && truthy(raw) {
		return true
	}
	if raw, ok := payload["top_logprobs"]; ok {
		var n float64
		if json.Unmarshal(raw, &n) == nil && n > 0 {
			return true
		}
	}
	return false
}

// answeredWhole reports whether the model server is asked for its answer in
// one piece: a request the client did not ask to stream, kept unstreamed
// because it asked for logprobs. Read after askForStream, it is false for a
// request being assembled, whose "stream" is then true.
func answeredWhole(payload map[string]json.RawMessage) bool {
	return streamOff(payload) && asksForLogprobs(payload)
}

// askForStream turns a request assembles accepted into the one sent to the
// model server: streamed, with the counts asked for. The client's own
// stream_options, if it sent one, is replaced rather than merged: OpenAI
// refuses it without a stream and the pinned server ignores it unstreamed, so
// it meant nothing, and the server reads stream_options["include_usage"]
// directly — a malformed one would make it raise part-way through the answer.
func askForStream(payload map[string]json.RawMessage) {
	payload["stream"] = json.RawMessage("true")
	payload[streamOptionsField] = json.RawMessage(`{"` + includeUsageField + `":true}`)
}

var (
	// errAnswerTooLarge is an answer whose assembled parts passed the cap.
	errAnswerTooLarge = errors.New("the answer is larger than the gateway assembles")
	// errStreamCut is a stream that ended, or failed, before its "[DONE]".
	errStreamCut = errors.New("the model server's stream ended before the answer did")
	// errStreamFailed is a stream that said it failed, or said something
	// that is not a completion event.
	errStreamFailed = errors.New("the model server's stream carried an error")
)

// assembleAnswer reads a streamed answer to its "[DONE]" and returns the
// unstreamed object for it. No more than limit bytes of the answer's own parts
// are held, whatever the stream's framing adds, and no line longer than the
// relay's own line cap is read.
func assembleAnswer(src io.Reader, limit int) ([]byte, error) {
	a := &assembly{limit: limit}
	br := bufio.NewReader(src)
	for {
		line, err := readBoundedLine(br, maxStreamLine)
		if errors.Is(err, errLineTooLong) {
			return nil, errLineTooLong
		}
		if _, payload, ok := cutDataPrefix(line); ok {
			if isTerminalEvent(payload) {
				return a.render()
			}
			if aerr := a.add(payload); aerr != nil {
				return nil, aerr
			}
		}
		// Anything else — a ": keepalive" comment the server writes during
		// prefill, the blank line that ends an event — carries no answer.
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errStreamCut
			}
			return nil, errors.Join(errStreamCut, err)
		}
	}
}

// member is one field of a JSON object, in the order it was written.
type member struct {
	key string
	raw json.RawMessage
}

// assembly is an answer being put together.
type assembly struct {
	limit, held int
	// frame is the first event's top-level fields other than choices and
	// usage, in its order.
	frame   []member
	choices []*choiceParts
	usage   json.RawMessage
}

// choiceParts is one choice being put together.
type choiceParts struct {
	index        json.RawMessage
	finishReason json.RawMessage
	// text is a text completion's text, joined; hasText says it had one.
	text    []byte
	hasText bool
	// message is a chat choice's deltas, joined; nil for a text completion.
	message *messageParts
	// other is any further field of the choice, last value kept.
	other []member
}

// messageParts is a chat answer's message being put together.
type messageParts struct {
	role json.RawMessage
	// strings are the text fields — content, reasoning — joined in order.
	strings []stringPart
	// toolCalls are the calls, in order, each with its streaming index gone.
	toolCalls []json.RawMessage
	hasTools  bool
	// other is any further field, last value kept.
	other []member
}

type stringPart struct {
	key string
	buf []byte
}

// The fields of a completion this file names, beyond the ones gateway.go
// already does.
const (
	deltaField        = "delta"
	messageField      = "message"
	textField         = "text"
	indexField        = "index"
	finishReasonField = "finish_reason"
	toolCallsField    = "tool_calls"
	answerRoleField   = "role"
	errorField        = "error"
	objectField       = "object"
)

// messageOrder is the order the pinned server writes a message's fields in.
var messageOrder = []string{answerRoleField, "content", "reasoning", toolCallsField}

func (a *assembly) hold(n int) error {
	a.held += n
	if a.held > a.limit {
		return errAnswerTooLarge
	}
	return nil
}

// add folds one event into the answer.
func (a *assembly) add(payload []byte) error {
	ev, err := decodeOrdered(payload)
	if err != nil {
		return errStreamFailed
	}
	var choicesRaw json.RawMessage
	hasChoices := false
	for _, m := range ev {
		switch m.key {
		case errorField:
			if string(m.raw) != "null" {
				return errStreamFailed
			}
		case usageField:
			if string(m.raw) != "null" {
				if err := a.hold(len(m.raw)); err != nil {
					return err
				}
				a.usage = m.raw
			}
		case choicesField:
			choicesRaw, hasChoices = m.raw, true
		}
	}
	if !hasChoices {
		return errStreamFailed
	}
	choices, ok := decodeChoices(choicesRaw)
	if !ok {
		return errStreamFailed
	}
	if len(choices) == 0 {
		// The counts-only event, read above.
		return nil
	}
	if a.frame == nil {
		for _, m := range ev {
			if m.key == choicesField || m.key == usageField {
				continue
			}
			if m.key == objectField {
				m.raw = unchunked(m.raw)
			}
			a.frame = append(a.frame, m)
			if err := a.hold(len(m.raw)); err != nil {
				return err
			}
		}
	}
	for _, raw := range choices {
		if err := a.addChoice(raw); err != nil {
			return err
		}
	}
	return nil
}

// unchunked is a chunk's object type as the unstreamed answer spells it.
func unchunked(raw json.RawMessage) json.RawMessage {
	var s string
	if json.Unmarshal(raw, &s) != nil || !strings.HasSuffix(s, ".chunk") {
		return raw
	}
	out, err := json.Marshal(strings.TrimSuffix(s, ".chunk"))
	if err != nil {
		return raw
	}
	return out
}

func (a *assembly) addChoice(raw json.RawMessage) error {
	fields, err := decodeOrdered(raw)
	if err != nil {
		return errStreamFailed
	}
	var index json.RawMessage = json.RawMessage("0")
	for _, m := range fields {
		if m.key == indexField {
			index = m.raw
		}
	}
	var c *choiceParts
	for _, have := range a.choices {
		if bytes.Equal(have.index, index) {
			c = have
		}
	}
	if c == nil {
		c = &choiceParts{index: index, finishReason: json.RawMessage("null")}
		a.choices = append(a.choices, c)
	}
	for _, m := range fields {
		switch m.key {
		case indexField:
		case finishReasonField:
			if string(m.raw) != "null" {
				c.finishReason = m.raw
			}
		case textField:
			s, ok := stringBody(m.raw)
			if !ok {
				return errStreamFailed
			}
			if err := a.hold(len(s)); err != nil {
				return err
			}
			c.text = append(c.text, s...)
			c.hasText = true
		case deltaField:
			if c.message == nil {
				c.message = &messageParts{}
			}
			if err := a.addDelta(c.message, m.raw); err != nil {
				return err
			}
		default:
			if c.other, err = a.set(c.other, m); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *assembly) addDelta(msg *messageParts, raw json.RawMessage) error {
	fields, err := decodeOrdered(raw)
	if err != nil {
		return errStreamFailed
	}
	for _, m := range fields {
		if string(m.raw) == "null" {
			continue
		}
		// Only what is kept counts against the cap: the role every chunk
		// repeats is held once, and a stream's framing not at all.
		switch {
		case m.key == answerRoleField:
			if msg.role == nil {
				if err := a.hold(len(m.raw)); err != nil {
					return err
				}
				msg.role = m.raw
			}
		case m.key == toolCallsField:
			calls, ok := decodeChoices(m.raw)
			if !ok {
				return errStreamFailed
			}
			for _, call := range calls {
				fields, err := decodeOrdered(call)
				if err != nil {
					return errStreamFailed
				}
				kept := encodeOrdered(withoutMember(fields, indexField))
				if err := a.hold(len(kept)); err != nil {
					return err
				}
				msg.toolCalls = append(msg.toolCalls, kept)
				msg.hasTools = true
			}
		default:
			s, ok := stringBody(m.raw)
			if !ok {
				if msg.other, err = a.set(msg.other, m); err != nil {
					return err
				}
				continue
			}
			if err := a.hold(len(s)); err != nil {
				return err
			}
			found := false
			for i := range msg.strings {
				if msg.strings[i].key == m.key {
					msg.strings[i].buf = append(msg.strings[i].buf, s...)
					found = true
				}
			}
			if !found {
				msg.strings = append(msg.strings, stringPart{key: m.key, buf: append([]byte(nil), s...)})
			}
		}
	}
	return nil
}

// render writes the assembled answer in the order the pinned server writes
// an unstreamed one: the frame, the choices, the counts.
func (a *assembly) render() ([]byte, error) {
	if len(a.choices) == 0 {
		// Not even the final chunk, which the server always writes.
		return nil, errStreamFailed
	}
	out := append([]member(nil), a.frame...)
	var choices bytes.Buffer
	choices.WriteByte('[')
	for i, c := range a.choices {
		if i > 0 {
			choices.WriteByte(',')
		}
		fields := []member{{indexField, c.index}, {finishReasonField, c.finishReason}}
		fields = append(fields, c.other...)
		switch {
		case c.message != nil:
			fields = append(fields, member{messageField, c.message.render()})
		case c.hasText:
			fields = append(fields, member{textField, quoted(c.text)})
		}
		choices.Write(encodeOrdered(fields))
	}
	choices.WriteByte(']')
	out = append(out, member{choicesField, choices.Bytes()})
	if a.usage != nil {
		out = append(out, member{usageField, a.usage})
	}
	return encodeOrdered(out), nil
}

func (m *messageParts) render() json.RawMessage {
	byKey := map[string]json.RawMessage{}
	var extra []string
	if m.role != nil {
		byKey[answerRoleField] = m.role
	}
	for _, s := range m.strings {
		byKey[s.key] = quoted(s.buf)
		extra = append(extra, s.key)
	}
	if m.hasTools {
		var b bytes.Buffer
		b.WriteByte('[')
		for i, call := range m.toolCalls {
			if i > 0 {
				b.WriteByte(',')
			}
			b.Write(call)
		}
		b.WriteByte(']')
		byKey[toolCallsField] = b.Bytes()
	}
	for _, o := range m.other {
		byKey[o.key] = o.raw
		extra = append(extra, o.key)
	}
	var fields []member
	placed := map[string]bool{}
	for _, k := range append(append([]string(nil), messageOrder...), extra...) {
		if raw, ok := byKey[k]; ok && !placed[k] {
			fields = append(fields, member{k, raw})
			placed[k] = true
		}
	}
	return encodeOrdered(fields)
}

// stringBody returns the inside of a JSON string literal, escapes and all,
// and whether raw is one. Joining two such insides is the inside of the
// literal for the joined strings: every escape is complete within the literal
// it came from, so the answer is re-quoted without being decoded and every
// character keeps the spelling the model server gave it.
func stringBody(raw json.RawMessage) ([]byte, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil, false
	}
	return raw[1 : len(raw)-1], true
}

func quoted(body []byte) json.RawMessage {
	out := make([]byte, 0, len(body)+2)
	out = append(out, '"')
	out = append(out, body...)
	return append(out, '"')
}

// set keeps the last value a field was given, counting what that adds to
// what is held.
func (a *assembly) set(ms []member, m member) ([]member, error) {
	for i := range ms {
		if ms[i].key == m.key {
			if grew := len(m.raw) - len(ms[i].raw); grew > 0 {
				if err := a.hold(grew); err != nil {
					return nil, err
				}
			}
			ms[i].raw = m.raw
			return ms, nil
		}
	}
	if err := a.hold(len(m.raw)); err != nil {
		return nil, err
	}
	return append(ms, m), nil
}

func withoutMember(ms []member, key string) []member {
	out := ms[:0:0]
	for _, m := range ms {
		if m.key != key {
			out = append(out, m)
		}
	}
	return out
}

// decodeOrdered parses one JSON object into its fields in the order they were
// written, each value's own bytes untouched.
func decodeOrdered(b []byte) ([]member, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not an object")
	}
	var out []member
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, errors.New("not an object")
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		out = append(out, member{key, raw})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the object")
	}
	return out, nil
}

// encodeOrdered writes fields as one JSON object in their order.
func encodeOrdered(ms []member) json.RawMessage {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range ms {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(m.key)
		b.Write(key)
		b.WriteByte(':')
		b.Write(m.raw)
	}
	b.WriteByte('}')
	return b.Bytes()
}

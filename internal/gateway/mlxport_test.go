package gateway

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// This file is a port of the parts of the pinned mlx-lm 0.31.3 server
// (mlx_lm/server.py) that decide what a completion answer looks like on the
// wire, so the gateway's tests can be held to the model server's own builders
// rather than to a shape this repository made up:
//
//   - APIHandler.generate_response, for both the streamed chunk and the
//     non-streamed answer;
//   - APIHandler.completion_usage_response, the counts-only event;
//   - ToolCallFormatter, with its streaming-only "index";
//   - the token loop of APIHandler.handle_completion, which decides when a
//     chunk is written, how text, reasoning and tool calls are gathered, and
//     how "stop" becomes "tool_calls";
//   - Python's json.dumps with its default separators and ensure_ascii.
//
// It is test code: it exists so that a golden test compares what the gateway
// assembles from a stream with what the same server would have answered
// unstreamed, both produced by the same port of the same builders.

// pyKV is one member of a Python dict, which keeps insertion order.
type pyKV struct {
	k string
	v any
}

// pyObj is a Python dict.
type pyObj []pyKV

// pyNone is Python's None, spelt so a nil interface is never ambiguous.
type pyNoneT struct{}

var pyNone = pyNoneT{}

// pyDumps is json.dumps(v) with Python's defaults: ", " and ": " separators,
// and ensure_ascii, which escapes everything outside printable ASCII.
func pyDumps(v any) string {
	var b strings.Builder
	pyEncode(&b, v)
	return b.String()
}

func pyEncode(b *strings.Builder, v any) {
	switch x := v.(type) {
	case pyNoneT:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.Itoa(x))
	case string:
		pyEncodeString(b, x)
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			pyEncode(b, e)
		}
		b.WriteByte(']')
	case pyObj:
		b.WriteByte('{')
		for i, kv := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			pyEncodeString(b, kv.k)
			b.WriteString(": ")
			pyEncode(b, kv.v)
		}
		b.WriteByte('}')
	default:
		panic(fmt.Sprintf("pyEncode: %T", v))
	}
}

// pyEncodeString is json.encoder.py_encode_basestring_ascii: a backslash
// escape for the seven characters that have one, and \uXXXX (lower-case hex,
// a surrogate pair above the BMP) for every other character outside ' '..'~'.
func pyEncodeString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r >= ' ' && r <= '~':
				b.WriteRune(r)
			case r > 0xFFFF:
				r -= 0x10000
				fmt.Fprintf(b, `\u%04x\u%04x`, 0xD800|((r>>10)&0x3FF), 0xDC00|(r&0x3FF))
			default:
				fmt.Fprintf(b, `\u%04x`, r)
			}
		}
	}
	b.WriteByte('"')
}

// mlxGen is one token as the server's token loop sees it: the text it
// decoded to, the state machine's state (normal, reasoning or tool), and the
// finish reason generation reported with it, empty for None.
type mlxGen struct {
	text, state, finish string
}

// mlxToolCall is what the model's tool parser returns for one tool text:
// a name, the arguments as json.dumps(..., ensure_ascii=False) renders them,
// and the id the parser carried, which ToolCallFormatter pops.
type mlxToolCall struct {
	name, arguments, id string
}

// mlxRequest is everything about one request that decides its answer.
type mlxRequest struct {
	// chat is /v1/chat/completions; otherwise /v1/completions.
	chat bool
	// id, fingerprint, model and created are the per-request constants every
	// event of the answer repeats.
	id, fingerprint, model string
	created                int
	gens                   []mlxGen
	// tools maps a tool text gathered in the "tool" state to what the
	// parser makes of it.
	tools map[string][]mlxToolCall
	// promptTokens is len(ctx.prompt); cached is ctx.prompt_cache_count,
	// where -1 means no figure.
	promptTokens, cached int
}

func (q mlxRequest) objectType(stream bool) string {
	switch {
	case !q.chat:
		return "text_completion"
	case stream:
		return "chat.completion.chunk"
	default:
		return "chat.completion"
	}
}

// mlxFormatter is ToolCallFormatter: one per request, whose index counts
// across every call it formats when streaming.
type mlxFormatter struct {
	q         mlxRequest
	streaming bool
	idx       int
}

func (f *mlxFormatter) format(texts []string) []any {
	var out []any
	for _, text := range texts {
		for _, tc := range f.q.tools[text] {
			o := pyObj{
				{"function", pyObj{{"name", tc.name}, {"arguments", tc.arguments}}},
				{"type", "function"},
				{"id", tc.id},
			}
			if f.streaming {
				o = append(o, pyKV{"index", f.idx})
				f.idx++
			}
			out = append(out, o)
		}
	}
	return out
}

// generateResponse is APIHandler.generate_response.
func (q mlxRequest) generateResponse(stream bool, text string, finish any, usage pyObj, toolCalls []any, reasoning string) pyObj {
	choice := pyObj{{"index", 0}, {"finish_reason", finish}}
	if q.chat {
		key := "message"
		if stream {
			key = "delta"
		}
		msg := pyObj{{"role", "assistant"}}
		if text != "" {
			msg = append(msg, pyKV{"content", text})
		}
		if reasoning != "" {
			msg = append(msg, pyKV{"reasoning", reasoning})
		}
		if len(toolCalls) > 0 {
			msg = append(msg, pyKV{"tool_calls", toolCalls})
		}
		choice = append(choice, pyKV{key, msg})
	} else {
		choice = append(choice, pyKV{"text", text})
	}
	resp := pyObj{
		{"id", q.id},
		{"system_fingerprint", q.fingerprint},
		{"object", q.objectType(stream)},
		{"model", q.model},
		{"created", q.created},
		{"choices", []any{choice}},
	}
	if !stream {
		resp = append(resp, pyKV{"usage", usage})
	}
	return resp
}

func (q mlxRequest) usage(completionTokens int) pyObj {
	u := pyObj{
		{"prompt_tokens", q.promptTokens},
		{"completion_tokens", completionTokens},
		{"total_tokens", q.promptTokens + completionTokens},
	}
	if q.cached >= 0 {
		u = append(u, pyKV{"prompt_tokens_details", pyObj{{"cached_tokens", q.cached}}})
	}
	return u
}

// completionUsageResponse is APIHandler.completion_usage_response, whose
// object is "chat.completion" whatever the endpoint.
func (q mlxRequest) completionUsageResponse(completionTokens int) pyObj {
	return pyObj{
		{"id", q.id},
		{"system_fingerprint", q.fingerprint},
		{"object", "chat.completion"},
		{"model", q.model},
		{"created", q.created},
		{"choices", []any{}},
		{"usage", q.usage(completionTokens)},
	}
}

// run is the token loop of APIHandler.handle_completion. Streamed, it returns
// each event's JSON in the order the server writes them — the chunks, the
// final chunk, and the counts-only event when includeUsage — without the
// "[DONE]" that follows. Unstreamed, it returns the one answer.
func (q mlxRequest) run(stream, includeUsage bool) []string {
	f := &mlxFormatter{q: q, streaming: stream}
	var (
		events       []string
		prevState    string
		finish       = "stop"
		reasoning    string
		madeToolCall bool
		toolText     string
		toolCalls    []string
		text         string
		tokens       int
	)
	for _, gen := range q.gens {
		switch gen.state {
		case "reasoning":
			reasoning += gen.text
		case "tool":
			toolText += gen.text
		case "normal":
			if prevState == "tool" {
				toolCalls = append(toolCalls, toolText)
				toolText = ""
				madeToolCall = true
			}
			text += gen.text
		}
		tokens++
		if stream && gen.state != "tool" && (text != "" || len(toolCalls) > 0 || reasoning != "") {
			events = append(events, pyDumps(q.generateResponse(true, text, pyNone, nil, f.format(toolCalls), reasoning)))
			reasoning, text, toolCalls = "", "", nil
		}
		if gen.finish != "" {
			finish = gen.finish
		}
		prevState = gen.state
	}
	if prevState == "tool" && toolText != "" {
		toolCalls = append(toolCalls, toolText)
		madeToolCall = true
	}
	if finish == "stop" && madeToolCall {
		finish = "tool_calls"
	}
	if stream {
		events = append(events, pyDumps(q.generateResponse(true, text, finish, nil, f.format(toolCalls), reasoning)))
		if includeUsage {
			events = append(events, pyDumps(q.completionUsageResponse(tokens)))
		}
		return events
	}
	return []string{pyDumps(q.generateResponse(false, text, finish, q.usage(tokens), f.format(toolCalls), reasoning))}
}

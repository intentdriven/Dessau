package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/intentdriven/Dessau/internal/config"
)

// The pinned mlx-lm serves every request from one generation thread, and its
// own checks, run on the request's handler thread, let through values that
// raise once they reach that thread: the thread dies for every client until
// the model restarts (iss-2610031444343397). Each value below is one the audit
// of mlx-lm 0.32.0 found raising there, and is refused here, before anything
// is resolved or loaded, as emptyAnswerBudget and badStop are. A value that
// is not a number, where one is expected, is the model server's own to
// refuse, as it was: its type checks run on the handler thread.

// numericBounds are the sampling fields mlx-lm passes to its sampler and
// logits processors, with the range each is held to. Each bound is far beyond
// any useful value, and each closes a value the generation thread dies on:
// xtc_threshold above one half (sample_utils.py apply_xtc), a top_k at or
// above the vocabulary (apply_top_k; config.MaxTopK is far below every
// shipped vocabulary), and integers too large for a float.
var numericBounds = []struct {
	key      string
	min, max float64
}{
	{"temperature", 0, 100},
	{"top_p", 0, 1},
	{"min_p", 0, 1},
	{"top_k", 0, config.MaxTopK},
	{"xtc_probability", 0, 1},
	{"xtc_threshold", 0, 0.5},
	{"repetition_penalty", 0, 100},
	{"presence_penalty", -100, 100},
	{"frequency_penalty", -100, 100},
	{"repetition_context_size", 0, 1 << 20},
	{"presence_context_size", 0, 1 << 20},
	{"frequency_context_size", 0, 1 << 20},
}

// logit_bias is held to what a client can mean by it: a few hundred token
// ids, each a decimal integer an int32 holds, each biased by at most what
// OpenAI documents (-100 to 100). A key past the model's own vocabulary is
// not known here; the pool's health watch is the net for that.
const (
	maxLogitBiasEntries = 300
	maxLogitBias        = 100
)

// templateOwnArgs are apply_chat_template's own parameters, which
// chat_template_kwargs is merged into (server.py): one of them changes what
// the call returns, past the guard around it, or runs a client's template.
var templateOwnArgs = map[string]bool{
	"chat_template": true, "tokenize": true, "return_dict": true, "return_tensors": true,
	"return_assistant_tokens_mask": true, "add_generation_prompt": true,
	"continue_final_message": true, "tools": true, "documents": true,
	"tokenizer_kwargs": true, "truncation": true, "max_length": true, "padding": true,
	"conversation": true,
}

// generationRefusal is the refusal for a request carrying a value the model
// server's generation thread would die on, or "" for none. The conversation
// itself is not read: adr-2609061610102325 grants that to the merge alone, so
// a prompt that tokenizes to nothing is left to the pool's health watch.
func generationRefusal(payload map[string]json.RawMessage) string {
	for _, b := range numericBounds {
		raw, ok := payload[b.key]
		if !ok {
			continue
		}
		f, isNumber := jsonNumber(raw)
		if isNumber && (math.IsNaN(f) || f < b.min || f > b.max) {
			return fmt.Sprintf("%q must be between %s and %s", b.key, fmtBound(b.min), fmtBound(b.max))
		}
	}
	if msg := badLogitBias(payload); msg != "" {
		return msg
	}
	if msg := badStopText(payload); msg != "" {
		return msg
	}
	if msg := badTemplateKwargs(payload); msg != "" {
		return msg
	}
	return ""
}

// jsonNumber reads a bare JSON number. One too large for a float64 reads as
// infinite, which every bound refuses; anything else — a string, a bool,
// null — is not a number here.
func jsonNumber(raw json.RawMessage) (float64, bool) {
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || !(t[0] == '-' || (t[0] >= '0' && t[0] <= '9')) {
		return 0, false
	}
	var n json.Number
	if json.Unmarshal(t, &n) != nil {
		return 0, false
	}
	f, err := strconv.ParseFloat(n.String(), 64)
	if err != nil && !math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func fmtBound(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// badLogitBias is the refusal for a logit_bias outside the bounds above.
func badLogitBias(payload map[string]json.RawMessage) string {
	raw, ok := payload["logit_bias"]
	if !ok || string(bytes.TrimSpace(raw)) == "null" {
		return ""
	}
	var bias map[string]json.RawMessage
	if json.Unmarshal(raw, &bias) != nil {
		return `"logit_bias" must be an object of token ids to numbers`
	}
	if len(bias) > maxLogitBiasEntries {
		return fmt.Sprintf(`"logit_bias" may name at most %d tokens`, maxLogitBiasEntries)
	}
	for k, v := range bias {
		id, err := strconv.ParseInt(k, 10, 32)
		if err != nil || id < 0 || strconv.FormatInt(id, 10) != k {
			return `"logit_bias" keys must be token ids: whole numbers from 0`
		}
		f, isNumber := jsonNumber(v)
		if !isNumber || math.IsNaN(f) || f < -maxLogitBias || f > maxLogitBias {
			return fmt.Sprintf(`"logit_bias" values must be numbers between %d and %d`, -maxLogitBias, maxLogitBias)
		}
	}
	return ""
}

// badStopText is the refusal for a stop string the model server's tokenizer
// cannot encode: one carrying a surrogate escape that is not half of a pair.
// Go decodes such an escape to U+FFFD, so badStop passes it, but the model
// server's Python keeps the lone surrogate, and encoding it raises on the
// generation thread. The raw bytes are what is read.
func badStopText(payload map[string]json.RawMessage) string {
	raw, ok := payload["stop"]
	if !ok {
		return ""
	}
	if loneSurrogateEscape(raw) {
		return `"stop" must be text: it carries a surrogate escape that is not half of a pair`
	}
	return ""
}

// loneSurrogateEscape reports whether raw JSON carries a \u escape of a
// surrogate that is not half of a high-then-low pair. A backslash escaped by
// another is not the start of an escape.
func loneSurrogateEscape(raw []byte) bool {
	hex4 := func(i int) (int, bool) {
		if i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
			return 0, false
		}
		v, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
		return int(v), err == nil
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		if i+1 < len(raw) && raw[i+1] != 'u' {
			i++ // \\, \", \n and the rest: the escaped byte is not a backslash
			continue
		}
		v, ok := hex4(i)
		if !ok {
			continue
		}
		switch {
		case v >= 0xD800 && v <= 0xDBFF:
			if lo, ok := hex4(i + 6); ok && lo >= 0xDC00 && lo <= 0xDFFF {
				i += 11
				continue
			}
			return true
		case v >= 0xDC00 && v <= 0xDFFF:
			return true
		}
		i += 5
	}
	return false
}

// badTemplateKwargs is the refusal for chat_template_kwargs that are not an
// object of plain values, or that name one of apply_chat_template's own
// parameters.
func badTemplateKwargs(payload map[string]json.RawMessage) string {
	raw, ok := payload["chat_template_kwargs"]
	if !ok || string(bytes.TrimSpace(raw)) == "null" {
		return ""
	}
	var kwargs map[string]json.RawMessage
	if json.Unmarshal(raw, &kwargs) != nil {
		return `"chat_template_kwargs" must be an object`
	}
	for k, v := range kwargs {
		if templateOwnArgs[k] {
			return fmt.Sprintf(`"chat_template_kwargs" may not set %q`, k)
		}
		if t := bytes.TrimSpace(v); len(t) == 0 || t[0] == '{' || t[0] == '[' {
			return `"chat_template_kwargs" values must be booleans, numbers or strings`
		}
	}
	return ""
}

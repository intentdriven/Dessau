package gateway

import (
	"bytes"
	"encoding/json"
	"strings"
)

// promptField is the field a /v1/completions request carries its text in.
const promptField = "prompt"

// emptyConversation is the refusal for a request with nothing to generate
// from, or "" for none: a /v1/completions prompt that is not a JSON string, or
// is a string that is empty or only whitespace, or a chat request whose
// messages are an empty array.
//
// One empty prompt freezes the pinned mlx-lm's model server: the process
// stays up, Dessau still reports the model loaded, and every later request to
// that model hangs until the model is unloaded (iss-2610031758029994, reproduced on the
// Mac with mlx-lm 0.32.0). adr-2610040749545010 grants this check as a narrow
// exception to the conversation boundary: it reads whether the prompt or the
// messages are empty and nothing else — never what a message says, never how
// many there are — and keeps nothing.
//
// A prompt that is an array of strings freezes the same server in the same
// way (["hi"] and [""], iss-2610040805353721, reproduced on the Mac), and a
// token array is refused by it, so no prompt but a string yields a generation
// there. adr-2610042021365934 widens the grant for the prompt alone to its
// JSON kind, read from the first byte of the raw value: nothing inside an
// array or an object is looked at, and only a value that is already a string
// is decoded. A messages value that is not an array, and a missing field, are
// passed on as they were.
func emptyConversation(payload map[string]json.RawMessage, chat bool) string {
	if chat {
		if isEmptyArray(payload[messagesField]) {
			return `"messages" is empty: there is no conversation to answer`
		}
		return ""
	}
	raw, ok := payload[promptField]
	if !ok {
		return ""
	}
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || t[0] != '"' {
		return `"prompt" must be a string`
	}
	var s string
	if json.Unmarshal(t, &s) == nil && strings.TrimSpace(s) == "" {
		return `"prompt" is empty: there is nothing to complete`
	}
	return ""
}

// isEmptyArray reports whether raw is a JSON array with no elements, without
// decoding anything inside one that has some.
func isEmptyArray(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	if len(t) < 2 || t[0] != '[' || t[len(t)-1] != ']' {
		return false
	}
	return len(bytes.TrimSpace(t[1:len(t)-1])) == 0
}

package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// The refusal a body that is not valid UTF-8 gets, word for word: it names
// the encoding and nothing of what the body carried.
const wantInvalidUTF8Refusal = "request body is not valid UTF-8"

// A body whose bytes are not valid UTF-8 is refused before anything is
// resolved or loaded (iss-2610042036094209). Go's decoder keeps such bytes as
// they came inside a RawMessage, and json.Marshal forwards them, so before
// this check they reached the model server, whose request handler fails to
// decode them on its handler thread and drops the connection: the client saw
// a 502 rather than a refusal it could act on. Valid multi-byte UTF-8 —
// accented Latin, CJK, emoji — is relayed as it always was.
func TestABodyThatIsNotValidUTF8IsRefused(t *testing.T) {
	const model = `"model":"mlx-community/Qwen3-8B-4bit"`
	const text = "/v1/completions"
	chat := func(content string) string {
		return `{` + model + `,"messages":[{"role":"user","content":"` + content + `"}]}`
	}
	completion := func(prompt string) string {
		return `{` + model + `,"prompt":"` + prompt + `"}`
	}
	for _, c := range []struct {
		name, path, body string
		refused          bool
	}{
		{"a lone 0xff in a chat message", chatCompletionsPath, chat("hi \xff there"), true},
		{"a truncated sequence in a chat message", chatCompletionsPath, chat("\xe4\xb8"), true},
		{"an overlong encoding in a chat message", chatCompletionsPath, chat("\xc0\xaf"), true},
		{"an encoded surrogate in a chat message", chatCompletionsPath, chat("\xed\xa0\x80"), true},
		{"a lone 0xff in a completions prompt", text, completion("The capital of France is \xff"), true},
		{"a lone 0xff in a key", text, `{` + model + `,"prompt":"hi","x` + "\xff" + `":1}`, true},
		{"accented Latin in a chat message", chatCompletionsPath, chat("Bonjour, ça va ?"), false},
		{"CJK in a chat message", chatCompletionsPath, chat("你好，世界"), false},
		{"emoji in a chat message", chatCompletionsPath, chat("hello 👋🏽 world"), false},
		{"CJK and emoji in a completions prompt", text, completion("東京は 🗼"), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, pool, fake := newTestGateway(t, config.Default())
			req, _ := http.NewRequest(http.MethodPost, srv.URL+c.path, strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			pool.mu.Lock()
			acquired := len(pool.acquired)
			pool.mu.Unlock()
			if !c.refused {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("status %d, %s; want valid UTF-8 relayed", resp.StatusCode, got)
				}
				if fake.LastBody() == nil {
					t.Error("a valid UTF-8 body did not reach the model server")
				}
				return
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status %d, %s; want a 400", resp.StatusCode, got)
			}
			if acquired != 0 || fake.LastBody() != nil {
				t.Error("a body that is not valid UTF-8 reached the pool or the model server")
			}
			var e struct {
				Error struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    int    `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(got, &e); err != nil {
				t.Fatalf("the refusal is not the usual error shape: %v: %s", err, got)
			}
			if e.Error.Message != wantInvalidUTF8Refusal || e.Error.Type != "invalid_request_error" || e.Error.Code != http.StatusBadRequest {
				t.Errorf("refusal %+v; want %q, invalid_request_error, 400", e.Error, wantInvalidUTF8Refusal)
			}
		})
	}
}

// The reference page states the refusal word for word.
func TestTheRequestFieldsReferenceStatesTheUTF8Refusal(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "docs", "request-fields.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "`"+wantInvalidUTF8Refusal+"`") {
		t.Errorf("docs/request-fields.md does not state the refusal %q", wantInvalidUTF8Refusal)
	}
}

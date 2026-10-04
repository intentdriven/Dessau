package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/stats"
)

// A request with nothing to generate from is refused before anything is
// resolved or loaded: one empty prompt froze the pinned mlx-lm's model server
// on the Mac, and every later request to that model hung until it was
// unloaded (iss-2610031758029994). Only emptiness is refused here; a request
// with anything in it is relayed as it always was, and a missing field stays
// the model server's to refuse. A prompt that is not a string is the next
// test's.
func TestARequestWithNothingToGenerateFromIsRefused(t *testing.T) {
	const model = `"model":"mlx-community/Qwen3-8B-4bit"`
	const text = "/v1/completions"
	for _, c := range []struct {
		name, path, body string
		refused          bool
	}{
		{"an empty prompt", text, `{` + model + `,"prompt":""}`, true},
		{"a prompt of spaces", text, `{` + model + `,"prompt":"   "}`, true},
		{"a prompt of mixed whitespace", text, `{` + model + `,"prompt":" \n\t\r "}`, true},
		{"a prompt of escaped whitespace", text, `{` + model + `,"prompt":" \u000a"}`, true},
		{"an empty prompt array", text, `{` + model + `,"prompt":[ ]}`, true},
		{"the last of a duplicated prompt, empty", text, `{` + model + `,"prompt":"hi","prompt":""}`, true},
		{"a duplicated prompt whose last has text", text, `{` + model + `,"prompt":"","prompt":"hi"}`, false},
		{"empty messages", chatCompletionsPath, `{` + model + `,"messages":[]}`, true},
		{"empty messages with space inside", chatCompletionsPath, `{` + model + `,"messages":[ ` + "\n" + ` ]}`, true},
		{"an ordinary prompt", text, `{` + model + `,"prompt":"The capital of France is"}`, false},
		{"a prompt with a leading space", text, `{` + model + `,"prompt":" One plus one equals"}`, false},
		{"no prompt at all, for the server to refuse", text, `{` + model + `}`, false},
		{"messages on the completions path, which reads the prompt", text, `{` + model + `,"prompt":"hi","messages":[]}`, false},
		{"ordinary messages", chatCompletionsPath, `{` + model + `,"messages":[{"role":"user","content":"hi"}]}`, false},
		{"a message with empty content, which the template still frames", chatCompletionsPath, `{` + model + `,"messages":[{"role":"user","content":""}]}`, false},
		{"a prompt on the chat path, which reads the messages", chatCompletionsPath, `{` + model + `,"prompt":"","messages":[{"role":"user","content":"hi"}]}`, false},
		{"no messages at all, for the server to refuse", chatCompletionsPath, `{` + model + `}`, false},
		{"messages that are not an array, for the server to refuse", chatCompletionsPath, `{` + model + `,"messages":"[]"}`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, pool, fake := newTestGateway(t, config.Default())
			req, _ := http.NewRequest(http.MethodPost, srv.URL+c.path, strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			pool.mu.Lock()
			acquired := len(pool.acquired)
			pool.mu.Unlock()
			if c.refused {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("status %d, %s; want a 400", resp.StatusCode, body)
				}
				if acquired != 0 || fake.LastBody() != nil {
					t.Error("a refused request reached the pool or the model server")
				}
				return
			}
			if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "is empty") {
				t.Errorf("a request with something in it was refused as empty: %s", body)
			}
		})
	}
}

// A /v1/completions prompt that is not a JSON string is refused before
// anything is resolved or loaded: on the Mac, ["hi"] and [""] each froze the
// pinned mlx-lm's model server as the empty prompt did, and a token array is
// refused by it with a 404, so no prompt of another kind yields a generation
// there (iss-2610040805353721, adr-2610042021365934). Only the value's kind is
// read; a string, whatever it says, is relayed.
func TestACompletionsPromptThatIsNotAStringIsRefused(t *testing.T) {
	const model = `"model":"mlx-community/Qwen3-8B-4bit"`
	for _, c := range []struct {
		name, prompt string
		refused      bool
	}{
		{"an array of one string", `["hi"]`, true},
		{"an array of one empty string", `[""]`, true},
		{"an array of token ids", `[72,73]`, true},
		{"a nested array", `[["hi"]]`, true},
		{"a number", `123`, true},
		{"an object", `{}`, true},
		{"null", `null`, true},
		{"a boolean", `true`, true},
		{"null, after a string given first", `"hi","prompt":null`, true},
		{"a string", `"The capital of France is"`, false},
		{"a string that looks like an array", `"[\"hi\"]"`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, pool, fake := newTestGateway(t, config.Default())
			body := `{` + model + `,"prompt":` + c.prompt + `}`
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/completions", strings.NewReader(body))
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
					t.Errorf("status %d, %s; want the string prompt relayed", resp.StatusCode, got)
				}
				if fake.LastBody() == nil {
					t.Error("a string prompt did not reach the model server")
				}
				return
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status %d, %s; want a 400", resp.StatusCode, got)
			}
			if !strings.Contains(string(got), `\"prompt\" must be a string`) {
				t.Errorf("the refusal does not say the prompt must be a string: %s", got)
			}
			if acquired != 0 || fake.LastBody() != nil {
				t.Error("a refused request reached the pool or the model server")
			}
		})
	}
}

// The bridge's in-process path holds the same rule as the network's.
func TestTheBridgeRefusesEmptyMessages(t *testing.T) {
	g, _, fake := askGateway(t, "ok")
	err := g.Ask(context.Background(), AskRequest{
		Model:  testModelID,
		Body:   []byte(`{"model":"` + testModelID + `","messages":[]}`),
		Source: stats.SourceBridge,
	})
	if err == nil {
		t.Fatal("the bridge relayed a request with no messages")
	}
	var refusal *AskError
	if !errors.As(err, &refusal) || refusal.Public() != genericRefusal {
		t.Errorf("the refusal travelling to the platform is %v, want the generic text", err)
	}
	if fake.LastBody() != nil {
		t.Error("the refused request reached the model server")
	}
}

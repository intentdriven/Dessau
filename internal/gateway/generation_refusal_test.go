package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/stats"
)

// A request value the pinned mlx-lm's own checks let through and its one
// generation thread then dies on is refused before anything is resolved or
// loaded; the values either side of each bound are relayed as they always
// were (iss-2610031444343397 and the audit's findings).
func TestAValueTheGenerationThreadWouldDieOnIsRefused(t *testing.T) {
	const chat = `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],`
	const text = `{"model":"mlx-community/Qwen3-8B-4bit",`
	huge := strings.Repeat("9", 310)
	bias := func(n int) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf(`"%d":1`, i)
		}
		return `{` + strings.Join(parts, ",") + `}`
	}
	for _, c := range []struct {
		name, path, body string
		refused          bool
	}{
		{"xtc_threshold above one half", chatCompletionsPath, chat + `"xtc_probability":0.5,"xtc_threshold":0.6}`, true},
		{"xtc_threshold one half", chatCompletionsPath, chat + `"xtc_probability":0.5,"xtc_threshold":0.5}`, false},
		{"top_k above the cap", chatCompletionsPath, chat + `"top_k":1025}`, true},
		{"top_k beyond a float", chatCompletionsPath, chat + `"top_k":` + huge + `}`, true},
		{"top_k at the cap", chatCompletionsPath, chat + `"top_k":1024}`, false},
		{"temperature beyond a float", chatCompletionsPath, chat + `"temperature":` + huge + `}`, true},
		{"temperature 1e400", chatCompletionsPath, chat + `"temperature":1e400}`, true},
		{"temperature too hot", chatCompletionsPath, chat + `"temperature":101}`, true},
		{"temperature ordinary", chatCompletionsPath, chat + `"temperature":0.7}`, false},
		{"repetition_penalty huge", chatCompletionsPath, chat + `"repetition_penalty":1e20}`, true},
		{"presence_penalty below the floor", chatCompletionsPath, chat + `"presence_penalty":-101}`, true},
		{"frequency_penalty ordinary", chatCompletionsPath, chat + `"frequency_penalty":0.5}`, false},
		{"repetition_context_size huge", chatCompletionsPath, chat + `"repetition_context_size":1048577}`, true},
		{"repetition_context_size ordinary", chatCompletionsPath, chat + `"repetition_context_size":64}`, false},
		{"logit_bias with too many entries", chatCompletionsPath, chat + `"logit_bias":` + bias(301) + `}`, true},
		{"logit_bias with a negative key", chatCompletionsPath, chat + `"logit_bias":{"-1":5}}`, true},
		{"logit_bias with a huge key", chatCompletionsPath, chat + `"logit_bias":{"99999999999":5}}`, true},
		{"logit_bias with a word for a key", chatCompletionsPath, chat + `"logit_bias":{"abc":5}}`, true},
		{"logit_bias with a huge value", chatCompletionsPath, chat + `"logit_bias":{"12":1000}}`, true},
		{"logit_bias not an object", chatCompletionsPath, chat + `"logit_bias":[1,2]}`, true},
		{"logit_bias ordinary, refused until keys can be held to the vocabulary", chatCompletionsPath, chat + `"logit_bias":{"12":-100,"7":2.5}}`, true},
		{"logit_bias null, which is unset", chatCompletionsPath, chat + `"logit_bias":null}`, false},
		{"a key spelled with an escape", chatCompletionsPath, chat + `"top\u005fk":5000}`, true},
		{"the last of a duplicated key", chatCompletionsPath, chat + `"top_k":5,"top_k":5000}`, true},
		{"a duplicated key whose last is fine", chatCompletionsPath, chat + `"top_k":5000,"top_k":5}`, false},
		{"a numeric field written as a string, for the server to refuse", chatCompletionsPath, chat + `"top_k":"5000"}`, false},
		{"stop with an escaped backslash before a lone surrogate", chatCompletionsPath, chat + `"stop":["\\\ud800"]}`, true},
		{"stop with a high surrogate before a plain escape", chatCompletionsPath, chat + `"stop":["\ud800\u0041"]}`, true},
		{"stop with an uppercase lone surrogate", chatCompletionsPath, chat + `"stop":["\uD800"]}`, true},
		{"stop with two high surrogates", chatCompletionsPath, chat + `"stop":["\ud800\ud800\udc00"]}`, true},
		{"stop with a lone surrogate", chatCompletionsPath, chat + `"stop":["\ud800"]}`, true},
		{"stop with a lone low surrogate", chatCompletionsPath, chat + `"stop":"a\udc00b"}`, true},
		{"stop with a surrogate pair", chatCompletionsPath, chat + `"stop":["\ud83d\ude00"]}`, false},
		{"stop with an escaped backslash before u", chatCompletionsPath, chat + `"stop":["\\ud800"]}`, false},
		{"chat_template_kwargs that change the template's answer", chatCompletionsPath, chat + `"chat_template_kwargs":{"return_dict":true}}`, true},
		{"chat_template_kwargs carrying a template", chatCompletionsPath, chat + `"chat_template_kwargs":{"chat_template":"{{ 1 }}"}}`, true},
		{"chat_template_kwargs with an object value", chatCompletionsPath, chat + `"chat_template_kwargs":{"x":{"a":1}}}`, true},
		{"chat_template_kwargs not an object", chatCompletionsPath, chat + `"chat_template_kwargs":"x"}`, true},
		{"chat_template_kwargs turning thinking off", chatCompletionsPath, chat + `"chat_template_kwargs":{"enable_thinking":false}}`, false},
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
			if resp.StatusCode == http.StatusBadRequest {
				t.Errorf("a request the server can answer was refused: %s", body)
			}
		})
	}
}

// A budget too large for a float64 counts as the largest there is, so the
// served-window check still sees it.
func TestAnOverflowingBudgetIsTheLargest(t *testing.T) {
	if got := asTokenCount(json.RawMessage(strings.Repeat("9", 401))); got != math.MaxInt64 {
		t.Errorf("asTokenCount of a 401-digit budget = %d, want MaxInt64", got)
	}
}

// The bridge's in-process path holds the same rule as the network's.
func TestTheBridgeRefusesAValueTheGenerationThreadWouldDieOn(t *testing.T) {
	g, _, fake := askGateway(t, "ok")
	err := g.Ask(context.Background(), AskRequest{
		Model:  testModelID,
		Body:   []byte(`{"model":"` + testModelID + `","messages":[{"role":"user","content":"hi"}],"xtc_probability":0.5,"xtc_threshold":0.9}`),
		Source: stats.SourceBridge,
	})
	if err == nil {
		t.Fatal("the bridge relayed an xtc_threshold the generation thread dies on")
	}
	if fake.LastBody() != nil {
		t.Error("the refused request reached the model server")
	}
}

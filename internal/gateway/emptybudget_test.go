package gateway

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// A request that asks for no answer at all is refused before anything is
// resolved or loaded: the pinned mlx-lm 0.32.0 accepts a budget of zero and
// then fails on it, and on the batched path the failure kills the model's
// generation thread for every client (spc-2610030846273729 step 1; the
// 0.32.0 re-verification note). A budget of one or more, and a request with
// no budget, are relayed as they always were.
func TestARequestForAnEmptyAnswerIsRefused(t *testing.T) {
	const head = `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],`
	for _, c := range []struct {
		name, body string
		refused    bool
	}{
		{"max_tokens zero", head + `"max_tokens":0}`, true},
		{"max_tokens zero as a float", head + `"max_tokens":0.0}`, true},
		{"max_tokens negative", head + `"max_tokens":-5}`, true},
		{"max_tokens a fraction", head + `"max_tokens":0.5}`, true},
		{"max_tokens false", head + `"max_tokens":false}`, true},
		{"max_completion_tokens zero", head + `"max_completion_tokens":0}`, true},
		{"zero, given last", head + `"max_tokens":100,"max_tokens":0}`, true},
		{"max_tokens one", head + `"max_tokens":1}`, false},
		{"max_tokens true", head + `"max_tokens":true}`, false},
		{"no budget", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}]}`, false},
		{"a string, which the server refuses itself", head + `"max_tokens":"0"}`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, pool, fake := newTestGateway(t, config.Default())
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", strings.NewReader(c.body))
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
				if resp.StatusCode != http.StatusBadRequest || !strings.Contains(errorMessage(t, string(body)), "must be at least 1") {
					t.Errorf("status %d, %s; want a 400 saying the budget must be at least 1", resp.StatusCode, body)
				}
				if acquired != 0 || fake.LastBody() != nil {
					t.Error("a refused request reached the pool or the model server")
				}
				return
			}
			if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "must be at least 1") {
				t.Errorf("a request the server can answer was refused: %s", body)
			}
		})
	}
}

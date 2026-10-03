package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/stats"
)

// refusedFieldBodies are request bodies that carry one of the two fields the
// model server reads as an instruction to load something a client names
// (iss-2610030752128514): draft_model, which it hands to load() as a second
// model — running whatever code that model's configuration names — and
// adapters, which it loads adapter weights from and reloads the served model
// for. The presence of the key is enough, whatever its value, and the key is
// the one the model server reads: spelled with a JSON escape it is the same
// key, and given twice it is still there. The escaped keys are built from
// pieces so that no tool which writes or reads this file can decode them into
// the plain key; the test asserts that their bytes carry a backslash.
var refusedFieldBodies = []struct {
	name  string
	field string
	body  string
}{
	{"draft_model, a path", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":"/elsewhere/model"}`},
	{"draft_model, null", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":null}`},
	{"draft_model, empty", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":""}`},
	{"draft_model, a number", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":0}`},
	{"draft_model, an object", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":{}}`},
	{"draft_model, spelled with an escape", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft` + "\\" + `u005fmodel":"/elsewhere/model"}`},
	{"draft_model, given twice, the last null", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":"/elsewhere/model","draft_model":null}`},
	{"draft_model, given twice, the last a path", "draft_model", `{"draft_model":null,"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"draft_model":"/elsewhere/model"}`},
	{"draft_model, streamed", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"stream":true,"draft_model":"/elsewhere/model"}`},
	{"adapters, a path", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"adapters":"/elsewhere/adapters"}`},
	{"adapters, null", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"adapters":null}`},
	{"adapters, empty", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"adapters":""}`},
	{"adapters, a list", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"adapters":[]}`},
	{"adapters, spelled with an escape", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"` + "\\" + `u0061dapters":"/elsewhere/adapters"}`},
	{"adapters, given twice", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"adapters":"/elsewhere/adapters","adapters":null}`},
	{"adapters, streamed", "adapters", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"stream":true,"adapters":"/elsewhere/adapters"}`},
	{"both, draft_model named", "draft_model", `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"adapters":"a","draft_model":"b"}`},
}

// A request carrying draft_model or adapters is refused with 400 and a
// message naming the field, on both completion routes, streamed or not —
// before any model is acquired, so nothing is loaded or evicted for it, and
// the model server never sees the request at all.
func TestARequestNamingSomethingToLoadIsRefused(t *testing.T) {
	for _, path := range []string{"/v1/chat/completions", "/v1/completions"} {
		for _, c := range refusedFieldBodies {
			t.Run(path+" "+c.name, func(t *testing.T) {
				if strings.Contains(c.name, "escape") && !strings.Contains(c.body, `\`) {
					t.Fatalf("the body %s carries no backslash, so it tests no escape", c.body)
				}
				srv, pool, fake := newTestGateway(t, config.Default())
				req, err := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(c.body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := srv.Client().Do(req)
				if err != nil {
					t.Fatal(err)
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("status = %d, want 400", resp.StatusCode)
				}
				msg := errorMessage(t, string(body))
				if !strings.HasPrefix(msg, `"`+c.field+`" is not accepted`) {
					t.Errorf("the client was told %q, want a refusal naming %s", msg, c.field)
				}
				pool.mu.Lock()
				acquired := len(pool.acquired)
				pool.mu.Unlock()
				if acquired != 0 {
					t.Errorf("a model was acquired %d time(s) for a refused request", acquired)
				}
				if got := fake.LastBody(); got != nil {
					t.Errorf("the model server was sent %v for a refused request", got)
				}
			})
		}
	}
}

// The refusal is of the two keys the model server reads and of nothing else,
// so every other request is relayed as it always was: a field spelled in
// another case is a different key to the model server, as to the gateway; the
// same name inside a message is content; and a request with neither is the
// ordinary case. What reaches the model server carries no draft_model and no
// adapters key, which is what the model server would act on.
func TestARequestNamingNothingToLoadIsRelayed(t *testing.T) {
	for name, body := range map[string]string{
		"an ordinary request":       `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}]}`,
		"another case":              `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"Draft_Model":"x","ADAPTERS":"y"}`,
		"inside a message":          `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi","draft_model":"x","adapters":"y"}]}`,
		"the names as content only": `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"\"draft_model\": \"x\", \"adapters\": \"y\""}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			srv, pool, fake := newTestGateway(t, config.Default())
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			pool.mu.Lock()
			acquired := len(pool.acquired)
			pool.mu.Unlock()
			if acquired != 1 {
				t.Errorf("acquired %d time(s), want 1", acquired)
			}
			got := fake.LastBody()
			if got == nil {
				t.Fatal("the model server was sent nothing")
			}
			for _, key := range []string{"draft_model", "adapters"} {
				if _, ok := got[key]; ok {
					t.Errorf("the model server was sent a %q key", key)
				}
			}
		})
	}
}

// Ask is the bridge's way to a model server, and it is a relay too: what it
// is handed is refused for the same two fields, before the pool is asked for
// anything and before the model server sees a byte. The bridge builds its own
// body and carries neither field, so this is the rule held on the second
// path rather than a refusal a bridged user can meet.
func TestAskRefusesARequestNamingSomethingToLoad(t *testing.T) {
	for _, c := range refusedFieldBodies {
		t.Run(c.name, func(t *testing.T) {
			const modelPath = "/models/" + testModelID
			fake := mlxtest.Start(mlxtest.Options{ModelArg: modelPath, Reply: "ok"})
			t.Cleanup(fake.Close)
			pool := &stubPool{srv: fake}
			g := New(Options{
				Config: config.Default(),
				Pool:   pool,
				Models: &stubModels{models: []registry.Model{{
					RepoID: testModelID, Path: modelPath, State: registry.StateReady, ContextLength: 131072,
				}}},
			})
			err := g.Ask(context.Background(), AskRequest{Model: testModelID, Body: []byte(c.body), Source: stats.SourceBridge})
			var askErr *AskError
			if !errors.As(err, &askErr) {
				t.Fatalf("Ask = %v, want an AskError", err)
			}
			if !strings.HasPrefix(askErr.Error(), `"`+c.field+`" is not accepted`) {
				t.Errorf("the operator's text is %q, want a refusal naming %s", askErr.Error(), c.field)
			}
			if askErr.Public() != genericRefusal {
				t.Errorf("the text that travels is %q, want the generic refusal", askErr.Public())
			}
			pool.mu.Lock()
			acquired := len(pool.acquired)
			pool.mu.Unlock()
			if acquired != 0 {
				t.Errorf("a model was acquired %d time(s) for a refused request", acquired)
			}
			if got := fake.LastBody(); got != nil {
				t.Errorf("the model server was sent %v for a refused request", got)
			}
		})
	}
}

// The reference page states each refused field and the message it is refused
// with, word for word, so a change to either is a change to the page too.
func TestTheRefusedFieldsReferenceStatesEveryRefusal(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "docs", "request-fields.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range loadFields {
		if !strings.Contains(string(page), "| `"+f.name+"` |") {
			t.Errorf("docs/request-fields.md has no row for %s", f.name)
		}
		if !strings.Contains(string(page), "`"+f.refusal+"`") {
			t.Errorf("docs/request-fields.md does not state the refusal %q", f.refusal)
		}
	}
}

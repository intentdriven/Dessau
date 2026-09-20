package toolprobe

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
)

// fakeSources is the app as the probe sees it: one model server, resident or
// not, busy or not, and a place to keep the verdict.
type fakeSources struct {
	srv     *mlxtest.Server
	runtime string
	// modelArg, when set, is what Acquire reports instead of the server's
	// own: a load instruction the server will refuse.
	modelArg string

	mu       sync.Mutex
	loaded   bool
	inFlight int
	held     int
	acquired int
	saved    map[string]*registry.ToolCalling
	saveErr  error
}

func newSources(srv *mlxtest.Server) *fakeSources {
	return &fakeSources{srv: srv, runtime: "0.31.3", loaded: true, saved: map[string]*registry.ToolCalling{}}
}

func (s *fakeSources) Resident(string) (bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loaded, s.inFlight
}

func (s *fakeSources) Acquire(ctx context.Context, repoID string) (Upstream, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		return Upstream{}, nil, errors.New("not resident")
	}
	s.acquired++
	s.held++
	arg := s.srv.ModelArg
	if s.modelArg != "" {
		arg = s.modelArg
	}
	return Upstream{BaseURL: s.srv.URL(), ModelArg: arg}, func() {
		s.mu.Lock()
		s.held--
		s.mu.Unlock()
	}, nil
}

func (s *fakeSources) Runtime() string { return s.runtime }

func (s *fakeSources) Save(repoID string, tc *registry.ToolCalling) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved[repoID] = tc
	return nil
}

func (s *fakeSources) verdict(repoID string) *registry.ToolCalling {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saved[repoID]
}

func (s *fakeSources) counts() (acquired, held int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.acquired, s.held
}

func newProbe(t *testing.T, src *fakeSources) *Probe {
	t.Helper()
	p := New(Options{
		Sources: src, Log: slog.New(slog.DiscardHandler),
		Now:  func() time.Time { return time.Unix(1_788_696_000, 0) },
		Poll: 5 * time.Millisecond,
	})
	t.Cleanup(p.Close)
	return p
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// The verdict is what the answer carried: a tool call records can; text and
// an empty message record cannot. Each is stamped with the runtime in force
// and the clock, and the model is released afterwards.
func TestTheVerdictIsReadFromTheAnswer(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts mlxtest.Options
		can  bool
	}{
		{"a tool call", mlxtest.Options{ToolCall: true}, true},
		{"text", mlxtest.Options{Reply: "It is noon in Paris."}, false},
		{"an empty message", mlxtest.Options{EmptyMessage: true}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.ModelArg = "/models/org/m"
			srv := mlxtest.Start(tt.opts)
			t.Cleanup(srv.Close)
			src := newSources(srv)
			p := newProbe(t, src)
			got, err := p.Run(context.Background(), "org/m")
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || got.Can != tt.can || got.Runtime != "0.31.3" || got.At != 1_788_696_000 || got.Stale != "" {
				t.Errorf("verdict = %+v, want can=%v under 0.31.3 at the clock's time", got, tt.can)
			}
			if saved := src.verdict("org/m"); saved == nil || saved.Can != tt.can {
				t.Errorf("saved = %+v, want the verdict recorded", saved)
			}
			if acquired, held := src.counts(); acquired != 1 || held != 0 {
				t.Errorf("acquired %d times and still holding %d; want one acquisition, released", acquired, held)
			}
		})
	}
}

// The request is the fixed one: non-streaming, temperature 0, the small
// max_tokens, the one user message from the package constant, one tool, and
// the model field exactly as the server expects it — never the friendly name.
func TestTheRequestIsFixedAndGoesToTheModelsOwnServer(t *testing.T) {
	srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/m", ToolCall: true})
	t.Cleanup(srv.Close)
	src := newSources(srv)
	p := newProbe(t, src)
	if _, err := p.Run(context.Background(), "org/m"); err != nil {
		t.Fatal(err)
	}
	body := srv.LastBody()
	if body["model"] != "/models/org/m" {
		t.Errorf("model = %v, want the server's own ModelArg", body["model"])
	}
	if stream, _ := body["stream"].(bool); stream {
		t.Error("the probe streamed")
	}
	if temp, _ := body["temperature"].(float64); temp != 0 {
		t.Errorf("temperature = %v, want 0", body["temperature"])
	}
	if mt, _ := body["max_tokens"].(float64); mt <= 0 || mt > 512 {
		t.Errorf("max_tokens = %v, want small and set", body["max_tokens"])
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages = %v, want exactly one", msgs)
	}
	msg, _ := msgs[0].(map[string]any)
	if msg["role"] != "user" || msg["content"] != Prompt {
		t.Errorf("message = %v, want the user message %q", msg, Prompt)
	}
	tools, _ := body["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %v, want exactly one", tools)
	}
	tool, _ := tools[0].(map[string]any)
	fn, _ := tool["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)
	props, _ := params["properties"].(map[string]any)
	city, _ := props["city"].(map[string]any)
	if tool["type"] != "function" || fn["name"] != ToolName || city["type"] != "string" {
		t.Errorf("tool = %v, want function %s taking a string city", tool, ToolName)
	}
	if srv.LastAuthHeader() != "" {
		t.Errorf("the probe sent an Authorization header to the model server: %q", srv.LastAuthHeader())
	}
}

// A server that did not answer is no evidence about the model: a transport
// error, a non-200 and a timeout each record nothing, and the model stays
// unprobed rather than being written down as unable.
func TestAServerThatDidNotAnswerRecordsNothing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) *fakeSources
	}{
		{"a non-200", func(t *testing.T) *fakeSources {
			// The fake refuses a model field that is not its own with a 404,
			// which is the shape a wrong load instruction produces.
			srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/other"})
			t.Cleanup(srv.Close)
			src := newSources(srv)
			src.modelArg = "/models/org/m"
			return src
		}},
		{"a transport error", func(t *testing.T) *fakeSources {
			srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/m"})
			src := newSources(srv)
			srv.Close()
			return src
		}},
		{"a timeout", func(t *testing.T) *fakeSources {
			srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/m", ResponseDelay: 2 * time.Second})
			t.Cleanup(srv.Close)
			return newSources(srv)
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			src := tt.setup(t)
			p := New(Options{
				Sources: src, Log: slog.New(slog.DiscardHandler),
				RequestTimeout: 100 * time.Millisecond, Poll: 5 * time.Millisecond,
			})
			t.Cleanup(p.Close)
			got, err := p.Run(context.Background(), "org/m")
			if err == nil || got != nil {
				t.Errorf("Run = %+v, %v; want no verdict and an error", got, err)
			}
			if saved := src.verdict("org/m"); saved != nil {
				t.Errorf("a server that did not answer recorded %+v", saved)
			}
			if _, held := src.counts(); held != 0 {
				t.Errorf("still holding the model after a failed probe")
			}
		})
	}
}

// A queued probe waits for the request that loaded the model to be served,
// then runs; it takes one acquisition and releases it. Queued twice is
// queued once.
func TestAQueuedProbeRunsAfterTheLoadingRequestAndOnlyOnce(t *testing.T) {
	srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/m", ToolCall: true})
	t.Cleanup(srv.Close)
	src := newSources(srv)
	src.inFlight = 1
	p := newProbe(t, src)
	if !p.Enqueue("org/m") {
		t.Fatal("the first Enqueue was refused")
	}
	if p.Enqueue("Org/M") {
		t.Error("a model already queued was queued again")
	}
	time.Sleep(50 * time.Millisecond)
	if srv.Completions() != 0 {
		t.Fatal("the probe ran while the loading request was still in flight")
	}
	if got := p.Queued(); len(got) != 1 || got[0] != "org/m" {
		t.Errorf("Queued = %v, want the one model", got)
	}
	src.mu.Lock()
	src.inFlight = 0
	src.mu.Unlock()
	waitFor(t, "the verdict", func() bool { return src.verdict("org/m") != nil })
	waitFor(t, "the queue to drain", func() bool { return len(p.Queued()) == 0 })
	if srv.Completions() != 1 {
		t.Errorf("the probe sent %d requests, want exactly one", srv.Completions())
	}
	if acquired, held := src.counts(); acquired != 1 || held != 0 {
		t.Errorf("acquired %d times, holding %d; want one acquisition, released", acquired, held)
	}
	// Queued again after the fact — a stale verdict, say — it runs again and
	// overwrites.
	src.runtime = "0.32.0"
	if !p.Enqueue("org/m") {
		t.Fatal("a model with a finished probe could not be queued again")
	}
	waitFor(t, "the overwritten verdict", func() bool {
		v := src.verdict("org/m")
		return v != nil && v.Runtime == "0.32.0"
	})
	if v := src.verdict("org/m"); !v.Can || v.Stale != "" {
		t.Errorf("the re-run verdict is %+v, want a current can under the new runtime", v)
	}
}

// A model that has gone by the time the queue reaches it is dropped: no
// acquisition — which would load it — no request, and nothing recorded.
func TestAModelThatHasGoneIsDroppedNotLoaded(t *testing.T) {
	srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/m", ToolCall: true})
	t.Cleanup(srv.Close)
	src := newSources(srv)
	src.loaded = false
	p := newProbe(t, src)
	p.Enqueue("org/m")
	waitFor(t, "the queue to drain", func() bool { return len(p.Queued()) == 0 })
	if acquired, _ := src.counts(); acquired != 0 {
		t.Errorf("the probe acquired a model that was not resident, which loads it")
	}
	if srv.Completions() != 0 || src.verdict("org/m") != nil {
		t.Errorf("a dropped probe sent %d requests and recorded %+v", srv.Completions(), src.verdict("org/m"))
	}
	// Queued again once the model is back, it runs.
	src.mu.Lock()
	src.loaded = true
	src.mu.Unlock()
	p.Enqueue("org/m")
	waitFor(t, "the verdict", func() bool { return src.verdict("org/m") != nil })
}

// Close stops a queued probe where it stands and returns; nothing runs after.
func TestCloseStopsTheQueue(t *testing.T) {
	srv := mlxtest.Start(mlxtest.Options{ModelArg: "/models/org/m", ToolCall: true})
	t.Cleanup(srv.Close)
	src := newSources(srv)
	src.inFlight = 1
	p := newProbe(t, src)
	p.Enqueue("org/m")
	done := make(chan struct{})
	go func() { p.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return while a probe was waiting")
	}
	src.mu.Lock()
	src.inFlight = 0
	src.mu.Unlock()
	time.Sleep(50 * time.Millisecond)
	if srv.Completions() != 0 {
		t.Error("a probe ran after Close")
	}
}

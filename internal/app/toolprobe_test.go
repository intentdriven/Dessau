package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/toolprobe"
)

// toolLauncher stands up a fake model server that answers with a tool call,
// on the port the pool allocated, so a real pool can load and hold a model
// and the observer can fire the way it does in the shipping app.
type toolLauncher struct {
	mu      sync.Mutex
	servers map[string]*mlxtest.Server
	// hangAbove, when set, makes every server hang a prompt counted above
	// that many tokens: the readiness probe's "hi" is a handful, the tool
	// probe's question is a dozen, so a bound between them hangs the probe
	// alone.
	hangAbove int
}

type toolProcess struct {
	srv  *mlxtest.Server
	done chan struct{}
	once sync.Once
}

func (p *toolProcess) Stop(context.Context) error {
	p.once.Do(func() { p.srv.Close(); close(p.done) })
	return nil
}
func (p *toolProcess) Done() <-chan struct{} { return p.done }
func (p *toolProcess) Err() error            { return nil }
func (p *toolProcess) Pid() int              { return 4242 }

func (l *toolLauncher) Precheck(runtime.Spec) error { return nil }

func (l *toolLauncher) Launch(_ context.Context, spec runtime.Spec) (runtime.Process, error) {
	srv := mlxtest.Start(mlxtest.Options{
		ModelArg: spec.ModelPath, Port: spec.Port, ToolCall: true,
		PromptTokensFromBody: l.hangAbove > 0, HangAbove: l.hangAbove,
	})
	l.mu.Lock()
	l.servers[spec.RepoID] = srv
	l.mu.Unlock()
	return &toolProcess{srv: srv, done: make(chan struct{})}, nil
}

func (l *toolLauncher) server(repoID string) *mlxtest.Server {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.servers[repoID]
}

func newToolProbeApp(t *testing.T) (*App, *toolLauncher) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	l := &toolLauncher{servers: map[string]*mlxtest.Server{}}
	a, err := New(Options{Paths: paths, Config: config.Default(), Launcher: l})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	for _, id := range []string{"org/a", "org/b", "org/c"} {
		dir := paths.ModelDir(id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "weights.safetensors"), []byte("w"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := a.Registry.Put(registry.Model{RepoID: id, Path: dir, State: registry.StateReady, Bytes: 1 << 20}); err != nil {
			t.Fatal(err)
		}
	}
	return a, l
}

// A model reaching loaded with no current verdict queues exactly one probe;
// a model with a current verdict queues none; and the probe runs after the
// request that loaded the model has been served, against the model's own
// server, and records the verdict under the runtime in force.
func TestAModelReachingLoadedWithoutACurrentVerdictQueuesOneProbe(t *testing.T) {
	a, l := newToolProbeApp(t)
	// org/b already has a current verdict.
	if err := a.Registry.SetToolCalling("org/b", &registry.ToolCalling{Can: false, At: 1, Runtime: runtime.MLXLMVersion()}); err != nil {
		t.Fatal(err)
	}
	// org/c has one taken under another runtime, judged stale at the refresh
	// every start and save runs.
	if err := a.Registry.SetToolCalling("org/c", &registry.ToolCalling{Can: true, At: 1, Runtime: "0.30.0"}); err != nil {
		t.Fatal(err)
	}
	a.refreshStaleness()

	// The request that loads org/a, held open: the pool reports the load
	// finished to the observer while the request is still in flight.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, release, err := a.Pool.Acquire(ctx, "org/a")
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the probe to be queued", func() bool {
		q := a.ToolProbe.Queued()
		return len(q) == 1 && q[0] == "org/a"
	})
	// While the loading request is in flight the probe waits: the model's
	// server has seen the one request the pool's readiness probe made, and
	// no other.
	time.Sleep(100 * time.Millisecond)
	if got := l.server("org/a").Completions(); got != 1 {
		t.Errorf("the model server saw %d completions while the loading request was in flight, want only the readiness probe's", got)
	}
	// A second load report queues nothing more, and a model with a current
	// verdict queues nothing at all.
	a.modelLoaded("org/a")
	a.modelLoaded("org/b")
	if q := a.ToolProbe.Queued(); len(q) != 1 {
		t.Errorf("Queued = %v, want the one model", q)
	}
	// A stale verdict is re-probed.
	a.modelLoaded("org/c")
	if q := a.ToolProbe.Queued(); len(q) != 2 || q[1] != "org/c" {
		t.Errorf("Queued = %v, want org/a then the stale org/c", q)
	}

	release()
	waitFor(t, "the verdict", func() bool {
		m, _ := a.Registry.Get("org/a")
		return m.ToolCalling != nil
	})
	m, _ := a.Registry.Get("org/a")
	if !m.ToolCalling.Can || m.ToolCalling.Runtime != runtime.MLXLMVersion() || !m.ToolCalling.Current() {
		t.Errorf("ToolCalling = %+v, want a current can under %s", m.ToolCalling, runtime.MLXLMVersion())
	}
	if got := l.server("org/a").Completions(); got != 2 {
		t.Errorf("the model server saw %d completions, want the readiness probe's and the tool-call probe's", got)
	}
	// The probe's request went to the model server with its own load
	// instruction, and nothing was counted: the statistics hold no request.
	if got := l.server("org/a").LastModelField(); got != a.Paths.ModelDir("org/a") {
		t.Errorf("the probe's model field was %q, want the server's own path", got)
	}
	if body := l.server("org/a").LastBody(); body["tools"] == nil {
		t.Errorf("the probe's request carried no tools: %v", body)
	}
	if reqs := a.Stats.View().Requests; len(reqs) != 0 {
		t.Errorf("the probe was recorded as %d request(s)", len(reqs))
	}
	// And the probe released the model: nothing is in flight on it.
	waitFor(t, "the model to be released", func() bool {
		for _, r := range a.Pool.Resident() {
			if r.RepoID == "org/a" {
				return r.InFlight == 0
			}
		}
		return false
	})
	// org/c was not loaded, so its probe was dropped, not run: it stays
	// stale and unprobed until it is served.
	waitFor(t, "the queue to drain", func() bool { return len(a.ToolProbe.Queued()) == 0 })
	if c, _ := a.Registry.Get("org/c"); c.ToolCalling == nil || c.ToolCalling.Runtime != "0.30.0" {
		t.Errorf("a model that was never loaded was probed: %+v", c.ToolCalling)
	}
	if l.server("org/c") != nil {
		t.Error("the probe loaded a model that was not resident")
	}
}

// The probe's hold is the self-test's: a client whose load needs the memory
// takes the model, the pool cancels the probe's request to get it, and the
// probe lets go promptly — well under its request timeout — recording
// nothing. The model is asked again the next time it is served.
func TestAClientsLoadTakesTheModelFromUnderTheProbe(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	l := &toolLauncher{servers: map[string]*mlxtest.Server{}, hangAbove: 6}
	cfg := config.Default()
	// Room for one of the two models below and not both: each is charged
	// 1.2 times its size.
	cfg.MaxResidentBytes = 250 << 20
	a, err := New(Options{Paths: paths, Config: cfg, Launcher: l, PhysicalMemory: func() int64 { return 1 << 30 }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	for _, id := range []string{"org/a", "org/b"} {
		dir := paths.ModelDir(id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "weights.safetensors"), []byte("w"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := a.Registry.Put(registry.Model{RepoID: id, Path: dir, State: registry.StateReady, Bytes: 200 << 20}); err != nil {
			t.Fatal(err)
		}
	}

	// org/a is served once; the probe follows and hangs in its request,
	// holding the model.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, release, err := a.Pool.Acquire(ctx, "org/a")
	if err != nil {
		t.Fatal(err)
	}
	release()
	waitFor(t, "the probe to hold the model", func() bool {
		for _, r := range a.Pool.Resident() {
			if r.RepoID == "org/a" && r.InFlight == 1 {
				return true
			}
		}
		return false
	})

	// A client's load that needs the memory: served, not refused, and
	// promptly.
	arrived := time.Now()
	_, releaseB, err := a.Pool.Acquire(ctx, "org/b")
	if err != nil {
		t.Fatalf("a client's load was refused the memory the probe was holding: %v", err)
	}
	defer releaseB()
	if took := time.Since(arrived); took > 5*time.Second {
		t.Errorf("the client waited %v for the probe to let go", took)
	}
	waitFor(t, "org/a to be gone", func() bool {
		for _, r := range a.Pool.Resident() {
			if r.RepoID == "org/a" {
				return false
			}
		}
		return true
	})
	if m, _ := a.Registry.Get("org/a"); m.ToolCalling != nil {
		t.Errorf("a yielded probe recorded %+v", m.ToolCalling)
	}
	// org/a leaves the queue; org/b, held by this test, is queued behind it
	// and waits for the hold to go, which is the ordinary case.
	waitFor(t, "org/a to leave the queue", func() bool {
		for _, q := range a.ToolProbe.Queued() {
			if q == "org/a" {
				return false
			}
		}
		return true
	})
}

// The probe's acquisition holds a resident model and never becomes a load:
// a model that went between the probe's residency check and its Acquire is
// reported gone, and no model server is launched for it.
func TestTheProbesAcquireNeverLoadsAModelThatHasGone(t *testing.T) {
	a, l := newToolProbeApp(t)
	src := toolProbeSources{a}
	_, _, err := src.Acquire(context.Background(), "org/a")
	if !errors.Is(err, toolprobe.ErrGone) {
		t.Fatalf("Acquire of a model the pool is not holding = %v, want ErrGone", err)
	}
	if l.server("org/a") != nil {
		t.Error("the probe's acquisition launched a model server")
	}
}

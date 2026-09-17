package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Gropius/internal/app"
	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/mlxtest"
	"github.com/intentdriven/Gropius/internal/registry"
	"github.com/intentdriven/Gropius/internal/runtime"
)

// fakeLauncher stands a fake model server up on the port the pool chose,
// counting prompts and refusing above a size, so the whole path — the
// probe's request, the gateway, the real pool, the launch, the unload — runs
// without Python or a GPU.
type fakeLauncher struct {
	refuseAbove int
	mu          sync.Mutex
	launched    int
}

type fakeProc struct {
	srv  *mlxtest.Server
	done chan struct{}
	once sync.Once
}

func (l *fakeLauncher) Precheck(runtime.Spec) error { return nil }

func (l *fakeLauncher) Launch(_ context.Context, spec runtime.Spec) (runtime.Process, error) {
	l.mu.Lock()
	l.launched++
	l.mu.Unlock()
	srv := mlxtest.Start(mlxtest.Options{
		ModelArg: spec.ModelPath, Port: spec.Port,
		PromptTokensFromBody: true, RefuseAbove: l.refuseAbove,
	})
	return &fakeProc{srv: srv, done: make(chan struct{})}, nil
}

func (p *fakeProc) Stop(context.Context) error {
	p.once.Do(func() { p.srv.Close(); close(p.done) })
	return nil
}
func (p *fakeProc) Done() <-chan struct{} { return p.done }
func (p *fakeProc) Err() error            { return nil }
func (p *fakeProc) Pid() int              { return 0 }

// probeStack is an App with a real pool over the fake launcher, its gateway
// listening on the port the App believes is its own, and the idle loop on a
// test cadence with the probe as its job.
func probeStack(t *testing.T, refuseAbove int) (*app.App, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	paths := config.NewPaths(t.TempDir())
	cfg := config.Default()
	cfg.Port = port
	launcher := &fakeLauncher{refuseAbove: refuseAbove}
	a, err := app.New(app.Options{
		Paths: paths, Config: cfg, Launcher: launcher, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		// The loop and the probe on a test cadence.
		Idle: app.IdleOptions{
			Tick: 10 * time.Millisecond, Poll: 5 * time.Millisecond, Quiet: time.Nanosecond,
			StepTimeout: func(int) time.Duration { return 5 * time.Second },
			UnloadWait:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	// A model directory the registry's own check accepts.
	dir := paths.ModelDir("org/m")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"max_position_embeddings": 131072}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.safetensors"), []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.Registry.Put(registry.Model{RepoID: "org/m", Path: dir, State: registry.StateReady, Bytes: 7, ContextLength: 131072}); err != nil {
		t.Fatal(err)
	}

	g := New(Options{ConfigFunc: a.Config, Pool: a.Pool, Models: a.Registry, Log: a.Log})
	srv := &http.Server{Handler: g.Handler()}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return a, "http://127.0.0.1:" + strconv.Itoa(port)
}

func measurementOf(a *app.App) *registry.Measurement {
	m, _ := a.Registry.Get("org/m")
	return m.Measured
}

// The whole path: the probe's requests go through the real gateway to a
// real pool that launches, serves, and unloads a (fake) model server; the
// bisection converges on the size the server refuses, the bound is the
// model's, and the model is left unloaded.
func TestTheProbeMeasuresThroughTheRealGatewayAndPool(t *testing.T) {
	a, _ := probeStack(t, 20_000)
	if err := a.MeasureNow("org/m"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(60 * time.Second)
	for measurementOf(a) == nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	m := measurementOf(a)
	if m == nil {
		t.Fatalf("no measurement after a minute; status %+v", a.SelfTest.Status())
	}
	if m.Bound != registry.BoundModel || m.Window < 17_000 || m.Window > 20_000 {
		t.Errorf("measurement = %+v, want the model's own limit near 20,000", m)
	}
	if m.Runtime != runtime.MLXLMVersion() || m.ServedContext != 131072 {
		t.Errorf("provenance = %+v", m)
	}
	// The figure is written inside the job's Run; the runner clears the job
	// and the probe hands the model back only as Run winds down. Wait for
	// that tail under the same deadline rather than asserting on the instant
	// the figure landed.
	for (a.SelfTest.Status().Job != "" || len(a.Pool.Residency().Models) != 0) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if res := a.Pool.Residency(); len(res.Models) != 0 {
		t.Errorf("the model was left resident after the probe: %+v", res.Models)
	}
	if a.SelfTest.Status().Job != "" {
		t.Errorf("a job is still reported running: %+v", a.SelfTest.Status())
	}
}

// A client's request through the gateway during a probe: the probe yields
// (the pool's in-flight count shows the client), its own request is
// cancelled and its server unloaded once the gateway has released it, the
// client is served, and the probe resumes and completes.
func TestAClientRequestThroughTheRealGatewayYieldsTheProbeAndItResumes(t *testing.T) {
	a, base := probeStack(t, 20_000)
	on := a.Config()
	on.ContextProbe = true
	if err := a.SetConfig(on); err != nil {
		t.Fatal(err)
	}
	// Wait for the probe to be mid-run.
	deadline := time.Now().Add(20 * time.Second)
	for a.SelfTest.Status().Job == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if a.SelfTest.Status().Job == "" {
		t.Fatal("the probe never started")
	}
	// The client's request, through the gateway, on the same model.
	body, _ := json.Marshal(map[string]any{
		"model": "org/m", "messages": []map[string]string{{"role": "user", "content": "hello"}}, "max_tokens": 1,
	})
	resp, err := http.Post(base+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the client got %d: %s", resp.StatusCode, raw)
	}
	if !strings.Contains(string(raw), "GROPIUS OK") {
		t.Errorf("the client's answer: %s", raw)
	}
	// The probe resumes and finishes.
	deadline = time.Now().Add(60 * time.Second)
	for measurementOf(a) == nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	m := measurementOf(a)
	if m == nil {
		t.Fatalf("no measurement after the client left; status %+v", a.SelfTest.Status())
	}
	if m.Bound != registry.BoundModel {
		t.Errorf("bound = %q after resuming, want the model's", m.Bound)
	}
	if res := a.Pool.Residency(); len(res.Models) != 0 {
		t.Errorf("the model was left resident: %+v", res.Models)
	}
}

// A settings save during a load must not wedge: the staleness refresh judges
// outside the registry's lock, so the pool's own path through the registry
// and a save's path through the pool cannot wait on each other.
func TestASaveDuringALoadDoesNotDeadlock(t *testing.T) {
	a, _ := probeStack(t, 20_000)
	if err := a.Registry.SetMeasurement("org/m", &registry.Measurement{
		Window: 1000, Bound: registry.BoundModel, At: 1, Runtime: runtime.MLXLMVersion(),
		BudgetBytes: a.Pool.MemoryBudget(), DecodeConcurrency: a.Pool.DecodeConcurrency(), ServedContext: 131072,
	}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, release, err := a.Pool.Acquire(ctx, "org/m")
			cancel()
			if err == nil {
				release()
			}
			a.Pool.RefreshCharges()
			_ = a.Pool.Unload("org/m")
		}
	}()
	for i := 0; i < 20; i++ {
		c := a.Config()
		c.LogLevel = []string{"sparse", "detailed"}[i%2]
		if err := a.SetConfig(c); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("a save and a load wedged each other")
	}
}

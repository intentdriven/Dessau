package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/app"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// idleLauncher stands a fake model server up for each launch and keeps the
// newest, so a test can take it away underneath a model the pool still holds:
// the process stays, the socket answers nothing, and a request through the
// gateway ends unreachable — what a wedged model server looks like from the
// gateway once its client gives up.
type idleLauncher struct {
	mu  sync.Mutex
	srv *mlxtest.Server
}

func (l *idleLauncher) Precheck(runtime.Spec) error { return nil }

func (l *idleLauncher) Launch(_ context.Context, spec runtime.Spec) (runtime.Process, error) {
	srv := mlxtest.Start(mlxtest.Options{ModelArg: spec.ModelPath, Socket: spec.Socket})
	l.mu.Lock()
	l.srv = srv
	l.mu.Unlock()
	return &fakeProc{srv: srv, done: make(chan struct{})}, nil
}

func (l *idleLauncher) server() *mlxtest.Server {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.srv
}

// idleStack is an App with a real pool over the idle launcher, one ready chat
// model, the idle loop on a fast tick with a one-hour idle threshold, and a
// gateway wired to the App's client clock the way the server's own start
// wires it.
func idleStack(t *testing.T) (*app.App, *Gateway, *idleLauncher) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	launcher := &idleLauncher{}
	a, err := app.New(app.Options{
		Paths: paths, Config: config.Default(), Launcher: launcher,
		Log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Idle: app.IdleOptions{Tick: 10 * time.Millisecond, Poll: 5 * time.Millisecond, Quiet: time.Hour},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	dir := paths.ModelDir("org/m")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.safetensors"), []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A current tool-calling verdict, so the tool probe queues nothing on the
	// load and the only request this test makes is its client's.
	if err := a.Registry.Put(registry.Model{
		RepoID: "org/m", Path: dir, State: registry.StateReady, Bytes: 7, ChatTemplate: true,
		ToolCalling: &registry.ToolCalling{Can: false, At: 1, Runtime: runtime.MLXLMVersion()},
	}); err != nil {
		t.Fatal(err)
	}
	g := New(Options{ConfigFunc: a.Config, Pool: a.Pool, Models: a.Registry, Log: a.Log, Clients: a.Clients})
	return a, g, launcher
}

// selfTestOn switches the self-test on, through the save a person's tick goes
// through.
func selfTestOn(t *testing.T, a *app.App) {
	t.Helper()
	c := a.Config()
	c.SelfTest = true
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
}

// firstVerdict waits for the idle loop's first word on the due model: held,
// and by what, or a run started. It reports the status it saw.
func firstVerdict(t *testing.T, a *app.App) selftest.Status {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := a.SelfTest.Status()
		if st.HeldBy != "" || st.Model != "" {
			return st
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("the idle loop said nothing; status %+v", a.SelfTest.Status())
	return selftest.Status{}
}

func chatBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"model": "org/m", "messages": []map[string]string{{"role": "user", "content": "hello"}}, "max_tokens": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// A client request that ends unreachable resets the idle clock, and the reset
// outlasts the model it was sent to: on 2026-10-04 a client's requests to a
// wedged model each ended unreachable, the model left the pool, and the
// self-test started a minute after the last of them although the idle
// threshold was an hour (iss-2610041945030758). The clock lived on the
// model's pool entry and went with it.
func TestAClientRequestThatEndsUnreachableHoldsTheIdleJobs(t *testing.T) {
	a, g, launcher := idleStack(t)

	// The model loaded by something other than a client, then its server
	// taken away underneath the pool's entry.
	_, release, err := a.Pool.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	launcher.server().Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatBody(t)))
	req.Header.Set("Content-Type", "application/json")
	g.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("the client got %d, want 502 (the model server did not respond): %s", rec.Code, rec.Body)
	}

	// The model leaves the pool, as a wedged one does when it is unloaded
	// or its server is stopped.
	if err := a.Pool.Unload("org/m"); err != nil {
		t.Fatal(err)
	}
	if res := a.Pool.Residency(); len(res.Models) != 0 {
		t.Fatalf("the model is still resident: %+v", res.Models)
	}

	selfTestOn(t, a)
	if st := firstVerdict(t, a); st.HeldBy != selftest.HeldByRecent {
		t.Errorf("the idle loop's first word a moment after an unreachable client request: %+v; want held by %q", st, selftest.HeldByRecent)
	}
}

// A client request in flight holds the idle jobs for as long as it runs, even
// where no model shows it: here it is still uploading its body, so the pool
// has not seen it at all.
func TestAClientRequestInFlightHoldsTheIdleJobs(t *testing.T) {
	a, g, _ := idleStack(t)

	body, upload := io.Pipe()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() {
		defer close(done)
		g.Handler().ServeHTTP(httptest.NewRecorder(), req)
	}()
	t.Cleanup(func() { upload.Close(); <-done })
	if _, err := upload.Write([]byte(`{"model":`)); err != nil {
		t.Fatal(err)
	}

	selfTestOn(t, a)
	if st := firstVerdict(t, a); st.HeldBy != selftest.HeldByInFlight {
		t.Errorf("the idle loop's first word with a client request in flight: %+v; want held by %q", st, selftest.HeldByInFlight)
	}
}

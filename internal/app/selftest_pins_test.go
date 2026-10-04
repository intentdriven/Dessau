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
	"github.com/intentdriven/Dessau/internal/selftest"
)

// chatLauncher stands up a fake model server that answers chat, on the
// socket the pool allocated, so a real pool loads and holds the model the
// self-test measures.
type chatLauncher struct{}

func (chatLauncher) Precheck(runtime.Spec) error { return nil }

func (chatLauncher) Launch(_ context.Context, spec runtime.Spec) (runtime.Process, error) {
	srv := mlxtest.Start(mlxtest.Options{ModelArg: spec.ModelPath, Socket: spec.Socket})
	return &toolProcess{srv: srv, done: make(chan struct{})}, nil
}

// newSelfTestPinApp is an app with one ready chat model behind a real pool.
// The model carries a current tool-calling verdict so the tool probe queues
// nothing on its load: a probe request in flight would read as a client's,
// end the run as yielded, and the run would skip its unload whatever the pin.
func newSelfTestPinApp(t *testing.T, id string) *App {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	a, err := New(Options{Paths: paths, Config: config.Default(), Launcher: chatLauncher{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	dir := paths.ModelDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "weights.safetensors"), []byte("w"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.Registry.Put(registry.Model{
		RepoID: id, Path: dir, State: registry.StateReady, Bytes: 1 << 20, ChatTemplate: true,
		ToolCalling: &registry.ToolCalling{Can: false, At: 1, Runtime: runtime.MLXLMVersion()},
	}); err != nil {
		t.Fatal(err)
	}
	return a
}

// pin saves the model pinned, through the settings save a person's pin goes
// through, spelt differently from the registry so the fold is exercised.
func pin(t *testing.T, a *App, id string) {
	t.Helper()
	c := a.Config()
	c.Models = map[string]config.ModelSettings{id: {Pinned: true}}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
}

// pinningServer is the app's own self-test adapter with one addition: the
// first time the self-test's load returns, the model is pinned, as a person
// pinning it on the panel while the run is in progress would.
type pinningServer struct {
	selfTestServer
	once  sync.Once
	pinIt func()
}

func (s *pinningServer) Acquire(ctx context.Context, repoID string) (selftest.Upstream, func(), error) {
	up, release, err := s.selfTestServer.Acquire(ctx, repoID)
	if err == nil {
		s.once.Do(s.pinIt)
	}
	return up, release, err
}

// runOnce runs the self-test loop against srv until it has written one run,
// then stops it. The run's result is written after its unload, so a run on
// file is a run whose unload has been decided.
func runOnce(t *testing.T, srv selftest.Server) selftest.Run {
	t.Helper()
	path := filepath.Join(t.TempDir(), selftest.FileName)
	r := selftest.New(selftest.Options{
		Server: srv, Path: path,
		Tick: 10 * time.Millisecond, Poll: 5 * time.Millisecond, Quiet: time.Millisecond,
	})
	r.SetEnabled(true)
	var runs []selftest.Run
	waitFor(t, "the self-test to write its run", func() bool {
		runs, _ = selftest.ReadResults(path)
		return len(runs) > 0
	})
	r.Close()
	return runs[0]
}

func residentIn(a *App, id string) bool {
	key := config.FoldRepoID(id)
	for _, m := range a.Pool.Residency().Models {
		if config.FoldRepoID(m.RepoID) == key {
			return true
		}
	}
	return false
}

// A model the self-test loaded and a person pinned while the run was in
// progress is still loaded, and still pinned, when the run ends: the pin
// promises the model stays in memory, and the self-test's tidy-up is not the
// person's unload (iss-2610032241098944).
func TestTheSelfTestLeavesAModelPinnedDuringItsRunLoaded(t *testing.T) {
	a := newSelfTestPinApp(t, "org/m")
	srv := &pinningServer{selfTestServer: selfTestServer{a}, pinIt: func() { pin(t, a, "ORG/M") }}
	run := runOnce(t, srv)
	if !run.ColdLoad {
		t.Fatalf("the run found the model resident; the case needs the self-test's own load")
	}
	if !a.isPinned("org/m") {
		t.Error("the model is no longer pinned after the run")
	}
	if !residentIn(a, "org/m") {
		t.Errorf("the self-test unloaded a model pinned during its run (outcome %q)", run.Outcome)
	}
	// The refusal is the self-test's own, so the loop can tell it from a
	// failed unload, and it changes nothing.
	if err := (selfTestServer{a}).Unload("org/m"); !errors.Is(err, selftest.ErrPinned) {
		t.Errorf("the self-test's unload of a pinned model = %v, want ErrPinned", err)
	}
	if !residentIn(a, "org/m") {
		t.Error("a refused unload took the model out")
	}
}

// A model pinned before the run and already resident is left resident and
// pinned too: the self-test leaves residency as it found it.
func TestTheSelfTestLeavesAModelPinnedBeforeItsRunLoaded(t *testing.T) {
	a := newSelfTestPinApp(t, "org/m")
	pin(t, a, "ORG/M")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, release, err := a.Pool.Acquire(ctx, "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	run := runOnce(t, selfTestServer{a})
	if run.ColdLoad {
		t.Fatalf("the run loaded the model itself; the case needs it resident beforehand")
	}
	if !a.isPinned("org/m") {
		t.Error("the model is no longer pinned after the run")
	}
	if !residentIn(a, "org/m") {
		t.Errorf("the self-test unloaded a pinned model that was resident before its run (outcome %q)", run.Outcome)
	}
}

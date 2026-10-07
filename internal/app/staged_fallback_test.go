package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// cannotLoad marks a version whose files pass every check made before a
// start — the directory checks and the launcher's Precheck — and whose model
// server still exits during startup, as mlx-lm's does for a model it cannot
// load.
const cannotLoad = `"dessau_test_cannot_load":true`

func brokenV2Config() []byte {
	return []byte(`{"model_type":"qwen3","max_position_embeddings":40961,` + cannotLoad + `}`)
}

// loadFailLauncher starts every model as precheckLauncher does, except one
// whose config.json carries cannotLoad: its server has exited, with an exit
// status, by the time the pool first asks it anything.
type loadFailLauncher struct{ precheckLauncher }

func (l *loadFailLauncher) Launch(ctx context.Context, spec runtime.Spec) (runtime.Process, error) {
	if b, _ := os.ReadFile(filepath.Join(spec.ModelPath, "config.json")); strings.Contains(string(b), cannotLoad) {
		done := make(chan struct{})
		close(done)
		return exitedProcess{done}, nil
	}
	return l.precheckLauncher.Launch(ctx, spec)
}

type exitedProcess struct{ done chan struct{} }

func (p exitedProcess) Stop(context.Context) error { return nil }
func (p exitedProcess) Done() <-chan struct{}      { return p.done }
func (p exitedProcess) Err() error                 { return errors.New("exit status 1") }
func (p exitedProcess) Pid() int                   { return 4243 }

// newFallbackApp is newStagedApp over paths, with a launcher that fails a
// version marked cannotLoad. The caller closes the app.
func newFallbackApp(t *testing.T, paths config.Paths, h *versionedHub) *App {
	t.Helper()
	l := &loadFailLauncher{precheckLauncher{toolLauncher{servers: map[string]*mlxtest.Server{}}}}
	a, err := New(Options{Paths: paths, Config: config.Default(), Launcher: l})
	if err != nil {
		t.Fatal(err)
	}
	a.Hub.BaseURL = h.srv.URL
	return a
}

// updateToV2 downloads v1 of org/repo, then updates it to v2 as the hub
// serves it, and waits for the swap to finish.
func updateToV2(t *testing.T, a *App, h *versionedHub) {
	t.Helper()
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "v1 to be ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready() && m.Commit == commitV1
	})
	h.set(func(h *versionedHub) { h.current = commitV2 })
	a.Registry.SetUpdate("org/repo", commitV1, registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: commitV2, CheckedAt: time.Now()})
	if err := a.Update("org/repo"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); !m.Ready() || m.Commit != commitV2 {
		t.Fatalf("the new version was not swapped in: state %s, commit %s", m.State, m.Commit)
	}
}

// asidesOf lists the copies of org/repo a swap holds aside.
func asidesOf(a *App) []string {
	entries, _ := os.ReadDir(filepath.Join(a.stagingRoot(), "org"))
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "repo"+asideSuffix) {
			out = append(out, e.Name())
		}
	}
	return out
}

// servable reports whether a request for org/repo is answered: the model
// loads, and is released again.
func servable(a *App) error {
	_, release, err := a.Pool.Acquire(context.Background(), "org/repo")
	if err != nil {
		return err
	}
	release()
	return nil
}

// A new version that passes every check made before a start, and then fails
// its first load, is not left as the only copy: the version it replaced is
// put back, recorded as it was, and serves the next request; the card says
// why the update failed and that the newer version is still there to try
// (iss-2610042101430192).
func TestANewVersionThatFailsItsFirstLoadPutsTheOldOneBack(t *testing.T) {
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files()}, commitV1)
	h.set(func(h *versionedHub) { h.versions[commitV2]["config.json"] = brokenV2Config() })
	a := newFallbackApp(t, config.NewPaths(t.TempDir()), h)
	t.Cleanup(func() { a.Close() })
	updateToV2(t, a, h)

	if err := servable(a); err == nil {
		t.Fatal("the broken new version loaded")
	}
	waitFor(t, "the old version to be put back", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Commit == commitV1
	})
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v1Files()) {
		t.Errorf("the model's folder holds %v, want the old version back", got)
	}
	m, _ := a.Registry.Get("org/repo")
	if !m.Ready() || m.LoadFailed() || len(m.FileHashes) != len(v1Files()) {
		t.Errorf("the record put back: state %s, load failure %+v, %d hashes", m.State, m.LoadFailure, len(m.FileHashes))
	}
	if m.UpdateFailed != registry.UpdateFailedLoad {
		t.Errorf("UpdateFailed = %q, want %q", m.UpdateFailed, registry.UpdateFailedLoad)
	}
	if m.Update == nil || m.Update.Status != registry.UpdateAvailable || m.Update.Commit != commitV2 {
		t.Errorf("Update = %+v, want the newer version still offered", m.Update)
	}
	if err := servable(a); err != nil {
		t.Errorf("the old version put back is not served: %v", err)
	}
	if got := asidesOf(a); len(got) != 0 {
		t.Errorf("left aside after the old version went back: %v", got)
	}
	if entries, _ := os.ReadDir(filepath.Join(a.stagingRoot(), "org")); len(entries) != 0 {
		t.Errorf("the failed new version was left in the staging folder: %d entries", len(entries))
	}
}

// The old copy is held aside until the new version has loaded and answered
// once, and goes then.
func TestTheOldCopyGoesOnceTheNewVersionHasLoaded(t *testing.T) {
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files()}, commitV1)
	a := newFallbackApp(t, config.NewPaths(t.TempDir()), h)
	t.Cleanup(func() { a.Close() })
	updateToV2(t, a, h)

	if got := asidesOf(a); len(got) != 1 {
		t.Fatalf("held aside before the new version loaded: %v, want the old copy", got)
	}
	if err := servable(a); err != nil {
		t.Fatalf("the new version did not load: %v", err)
	}
	waitFor(t, "the old copy to go", func() bool { return len(asidesOf(a)) == 0 })
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v2Files()) {
		t.Errorf("the model's folder holds %v, want the new version", got)
	}
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV2 || m.UpdateFailed != "" {
		t.Errorf("record after the first load: commit %s, update failed %q", m.Commit, m.UpdateFailed)
	}
}

// A restart between the swap and the first load keeps the old copy aside,
// and a first load that then fails still puts it back. The version it was is
// not kept across the restart, so it comes back as a version Dessau did not
// record, which Update brings to the current one.
func TestTheOldCopyIsKeptAcrossARestartUntilTheFirstLoad(t *testing.T) {
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files()}, commitV1)
	h.set(func(h *versionedHub) { h.versions[commitV2]["config.json"] = brokenV2Config() })
	paths := config.NewPaths(t.TempDir())
	first := newFallbackApp(t, paths, h)
	updateToV2(t, first, h)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	a := newFallbackApp(t, paths, h)
	t.Cleanup(func() { a.Close() })
	if got := asidesOf(a); len(got) != 1 {
		t.Fatalf("held aside after the restart: %v, want the old copy", got)
	}
	if err := servable(a); err == nil {
		t.Fatal("the broken new version loaded")
	}
	waitFor(t, "the old version to be put back", func() bool {
		return sameFiles(filesIn(t, a.Paths.ModelDir("org/repo")), v1Files())
	})
	waitFor(t, "the record to describe it", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.UpdateFailed == registry.UpdateFailedLoad
	})
	if m, _ := a.Registry.Get("org/repo"); m.Commit != "" || !m.Ready() {
		t.Errorf("record put back after a restart: commit %q, state %s; want ready, version unknown", m.Commit, m.State)
	}
	if err := servable(a); err != nil {
		t.Errorf("the old version put back is not served: %v", err)
	}
}

// Deleting a model removes the old copy its update holds aside too, so the
// next start does not bring the model back from it.
func TestDeletingAModelRemovesTheCopyHeldAside(t *testing.T) {
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files()}, commitV1)
	paths := config.NewPaths(t.TempDir())
	first := newFallbackApp(t, paths, h)
	updateToV2(t, first, h)
	if err := first.Delete("org/repo"); err != nil {
		t.Fatal(err)
	}
	if got := asidesOf(first); len(got) != 0 {
		t.Errorf("held aside after the model was deleted: %v", got)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	a := newFallbackApp(t, paths, h)
	t.Cleanup(func() { a.Close() })
	if _, err := a.Registry.Get("org/repo"); err == nil {
		t.Error("a deleted model came back at the next start")
	}
}

// A second update before the first one's version has loaded keeps the copy
// the first update held: that is the last version known to have served, and
// the version the second update replaces never loaded. So when the newest
// version fails its first load too, the version that served comes back.
func TestASecondUpdateBeforeAnyLoadKeepsTheVersionThatServed(t *testing.T) {
	const commitV3 = "3333333333333333333333333333333333333333"
	v3 := map[string][]byte{
		"config.json":       []byte(`{"model_type":"qwen3","max_position_embeddings":40962,` + cannotLoad + `}`),
		"model.safetensors": []byte("weights-all-v3"),
		"tokenizer.json":    []byte(`{"v":1}`),
	}
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files(), commitV3: v3}, commitV1)
	h.set(func(h *versionedHub) { h.versions[commitV2]["config.json"] = brokenV2Config() })
	a := newFallbackApp(t, config.NewPaths(t.TempDir()), h)
	t.Cleanup(func() { a.Close() })
	updateToV2(t, a, h)

	h.set(func(h *versionedHub) { h.current = commitV3 })
	a.Registry.SetUpdate("org/repo", commitV2, registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: commitV3, CheckedAt: time.Now()})
	if err := a.Update("org/repo"); err != nil {
		t.Fatalf("second Update: %v", err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV3 {
		t.Fatalf("the second update was not swapped in: commit %s", m.Commit)
	}
	// The second update's own aside copy is removed just after the download
	// mark is cleared, outside the lock, so wait for it rather than look once.
	waitFor(t, "one copy to be held after two updates", func() bool { return len(asidesOf(a)) == 1 })

	if err := servable(a); err == nil {
		t.Fatal("the broken newest version loaded")
	}
	waitFor(t, "the version that served to be put back", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Commit == commitV1
	})
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v1Files()) {
		t.Errorf("the model's folder holds %v, want the version that served", got)
	}
	if err := servable(a); err != nil {
		t.Errorf("the version put back is not served: %v", err)
	}
}

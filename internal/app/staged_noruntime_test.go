package app

import (
	"fmt"
	"os"
	"testing"

	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// noRuntime is a launcher on a Mac where the runtime is not installed yet: its
// Precheck says so, and it can check a model's own files without it.
type noRuntime struct{ toolLauncher }

func (l *noRuntime) Precheck(runtime.Spec) error {
	return fmt.Errorf("python runtime is not installed: %w", runtime.ErrRuntimeNotInstalled)
}

func (l *noRuntime) PrecheckModel(spec runtime.Spec) error {
	return registry.CheckModelCode(spec.ModelPath, os.Geteuid())
}

// noRuntimeNoModelCheck says the runtime is missing and cannot check a model
// on its own.
type noRuntimeNoModelCheck struct{ toolLauncher }

func (l *noRuntimeNoModelCheck) Precheck(runtime.Spec) error {
	return fmt.Errorf("python runtime is not installed: %w", runtime.ErrRuntimeNotInstalled)
}

// Without the runtime, a staged version is held to the model half of
// Precheck: a clean version is swapped in, and one that ships its own code is
// still refused, leaving the old version serving. A launcher that cannot check
// a model on its own refuses as before (iss-2610031317475284).
func TestWithoutTheRuntimeAStagedVersionIsHeldToItsModelCheck(t *testing.T) {
	for name, c := range map[string]struct {
		launcher   runtime.Launcher
		shipsCode  bool
		wantCommit string
	}{
		"clean version, model check":           {&noRuntime{toolLauncher{servers: map[string]*mlxtest.Server{}}}, false, commitV2},
		"version that ships code, model check": {&noRuntime{toolLauncher{servers: map[string]*mlxtest.Server{}}}, true, commitV1},
		"no model check to fall back to":       {&noRuntimeNoModelCheck{toolLauncher{servers: map[string]*mlxtest.Server{}}}, false, commitV1},
	} {
		t.Run(name, func(t *testing.T) {
			a, h := newStagedApp(t)
			a.launcher = c.launcher
			h.set(func(h *versionedHub) {
				h.current = commitV2
				if c.shipsCode {
					h.versions[commitV2]["config.json"] = []byte(`{"model_type":"qwen3","model_file":"x.py"}`)
				}
			})
			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)
			if m, _ := a.Registry.Get("org/repo"); !m.Ready() || m.Commit != c.wantCommit {
				t.Errorf("state %s, commit %s; want ready at %s", m.State, m.Commit, c.wantCommit)
			}
		})
	}
}

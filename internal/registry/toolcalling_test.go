package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func canCall() *ToolCalling {
	return &ToolCalling{Can: true, At: 1_788_696_000, Runtime: "0.31.3"}
}

// A verdict is recorded beside the measurement, persisted, and read back
// with the runtime it was taken under.
func TestAToolCallVerdictIsRecordedAndReadBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetToolCalling("org/m", canCall()); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/m")
	if m.ToolCalling == nil || !m.ToolCalling.Can || m.ToolCalling.Runtime != "0.31.3" || !m.ToolCalling.Current() {
		t.Errorf("ToolCalling = %+v, want a current can-call verdict under 0.31.3", m.ToolCalling)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"tool_calling"`) {
		t.Errorf("the file does not carry the verdict:\n%s", raw)
	}
	// Reopened, it is still there.
	r2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r2.Get("org/m"); m.ToolCalling == nil || !m.ToolCalling.Can {
		t.Errorf("the verdict did not survive a reopen: %+v", m.ToolCalling)
	}
	// A nil verdict clears, and a model the registry does not hold is an error.
	if err := r.SetToolCalling("org/m", nil); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.ToolCalling != nil {
		t.Errorf("a nil verdict did not clear: %+v", m.ToolCalling)
	}
	if err := r.SetToolCalling("org/absent", canCall()); err == nil {
		t.Error("a verdict for a model the registry does not hold was accepted")
	}
}

// A verdict is a fact about one runtime: recorded under one runtime string,
// it is not current under another, and the file says why. Back in force,
// it is current again.
func TestAToolCallVerdictRecordedUnderOneRuntimeIsNotPublishedUnderAnother(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetToolCalling("org/m", canCall()); err != nil {
		t.Fatal(err)
	}
	if got := canCall().StaleAgainst("0.32.0"); got != StaleRuntime {
		t.Errorf("StaleAgainst another runtime = %q, want %q", got, StaleRuntime)
	}
	if got := canCall().StaleAgainst("0.31.3"); got != "" {
		t.Errorf("StaleAgainst the same runtime = %q, want current", got)
	}
	moved := inForce()
	moved.Runtime = "0.32.0"
	if marked := r.RefreshStaleness(func(Model) Provenance { return moved }); len(marked) != 1 || marked[0] != "org/m" {
		t.Fatalf("RefreshStaleness marked %v, want org/m", marked)
	}
	m, _ := r.Get("org/m")
	if m.ToolCalling == nil || m.ToolCalling.Current() || m.ToolCalling.Stale != StaleRuntime {
		t.Errorf("ToolCalling = %+v, want stale for the runtime", m.ToolCalling)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"stale": "runtime"`) && !strings.Contains(string(raw), `"stale":"runtime"`) {
		t.Errorf("the file does not record why the verdict is stale:\n%s", raw)
	}
	if marked := r.RefreshStaleness(func(Model) Provenance { return inForce() }); len(marked) != 1 {
		t.Errorf("a runtime back in force marked %v, want the one model whose verdict changed", marked)
	}
	if m, _ := r.Get("org/m"); !m.ToolCalling.Current() {
		t.Errorf("ToolCalling = %+v after the runtime came back, want current", m.ToolCalling)
	}
	// A fresh verdict written under the runtime in force is current at once.
	if err := r.SetToolCalling("org/m", &ToolCalling{Can: false, At: 2, Runtime: "0.31.3"}); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); !m.ToolCalling.Current() || m.ToolCalling.Can {
		t.Errorf("a fresh verdict reads %+v, want a current cannot", m.ToolCalling)
	}
}

// A re-download writes a fresh record through Put, and a fresh record has no
// verdict: it belonged to the files that were replaced, by the rule that
// already clears the measurement.
func TestARedownloadClearsTheToolCallVerdict(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetToolCalling("org/m", canCall()); err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 2, AddedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.ToolCalling != nil {
		t.Errorf("a re-downloaded model kept its verdict: %+v", m.ToolCalling)
	}
}

// registry.json is, in shared-cache mode, a file another local account can
// write. A verdict read back is bounded exactly as one written by the probe
// is, and an implausible one is cleared rather than repaired.
func TestAPlantedToolCallVerdictIsClearedOnLoad(t *testing.T) {
	for _, tt := range []struct {
		name string
		json string
	}{
		{"a runtime string past any version", `{"can": true, "at": 1, "runtime": "` + strings.Repeat("9", 200) + `"}`},
		{"a negative time", `{"can": true, "at": -1, "runtime": "0.31.3"}`},
		{"a time past the ceiling", `{"can": true, "at": 1099511627776, "runtime": "0.31.3"}`},
		{"a stale reason outside the set", `{"can": true, "at": 1, "runtime": "0.31.3", "stale": "vibes"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "registry.json")
			file := `[{"repo_id": "org/m", "path": "` + dir + `", "state": "ready", "bytes": 1, "tool_calling": ` + tt.json + `}]`
			if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
				t.Fatal(err)
			}
			r, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			m, err := r.Get("org/m")
			if err != nil {
				t.Fatalf("the planted file did not load its model: %v", err)
			}
			if m.ToolCalling != nil {
				t.Errorf("%s survived the load: %+v", tt.name, m.ToolCalling)
			}
		})
	}
	// And a good one survives, stale mark included.
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	file := `[{"repo_id": "org/m", "path": "` + dir + `", "state": "ready", "bytes": 1, "tool_calling": {"can": false, "at": 1, "runtime": "0.31.3", "stale": "runtime"}}]`
	if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.ToolCalling == nil || m.ToolCalling.Can || m.ToolCalling.Stale != StaleRuntime {
		t.Errorf("a plausible verdict did not survive the load: %+v", m.ToolCalling)
	}
	// SetToolCalling refuses the same shapes rather than writing them.
	if err := r.SetToolCalling("org/m", &ToolCalling{Can: true, At: 1, Runtime: strings.Repeat("9", 200)}); err == nil {
		t.Error("an implausible verdict was written")
	}
}

package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func failed() *LoadFailure {
	return &LoadFailure{
		Reason:            "ValueError: Model type glm_ocr not supported.",
		At:                1_788_696_000,
		Runtime:           "0.31.3",
		BudgetBytes:       96 << 30,
		DecodeConcurrency: 4,
		ServedContext:     131_072,
	}
}

// A load failure is recorded on the model with the provenance it happened
// under, and it stands — no idle job picks the model, a request for it is
// told the reason — until the runtime, the budget, the concurrency or the
// served window moves, when it is lifted rather than kept as stale: a
// failure under another provenance says nothing about this one
// (iss-2609211334570516).
func TestALoadFailureStandsUntilItsProvenanceMoves(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1, ContextLength: 131_072}); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.LoadFailed() {
		t.Fatal("a fresh model reads as failed")
	}
	if err := r.SetLoadFailure("org/m", failed()); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/m")
	if !m.LoadFailed() || m.LoadFailure.Reason != failed().Reason {
		t.Fatalf("LoadFailure = %+v, want the recorded one", m.LoadFailure)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "glm_ocr not supported") {
		t.Errorf("the file does not carry the reason:\n%s", raw)
	}
	// The provenance unchanged: the failure stands, and nothing is reported.
	if marked := r.RefreshStaleness(func(Model) Provenance { return inForce() }); len(marked) != 0 {
		t.Errorf("an unchanged provenance changed %v", marked)
	}
	if m, _ := r.Get("org/m"); !m.LoadFailed() {
		t.Error("an unchanged provenance lifted the failure")
	}
	for _, tt := range []struct {
		name string
		move func(*Provenance)
	}{
		{"runtime", func(p *Provenance) { p.Runtime = "0.32.0" }},
		{"budget", func(p *Provenance) { p.BudgetBytes = 64 << 30 }},
		{"concurrency", func(p *Provenance) { p.DecodeConcurrency = 2 }},
		{"served window", func(p *Provenance) { p.ServedContext = 65_536 }},
	} {
		if err := r.SetLoadFailure("org/m", failed()); err != nil {
			t.Fatal(err)
		}
		p := inForce()
		tt.move(&p)
		if marked := r.RefreshStaleness(func(Model) Provenance { return p }); len(marked) != 1 || marked[0] != "org/m" {
			t.Errorf("%s moved: RefreshStaleness reported %v, want org/m", tt.name, marked)
		}
		if m, _ := r.Get("org/m"); m.LoadFailed() {
			t.Errorf("%s moved and the failure still stands: %+v", tt.name, m.LoadFailure)
		}
	}
	// Cleared by hand, and by a re-download's fresh record.
	if err := r.SetLoadFailure("org/m", failed()); err != nil {
		t.Fatal(err)
	}
	if err := r.SetLoadFailure("org/m", nil); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.LoadFailed() {
		t.Error("a nil failure did not clear the mark")
	}
	if err := r.SetLoadFailure("org/m", failed()); err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 2, ContextLength: 131_072, AddedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.LoadFailed() {
		t.Error("a re-downloaded model kept its load failure")
	}
	if err := r.SetLoadFailure("org/absent", failed()); err == nil {
		t.Error("a failure was recorded on a model the registry does not hold")
	}
}

// registry.json is, in shared-cache mode, a file another local account can
// write, and the reason is shown on the card and told to an entitled client.
// A failure read back is bounded exactly as one the pool wrote, and an
// implausible one is cleared rather than repaired.
func TestAPlantedLoadFailureIsClearedOnLoad(t *testing.T) {
	for _, tt := range []struct {
		name string
		json string
	}{
		{"a reason past the bound", `{"reason": "` + strings.Repeat("x", 2000) + `", "at": 1, "runtime": "0.31.3"}`},
		{"no reason at all", `{"reason": "", "at": 1, "runtime": "0.31.3"}`},
		{"a reason with a path in it", `{"reason": "ValueError: /Users/alice/x", "at": 1, "runtime": "0.31.3"}`},
		{"a runtime string past any version", `{"reason": "ValueError: x", "at": 1, "runtime": "` + strings.Repeat("9", 200) + `"}`},
		{"a time before the epoch", `{"reason": "ValueError: x", "at": -1, "runtime": "0.31.3"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "registry.json")
			file := `[{"repo_id": "org/m", "path": "` + dir + `", "state": "ready", "bytes": 1, "context_length": 131072, "load_failure": ` + tt.json + `}]`
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
			if m.LoadFailure != nil {
				t.Errorf("%s survived the load: %+v", tt.name, m.LoadFailure)
			}
		})
	}
	if err := (&Registry{models: map[string]Model{"org/m": {RepoID: "org/m"}}}).SetLoadFailure("org/m", &LoadFailure{Reason: strings.Repeat("x", 2000)}); err == nil {
		t.Error("an implausible failure was accepted from the pool's side too")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	file := `[{"repo_id": "org/m", "path": "` + dir + `", "state": "ready", "bytes": 1, "context_length": 131072, "load_failure": {"reason": "ValueError: Model type glm_ocr not supported.", "at": 1, "runtime": "0.31.3", "budget_bytes": 1, "decode_concurrency": 4, "served_context": 131072}}]`
	if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); !m.LoadFailed() {
		t.Errorf("a plausible failure did not survive the load: %+v", m.LoadFailure)
	}
}

// A transient failure — the pool's own bound, the readiness timeout or an
// exit by signal, not the child's verdict — stands for the process that
// wrote it and is dropped at the next start: a slow load under memory
// pressure must not keep a good model refused across restarts.
func TestATransientLoadFailureDoesNotOutliveTheProcess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1, ContextLength: 131_072}); err != nil {
		t.Fatal(err)
	}
	f := failed()
	f.Reason, f.Transient = "did not become ready within 10m0s", true
	if err := r.SetLoadFailure("org/m", f); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); !m.LoadFailed() {
		t.Fatal("a transient failure does not stand in the process that recorded it")
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := again.Get("org/m"); m.LoadFailed() {
		t.Errorf("a transient failure survived a restart: %+v", m.LoadFailure)
	}
	if err := r.SetLoadFailure("org/m", failed()); err != nil {
		t.Fatal(err)
	}
	if again, err = Open(path); err != nil {
		t.Fatal(err)
	}
	if m, _ := again.Get("org/m"); !m.LoadFailed() {
		t.Error("the child's own verdict did not survive a restart")
	}
}

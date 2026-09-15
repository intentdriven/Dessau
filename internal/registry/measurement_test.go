package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func measured() *Measurement {
	return &Measurement{
		Window:            91_000,
		Bound:             BoundModel,
		At:                1_788_696_000,
		Runtime:           "0.31.3",
		BudgetBytes:       96 << 30,
		DecodeConcurrency: 4,
		ServedContext:     131_072,
	}
}

func inForce() Provenance {
	return Provenance{Runtime: "0.31.3", BudgetBytes: 96 << 30, DecodeConcurrency: 4, ServedContext: 131_072}
}

// A measurement is a fact about one runtime, one budget, one concurrency and
// one served window. Move any of them and the figure is stale, and the
// record says which moved.
func TestAMeasurementGoesStaleWhenItsProvenanceMoves(t *testing.T) {
	if reason := measured().StaleAgainst(inForce()); reason != "" {
		t.Fatalf("an unchanged provenance made the measurement stale: %q", reason)
	}
	for _, tt := range []struct {
		name string
		move func(*Provenance)
		want string
	}{
		{"runtime", func(p *Provenance) { p.Runtime = "0.32.0" }, StaleRuntime},
		{"budget", func(p *Provenance) { p.BudgetBytes = 64 << 30 }, StaleBudget},
		{"concurrency", func(p *Provenance) { p.DecodeConcurrency = 2 }, StaleConcurrency},
		{"served window", func(p *Provenance) { p.ServedContext = 65_536 }, StaleServedContext},
	} {
		p := inForce()
		tt.move(&p)
		if got := measured().StaleAgainst(p); got != tt.want {
			t.Errorf("%s moved: stale = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// Staleness is a stored fact, not an inference: at start the provenance is
// compared with what is in force and the verdict is written to the file, so a
// reader of registry.json learns it.
func TestStalenessIsRecordedAtLoad(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1, ContextLength: 131_072}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetMeasurement("org/m", measured()); err != nil {
		t.Fatal(err)
	}
	moved := inForce()
	moved.Runtime = "0.32.0"
	if marked := r.RefreshStaleness(func(Model) Provenance { return moved }); len(marked) != 1 || marked[0] != "org/m" {
		t.Fatalf("RefreshStaleness marked %v, want org/m", marked)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"stale": "runtime"`) && !strings.Contains(string(raw), `"stale":"runtime"`) {
		t.Errorf("the file does not record why the measurement is stale:\n%s", raw)
	}
	m, _ := r.Get("org/m")
	if m.Measured == nil || m.Measured.Stale != StaleRuntime {
		t.Errorf("Measured = %+v, want stale for the runtime", m.Measured)
	}
	// Back in force: current again, and that change of verdict is reported
	// too; a second refresh with nothing moved reports nothing.
	if marked := r.RefreshStaleness(func(Model) Provenance { return inForce() }); len(marked) != 1 {
		t.Errorf("a provenance back in force marked %v, want the one model whose verdict changed", marked)
	}
	if marked := r.RefreshStaleness(func(Model) Provenance { return inForce() }); len(marked) != 0 {
		t.Errorf("an unchanged provenance marked %v", marked)
	}
	if m, _ := r.Get("org/m"); m.Measured.Stale != "" {
		t.Errorf("Measured.Stale = %q after the provenance came back", m.Measured.Stale)
	}
}

// A re-download writes a fresh record through Put, and a fresh record has no
// measurement: the figure belonged to the files that were replaced.
func TestARescanClearsAMeasurementForARedownloadedModel(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 1, ContextLength: 131_072}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetMeasurement("org/m", measured()); err != nil {
		t.Fatal(err)
	}
	if err := r.SetProbeIncomplete("org/m", true); err != nil {
		t.Fatal(err)
	}
	// The download's own Put: the shape finishDownload writes.
	if err := r.Put(Model{RepoID: "org/m", Path: dir, State: StateReady, Bytes: 2, ContextLength: 131_072, AddedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/m")
	if m.Measured != nil || m.ProbeIncomplete {
		t.Errorf("a re-downloaded model kept its measurement: %+v incomplete=%v", m.Measured, m.ProbeIncomplete)
	}
}

// registry.json is, in shared-cache mode, a file another local account can
// write. A measurement read back is bounded exactly as one written by the
// probe is, and an implausible one is cleared rather than repaired.
func TestAPlantedMeasurementIsClearedOnLoad(t *testing.T) {
	for _, tt := range []struct {
		name string
		json string
	}{
		{"a window past the bound", `{"window": 9999999999, "bound": "model", "at": 1, "runtime": "0.31.3"}`},
		{"a window of nothing", `{"window": 0, "bound": "model", "at": 1, "runtime": "0.31.3"}`},
		{"a bound outside the set", `{"window": 1000, "bound": "wishful", "at": 1, "runtime": "0.31.3"}`},
		{"a stale reason outside the set", `{"window": 1000, "bound": "model", "at": 1, "runtime": "0.31.3", "stale": "vibes"}`},
		{"a runtime string past any version", `{"window": 1000, "bound": "model", "at": 1, "runtime": "` + strings.Repeat("9", 200) + `"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "registry.json")
			file := `[{"repo_id": "org/m", "path": "` + dir + `", "state": "ready", "bytes": 1, "context_length": 131072, "measured": ` + tt.json + `}]`
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
			if m.Measured != nil {
				t.Errorf("%s survived the load: %+v", tt.name, m.Measured)
			}
		})
	}
	// And a good one survives.
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	file := `[{"repo_id": "org/m", "path": "` + dir + `", "state": "ready", "bytes": 1, "context_length": 131072, "measured": {"window": 91000, "bound": "prefill_deadline", "at": 1, "runtime": "0.31.3", "budget_bytes": 1, "decode_concurrency": 4, "served_context": 131072}}]`
	if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.Measured == nil || m.Measured.Window != 91000 || m.Measured.Bound != BoundPrefillDeadline {
		t.Errorf("a plausible measurement did not survive the load: %+v", m.Measured)
	}
}

// Every bound is either the model's own limit or a floor, and the set is
// closed: a reader can rely on the four names.
func TestOnlyTheModelBoundIsALimit(t *testing.T) {
	if !(&Measurement{Bound: BoundModel}).IsLimit() {
		t.Error("the model bound is not a limit")
	}
	for _, b := range []string{BoundPrefillDeadline, BoundServedWindow, BoundMemoryGuard} {
		if (&Measurement{Bound: b}).IsLimit() {
			t.Errorf("bound %q reads as the model's limit", b)
		}
		if !ValidBound(b) {
			t.Errorf("bound %q is not in the set", b)
		}
	}
	if ValidBound("declared") {
		t.Error("an unknown bound is in the set")
	}
}

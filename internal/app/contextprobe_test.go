package app

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

func readyModel(t *testing.T, a *App, id string, declared int64) {
	t.Helper()
	if err := a.Registry.Put(registry.Model{
		RepoID: id, Path: a.Paths.ModelDir(id), State: registry.StateReady,
		Bytes: 1 << 20, ContextLength: declared, KVChargePerToken: 64,
	}); err != nil {
		t.Fatal(err)
	}
}

// With the switch off a finished download arms nothing: the probe has no
// model due, the loop is not running, and only "Measure now" changes that.
func TestADownloadDoesNotStartAProbe(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	if due := a.Probe.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("a ready model is due for a probe with the switch off: %q", due)
	}
	if a.SelfTest.Enabled() {
		t.Error("the idle loop runs with both switches off")
	}
	if err := a.MeasureNow("org/m"); err != nil {
		t.Fatal(err)
	}
	if due := a.Probe.Due([]string{"org/m"}, time.Now()); due != "org/m" {
		t.Errorf("Measure now did not make the model due: %q", due)
	}
	if !a.SelfTest.Enabled() {
		t.Error("Measure now did not start the idle loop")
	}
	// A model with no declared window has nothing to bisect between.
	readyModel(t, a, "org/nowindow", 0)
	if err := a.MeasureNow("org/nowindow"); err == nil {
		t.Error("a model with no declared window was queued")
	}
}

// The switch on starts the loop and makes an unmeasured model due; off stops
// it. Quitting leaves nothing running.
func TestTheProbeSwitchStartsTheIdleLoopAndQuitStopsIt(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	on := config.Default()
	on.ContextProbe = true
	if err := a.SetConfig(on); err != nil {
		t.Fatal(err)
	}
	if !a.SelfTest.Enabled() {
		t.Error("the probe switch did not start the idle loop")
	}
	if due := a.Probe.Due([]string{"org/m"}, time.Now()); due != "org/m" {
		t.Errorf("an unmeasured model is not due with the switch on: %q", due)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if a.SelfTest.Enabled() || a.SelfTest.Status().Job != "" {
		t.Error("quitting left the idle loop or a job running")
	}
}

// Adopting a measurement writes the served window, through the same save
// every setting goes through; until then the measurement changes no charge.
func TestAdoptingAMeasurementSetsTheServedContextAndAnUnadoptedOneChangesNoCharge(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	m, _ := a.Registry.Get("org/m")
	before := a.chargeOf(m, chargedSize(m))
	if err := a.AdoptMeasurement("org/m"); !errors.Is(err, ErrNoMeasurement) {
		t.Errorf("adopting nothing = %v, want ErrNoMeasurement", err)
	}
	src := probeSources{a}
	prov := src.Provenance("org/m")
	if err := a.Registry.SetMeasurement("org/m", &registry.Measurement{
		Window: 65536, Bound: registry.BoundPrefillDeadline, At: 1,
		Runtime: prov.Runtime, BudgetBytes: prov.BudgetBytes, DecodeConcurrency: prov.DecodeConcurrency, ServedContext: prov.ServedContext,
	}); err != nil {
		t.Fatal(err)
	}
	m, _ = a.Registry.Get("org/m")
	if got := a.chargeOf(m, chargedSize(m)); got != before {
		t.Errorf("an unadopted measurement changed the charge from %d to %d", before, got)
	}
	if got := a.Config().ServedContext("org/m", 131072); got != 131072 {
		t.Errorf("an unadopted measurement changed the served window to %d", got)
	}
	if err := a.AdoptMeasurement("org/m"); err != nil {
		t.Fatal(err)
	}
	if got := a.Config().ServedContext("org/m", 131072); got != 65536 {
		t.Errorf("served window after adoption = %d, want 65536", got)
	}
	m, _ = a.Registry.Get("org/m")
	if got := a.chargeOf(m, chargedSize(m)); got >= before {
		t.Errorf("the adopted window did not lower the charge (%d before, %d after)", before, got)
	}
	// Adoption set the served window to the measured window itself, which
	// is not a move: the floor still stands, and the card keeps its figure.
	if m.Measured.Stale != "" {
		t.Errorf("after adoption the measurement is %q, want current", m.Measured.Stale)
	}
	// Any other served window is a move.
	c := a.Config()
	c.Models = map[string]config.ModelSettings{"org/m": {ServedContext: 32768}}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if m, _ := a.Registry.Get("org/m"); m.Measured.Stale != registry.StaleServedContext {
		t.Errorf("a served window other than the measured one left the measurement %q", m.Measured.Stale)
	}
}

// A save that moves the budget, the concurrency or a served window re-judges
// every measurement; a save that touches none of them leaves the verdict.
func TestASaveRejudgesMeasurementsAndAnUnrelatedSaveKeepsTheProbeSettings(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	prov := probeSources{a}.Provenance("org/m")
	if err := a.Registry.SetMeasurement("org/m", &registry.Measurement{
		Window: 65536, Bound: registry.BoundModel, At: 1,
		Runtime: prov.Runtime, BudgetBytes: prov.BudgetBytes, DecodeConcurrency: prov.DecodeConcurrency, ServedContext: prov.ServedContext,
	}); err != nil {
		t.Fatal(err)
	}
	c := config.Default()
	c.ContextProbe = true
	c.IdleThresholdSec = 120
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if m, _ := a.Registry.Get("org/m"); m.Measured.Stale != "" {
		t.Errorf("an unrelated save made the measurement stale: %q", m.Measured.Stale)
	}
	c = a.Config()
	c.LogLevel = "detailed"
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if got := a.Config(); !got.ContextProbe || got.IdleThresholdSec != 120 {
		t.Errorf("a save of the log level moved the probe settings: probe=%v threshold=%d", got.ContextProbe, got.IdleThresholdSec)
	}
	c = a.Config()
	c.Models = map[string]config.ModelSettings{"org/m": {ServedContext: 32768}}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if m, _ := a.Registry.Get("org/m"); m.Measured.Stale != registry.StaleServedContext {
		t.Errorf("a served-window save left the measurement %q, want stale for the served window", m.Measured.Stale)
	}
}

// The provenance the app stamps is the one it judges by: the pinned runtime,
// the pool's budget and concurrency, the model's served window.
func TestTheProvenanceIsWhatThePoolAndTheRuntimeSay(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	p := probeSources{a}.Provenance("org/m")
	if p.Runtime != runtime.MLXLMVersion() || p.BudgetBytes != a.Pool.MemoryBudget() ||
		p.DecodeConcurrency != a.Pool.DecodeConcurrency() || p.ServedContext != 131072 {
		t.Errorf("provenance = %+v", p)
	}
	url, _ := probeSources{a}.Endpoint()
	if url != "http://127.0.0.1:"+itoa(a.BindPort()) {
		t.Errorf("endpoint = %q, want this Mac's own loopback endpoint", url)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

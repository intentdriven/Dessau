package app

import (
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/capability"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
)

// The model from the capture (iss-2609202048576967): about 2 GB of weights,
// a declared window of 131,072 and a charged cache of 327,680 bytes a token,
// which at four batched requests is 160 GiB at the declared window — more than
// twice what a 128 GB Mac's default budget holds.
func captureModel(id string) registry.Model {
	return registry.Model{
		RepoID: id, Path: "/models/" + id, Bytes: 2 * gb, State: registry.StateReady,
		ContextLength: 131072, KVChargePerToken: 327680,
	}
}

// captureConfig is the shipped defaults at the capture's concurrency of four:
// the tests below are about the derivation, and they state the figure it
// divides by rather than inherit the default.
func captureConfig() config.Config {
	cfg := config.Default()
	cfg.DecodeConcurrency = 4
	return cfg
}

func putModel(t *testing.T, a *App, m registry.Model) registry.Model {
	t.Helper()
	if err := a.Registry.Put(m); err != nil {
		t.Fatalf("Put(%s): %v", m.RepoID, err)
	}
	got, err := a.Registry.Get(m.RepoID)
	if err != nil {
		t.Fatalf("Get(%s): %v", m.RepoID, err)
	}
	return got
}

// With no setting, a model whose declared window does not fit the budget is
// served at the largest window that does — worked out from the budget, the
// weights, the charge per token and the concurrency in force — and its charge
// at that window fits. A fresh install serves what it downloaded.
func TestTheDefaultServedWindowIsTheLargestThatFitsTheBudget(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	m := putModel(t, a, captureModel("org/long"))
	budget := a.Pool.MemoryBudget()
	sequences := int64(a.Pool.DecodeConcurrency())

	window, isDefault := a.ServedWindow(m)
	if !isDefault {
		t.Error("a model with no setting is not reported as served at the default")
	}
	want := (budget - capability.LoadCost(2*gb)) / (327680 * sequences)
	if window != want {
		t.Errorf("ServedWindow = %d, want %d — (budget − weights) / (charge per token × %d sequences)", window, want, sequences)
	}
	if window >= 131072 || window < MinServedWindow {
		t.Errorf("ServedWindow = %d, want it below the declared 131072 and no lower than the floor of %d", window, MinServedWindow)
	}
	charge := a.chargeOf(m, chargedSize(m))
	if charge > budget {
		t.Errorf("charge at the derived window = %d, more than the budget %d", charge, budget)
	}
	if want := capability.LoadCostOf(capability.Load{
		DiskBytes: 2 * gb, KVChargePerToken: 327680, Window: window, Sequences: sequences,
	}); charge != want {
		t.Errorf("charge = %d, want %d — the derived window is what is charged", charge, want)
	}
	// And the pool is handed the same window, so what the app says fits is
	// what the pool admits.
	resolved, err := modelSource{a}.Resolve("org/long")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ServedContext != window {
		t.Errorf("the pool resolves the window as %d, the app as %d", resolved.ServedContext, window)
	}
}

// A model that fits at its declared window is served at it: the derivation
// is capped at what the model declares, never above.
func TestTheDefaultServedWindowIsCappedAtTheDeclaredWindow(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	m := captureModel("org/fits")
	m.KVChargePerToken = 64
	m = putModel(t, a, m)
	window, isDefault := a.ServedWindow(m)
	if window != 131072 || !isDefault {
		t.Errorf("ServedWindow = %d (default %v), want the declared 131072 as the default", window, isDefault)
	}
}

// The derivation stops at a floor. A model that does not fit even there is
// served at the floor and refused by the pool, whose refusal names the knobs;
// it is never handed a window too small to be useful.
func TestTheDefaultServedWindowHasAFloor(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	m := captureModel("org/huge")
	m.KVChargePerToken = 10 << 20 // 10 MiB a token: 4,096 tokens at 4 sequences is 160 GiB
	m = putModel(t, a, m)
	window, isDefault := a.ServedWindow(m)
	if window != MinServedWindow || !isDefault {
		t.Errorf("ServedWindow = %d (default %v), want the floor %d as the default", window, isDefault, MinServedWindow)
	}
	if charge := a.chargeOf(m, chargedSize(m)); charge <= a.Pool.MemoryBudget() {
		t.Errorf("charge at the floor = %d, fits the budget %d; the floor is meant not to fit here", charge, a.Pool.MemoryBudget())
	}
	// A model declaring less than the floor cannot be served above what it
	// declares, so the floor is the declared window for it.
	small := captureModel("org/tiny")
	small.ContextLength, small.KVChargePerToken = 2048, 10<<20
	small = putModel(t, a, small)
	if window, _ := a.ServedWindow(small); window != 2048 {
		t.Errorf("ServedWindow = %d for a model declaring 2048, want 2048", window)
	}
}

// An explicit setting is what it always was: honoured as typed, capped at the
// declared window, and reported as the operator's rather than the default.
func TestAnExplicitServedContextIsHonoredAsBefore(t *testing.T) {
	cfg := captureConfig()
	cfg.Models = map[string]config.ModelSettings{
		"org/set":  {ServedContext: 32768},
		"org/over": {ServedContext: 300000},
	}
	a := newBudgetApp(t, 128*gb, cfg)
	set := putModel(t, a, captureModel("org/set"))
	over := putModel(t, a, captureModel("org/over"))
	if window, isDefault := a.ServedWindow(set); window != 32768 || isDefault {
		t.Errorf("ServedWindow = %d (default %v), want the operator's 32768", window, isDefault)
	}
	if window, isDefault := a.ServedWindow(over); window != 131072 || isDefault {
		t.Errorf("ServedWindow = %d (default %v), want a setting above the declared window capped at 131072", window, isDefault)
	}
	// A setting is honoured even where the default would be larger: a
	// setting above what fits is the operator's to make.
	big := captureModel("org/big")
	big.KVChargePerToken = 64
	big = putModel(t, a, big)
	c := a.Config()
	c.Models["org/big"] = config.ModelSettings{ServedContext: 100000}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if window, isDefault := a.ServedWindow(big); window != 100000 || isDefault {
		t.Errorf("ServedWindow = %d (default %v), want the operator's 100000", window, isDefault)
	}
}

// A model whose configuration gives no cache charge, or declares no window,
// keeps the behaviour it had: the declared window, and the flat charge.
func TestAModelWithNoCacheChargeKeepsItsDeclaredWindow(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	nokv := captureModel("org/nokv")
	nokv.KVChargePerToken = 0
	nokv = putModel(t, a, nokv)
	if window, isDefault := a.ServedWindow(nokv); window != 131072 || !isDefault {
		t.Errorf("ServedWindow = %d (default %v), want the declared 131072 as the default", window, isDefault)
	}
	if charge := a.chargeOf(nokv, chargedSize(nokv)); charge != capability.LoadCost(2*gb) {
		t.Errorf("charge = %d, want the flat %d", charge, capability.LoadCost(2*gb))
	}
	none := captureModel("org/nowindow")
	none.ContextLength = 0
	none = putModel(t, a, none)
	if window, _ := a.ServedWindow(none); window != 0 {
		t.Errorf("ServedWindow = %d for a model declaring no window, want 0", window)
	}
}

// The derivation follows the budget: a raise widens the default window, and
// the pool is charged again from the new figure at the save.
func TestTheDefaultServedWindowFollowsTheBudget(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	m := putModel(t, a, captureModel("org/long"))
	before, _ := a.ServedWindow(m)

	c := a.Config()
	c.MaxResidentBytes = 120 * gb
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	after, _ := a.ServedWindow(m)
	if after <= before {
		t.Errorf("ServedWindow = %d after raising the budget, was %d; want it wider", after, before)
	}
	if a.memoryBudget() != a.Pool.MemoryBudget() {
		t.Errorf("the app derives from a budget of %d while the pool enforces %d", a.memoryBudget(), a.Pool.MemoryBudget())
	}
}

// The too-small warning is silent for a model that fits at the derived
// window, and when it fires it names what to change: the served window, the
// batched requests, a smaller quantization — not the budget alone, which on
// the Mac from the capture could not be raised far enough.
func TestTheTooSmallWarningNamesTheKnobs(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	putModel(t, a, captureModel("org/long"))
	if w := a.MemoryBudgetWarning(); w != "" {
		t.Errorf("MemoryBudgetWarning() = %q for a model that fits at its default window, want none", w)
	}

	huge := captureModel("org/huge")
	huge.KVChargePerToken = 10 << 20
	putModel(t, a, huge)
	// The long model still fits, so the smallest charge is its and nothing
	// is warned about: what to keep is the operator's business.
	if w := a.MemoryBudgetWarning(); w != "" {
		t.Errorf("MemoryBudgetWarning() = %q while a model on this Mac fits, want none", w)
	}
	alone := newBudgetApp(t, 128*gb, captureConfig())
	putModel(t, alone, huge)
	w := alone.MemoryBudgetWarning()
	if w == "" {
		t.Fatal("a Mac whose only model does not fit at the floor draws no warning")
	}
	for _, want := range []string{"org/huge", "served context", "batched requests", "quantization", "4096", "4 batched"} {
		if !strings.Contains(w, want) {
			t.Errorf("warning %q does not say %q", w, want)
		}
	}
	if strings.Contains(w, "until it is raised") {
		t.Errorf("warning %q still says the budget is the knob", w)
	}
}

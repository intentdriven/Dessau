package app

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/Dessau/internal/capability"
	"github.com/intentdriven/Dessau/internal/registry"
)

// A hybrid model whose layout is spelled layers_block_type is charged for its
// attention layers only, so at a 128 GB Mac's default budget and four batched
// requests it is served at its whole declared window and leaves room beside
// it. Charged for every layer it was served at 57,384 tokens and its charge
// filled the budget, so nothing else could load (iss-2610031010368266).
func TestAHybridModelSpelledByBlockTypeLeavesRoomBesideIt(t *testing.T) {
	facts := registry.ReadModelFacts(filepath.Join("..", "registry", "testdata", "nemotron-h"))
	a := newBudgetApp(t, 128*gb, captureConfig())
	m := putModel(t, a, registry.Model{
		RepoID: "org/nemotron", Path: "/models/org/nemotron", State: registry.StateReady,
		Bytes:            17792502843, // the published 4-bit weights
		ContextLength:    facts.ContextLength,
		KVChargePerToken: facts.KVChargePerToken,
	})
	budget := a.Pool.MemoryBudget()
	sequences := int64(a.Pool.DecodeConcurrency())

	window, isDefault := a.ServedWindow(m)
	if window != 262144 || !isDefault {
		t.Errorf("ServedWindow = %d (default %v), want the declared 262144 as the default", window, isDefault)
	}
	charge := a.chargeOf(m, chargedSize(m))
	if want := capability.LoadCostOf(capability.Load{
		DiskBytes: 17792502843, KVChargePerToken: 30720, Window: 262144, Sequences: sequences,
	}); charge != want {
		t.Errorf("charge = %d, want %d — 30,720 bytes a token at the declared window", charge, want)
	}
	if room := budget - charge; room < 20*gb {
		t.Errorf("room beside the model = %d bytes of a %d budget, want at least 20 GiB", room, budget)
	}
}

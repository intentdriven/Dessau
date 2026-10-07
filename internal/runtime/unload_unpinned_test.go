package runtime

import (
	"context"
	"errors"
	"testing"
)

// UnloadUnpinned is Unload for the idle jobs: it refuses a pinned model with
// ErrPinned, whichever way the pin spells the id, and otherwise keeps
// Unload's rule — a model not loaded is ErrNotLoaded, a busy one ErrBusy
// (iss-2610042033419572).
func TestUnloadUnpinnedRefusesAPinnedModelAndOtherwiseUnloads(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	if err := p.UnloadUnpinned("org/m"); !errors.Is(err, ErrNotLoaded) {
		t.Errorf("a model not loaded: err = %v, want ErrNotLoaded", err)
	}

	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.UnloadUnpinned("org/m"); !errors.Is(err, ErrBusy) {
		t.Errorf("a model serving a request: err = %v, want ErrBusy", err)
	}
	release()

	p.SetPinned([]string{"ORG/M"})
	if err := p.UnloadUnpinned("Org/M"); !errors.Is(err, ErrPinned) {
		t.Errorf("a pinned model: err = %v, want ErrPinned", err)
	}
	if len(p.Resident()) != 1 {
		t.Fatal("a refused unload changed what is loaded")
	}

	// The operator's Unload still stops a pinned model.
	if err := p.Unload("org/m"); err != nil {
		t.Fatalf("the operator's unload of a pinned model: %v", err)
	}
	if len(p.Resident()) != 0 {
		t.Fatal("the operator's unload left a pinned model resident")
	}

	p.SetPinned(nil)
	if _, release, err = p.Acquire(context.Background(), "org/m"); err != nil {
		t.Fatal(err)
	}
	release()
	if err := p.UnloadUnpinned("org/m"); err != nil {
		t.Fatalf("an idle, unpinned model: %v", err)
	}
	if len(p.Resident()) != 0 {
		t.Error("the model is still resident after UnloadUnpinned")
	}
}

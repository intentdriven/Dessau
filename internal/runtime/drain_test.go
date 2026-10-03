package runtime

import (
	"context"
	"errors"
	"testing"
)

// A drained model admits nobody new, loaded or not, and the mark lifts when
// the swap says so (iss-2610031317470004).
func TestADrainedModelAdmitsNobodyNew(t *testing.T) {
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20, "org/cold": 1 << 20}}
	p := newTestPool(t, newFakeLauncher(), src, PoolOptions{MaxResidentBytes: 1 << 30})
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	undo := p.Drain("Org/M")
	undoCold := p.Drain("org/cold")
	if _, _, err := p.Acquire(context.Background(), "org/m"); !errors.Is(err, ErrUpdating) {
		t.Errorf("a loaded, drained model: err = %v, want ErrUpdating", err)
	}
	if _, _, err := p.Acquire(context.Background(), "org/cold"); !errors.Is(err, ErrUpdating) {
		t.Errorf("a drained model not loaded: err = %v, want ErrUpdating, and nothing loaded", err)
	}
	release()
	if err := p.Remove("org/m"); err != nil {
		t.Errorf("the drained model would not stop once its request finished: %v", err)
	}
	undo()
	undo() // lifting twice is harmless
	undoCold()
	_, release, err = p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatalf("after the mark lifted: %v", err)
	}
	release()
}

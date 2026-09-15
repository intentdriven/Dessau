package runtime

import (
	"context"
	"testing"
)

// Residency.Refusals is the sign a holder watches for a client it cannot
// otherwise see (internal/selftest): it counts a refusal an eviction could
// have cured, and not one that nothing given up would seat.
func TestResidencyCountsTheRefusalsAnEvictionCouldHaveCured(t *testing.T) {
	l := newFakeLauncher()
	// LoadCost is 1.2x, so 100 bytes costs 120: a 150-byte budget holds one.
	src := &fakeSource{models: map[string]int64{"org/a": 100, "org/b": 100}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 150})

	if n := p.Residency().Refusals; n != 0 {
		t.Fatalf("a fresh pool has %d refusals", n)
	}
	_, release, err := p.Acquire(context.Background(), "org/a")
	if err != nil {
		t.Fatal(err)
	}
	// org/a is in flight, so nothing can be evicted for org/b: refused, and
	// curable — releasing org/a would seat it.
	if _, _, err := p.Acquire(context.Background(), "org/b"); err == nil {
		t.Fatal("org/b loaded beside a held org/a on a budget that holds one")
	}
	if n := p.Residency().Refusals; n != 1 {
		t.Errorf("Refusals = %d after a curable refusal, want 1", n)
	}
	release()
}

func TestResidencyDoesNotCountARefusalNothingCouldCure(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/a": 100, "org/b": 100}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 150, Pinned: []string{"org/a"}})

	_, release, err := p.Acquire(context.Background(), "org/a")
	if err != nil {
		t.Fatal(err)
	}
	release()
	// The pinned model plus org/b is over the budget however long anyone
	// waits: the never-fits refusal, which yielding cannot help.
	if _, _, err := p.Acquire(context.Background(), "org/b"); err == nil {
		t.Fatal("org/b loaded beside a pinned org/a on a budget that holds one")
	}
	if n := p.Residency().Refusals; n != 0 {
		t.Errorf("Refusals = %d after a never-fits refusal, want 0: nothing given up would seat it", n)
	}
}

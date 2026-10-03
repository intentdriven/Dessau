package runtime

import (
	"context"
	"testing"
	"time"
)

// A model server whose generation thread has died answers /health with 503
// and keeps its socket up, so every request to it waits for ever. The pool
// reads that as a crash: the process is stopped, the entry goes, and the next
// request starts the model afresh (iss-2610031444343397).
func TestAServerWhoseGenerationDiedIsTreatedAsCrashed(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30, HealthInterval: 20 * time.Millisecond})
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	l.mu.Lock()
	first := l.servers["org/m"]
	l.mu.Unlock()
	first.SetGenerationDead()

	deadline := time.Now().Add(5 * time.Second)
	for len(p.Residency().Models) != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := len(p.Residency().Models); n != 0 {
		t.Fatalf("the dead server is still resident: %+v", p.Residency().Models)
	}
	_, release, err = p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatalf("the model did not start afresh: %v", err)
	}
	release()
	l.mu.Lock()
	launched := len(l.launched)
	l.mu.Unlock()
	if launched != 2 {
		t.Errorf("launched %d times, want 2", launched)
	}
}

// A healthy server is left alone, however often it is asked.
func TestAHealthyServerIsLeftAlone(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30, HealthInterval: 5 * time.Millisecond})
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	time.Sleep(100 * time.Millisecond)
	if n := len(p.Residency().Models); n != 1 {
		t.Errorf("a healthy server was taken out: %d resident", n)
	}
}

package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Release unloads a loaded model a program has finished with, and only one
// nobody is relying on: an idle, unpinned model outside its eviction grace
// (adr-2610031153127219). Every other case is refused at once, with nothing
// changed and nothing waited for.
func TestReleaseUnloadsOnlyAModelNobodyIsRelyingOn(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	if err := p.Release("org/m", Caller{Kind: "this_mac"}); !errors.Is(err, ErrNotLoaded) {
		t.Errorf("a model not loaded: err = %v, want ErrNotLoaded", err)
	}

	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Release("org/m", Caller{Kind: "this_mac"}); !errors.Is(err, ErrBusy) {
		t.Errorf("a model serving a request: err = %v, want ErrBusy", err)
	}
	release()

	p.SetPinned([]string{"ORG/M"})
	if err := p.Release("org/m", Caller{Kind: "this_mac"}); !errors.Is(err, ErrPinned) {
		t.Errorf("a pinned model: err = %v, want ErrPinned", err)
	}
	p.SetPinned(nil)
	if len(p.Resident()) != 1 {
		t.Fatal("a refused release changed what is loaded")
	}

	if err := p.Release("Org/M", Caller{Kind: "this_mac"}); err != nil {
		t.Fatalf("an idle, unpinned model: %v", err)
	}
	if len(p.Resident()) != 0 {
		t.Error("the model is still resident after Release")
	}
}

// A model still loading is refused at once: its load is somebody's request.
func TestReleaseRefusesAModelStillLoading(t *testing.T) {
	l := newFakeLauncher()
	l.loadDelay = 2 * time.Second
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Acquire(ctx, "org/m")
	deadline := time.Now().Add(2 * time.Second)
	for len(p.Resident()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	start := time.Now()
	err := p.Release("org/m", Caller{Kind: "this_mac"})
	if !errors.Is(err, ErrLoading) {
		t.Errorf("a model still loading: err = %v, want ErrLoading", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Errorf("the refusal waited %v; it must come at once", time.Since(start))
	}
}

// A model inside its eviction grace — protected for a while after it last
// answered — is refused: a program finishing does not take another client's
// protection away.
func TestReleaseRefusesAModelInItsGrace(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	now := time.Now()
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30, EvictionGrace: 2 * time.Minute,
		now: func() time.Time { return now }})
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if err := p.Release("org/m", Caller{Kind: "this_mac"}); !errors.Is(err, ErrInGrace) {
		t.Errorf("a model inside its grace: err = %v, want ErrInGrace", err)
	}
	now = now.Add(3 * time.Minute)
	if err := p.Release("org/m", Caller{Kind: "this_mac"}); err != nil {
		t.Errorf("a model past its grace: %v", err)
	}
}

// releasingObserver is a recordingObserver that also wants to know who
// released a model.
type releasingObserver struct {
	*recordingObserver
	mu       sync.Mutex
	released []Caller
}

func (o *releasingObserver) EntryReleased(repoID string, by Caller) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.released = append(o.released, by)
}

func (o *releasingObserver) callers() []Caller {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Caller(nil), o.released...)
}

// A released model leaves for its own reason, not as an operator's unload,
// and an observer that asks is told who released it: the statistics and the
// log say a program did it and which kind of caller it was.
func TestAReleaseIsReportedWithWhoAskedForIt(t *testing.T) {
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}

	plain := &recordingObserver{}
	p := newTestPool(t, newFakeLauncher(), src, PoolOptions{MaxResidentBytes: 1 << 30, Observer: plain})
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if err := p.Release("org/m", Caller{Kind: "api_key"}); err != nil {
		t.Fatal(err)
	}
	if got := awaitStop(t, plain, "org/m"); got != StopReleased {
		t.Errorf("an observer that does not ask is told %q, want %q", got, StopReleased)
	}

	asks := &releasingObserver{recordingObserver: &recordingObserver{}}
	p = newTestPool(t, newFakeLauncher(), src, PoolOptions{MaxResidentBytes: 1 << 30, Observer: asks})
	_, release, err = p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	want := Caller{Kind: "paired_client", Client: "Bob's iPad"}
	if err := p.Release("org/m", want); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(asks.callers()) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := asks.callers(); len(got) != 1 || got[0] != want {
		t.Errorf("the observer was told %v, want [%v]", got, want)
	}
	if got := asks.stopReasons(); len(got) != 0 {
		t.Errorf("a release was reported twice: also as %v", got)
	}
}

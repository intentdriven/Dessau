package runtime

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// softHolder is a caller that holds a model the way the self-test does: the
// hold is marked preemptible, and when the pool asks for it back the holder
// lets go. yielded closes when the pool asked.
type softHolder struct {
	yielded chan struct{}
	gone    chan struct{}
}

// holdSoftly acquires repoID as a preemptible hold that releases when the
// pool cancels it.
func holdSoftly(t *testing.T, p *Pool, repoID string) *softHolder {
	t.Helper()
	h := &softHolder{yielded: make(chan struct{}), gone: make(chan struct{})}
	held, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	_, release, err := p.Acquire(WithSoftHold(context.Background(), cancel), repoID)
	if err != nil {
		t.Fatalf("holding %s softly: %v", repoID, err)
	}
	go func() {
		defer close(h.gone)
		<-held.Done()
		close(h.yielded)
		release()
	}()
	return h
}

// The whole of iss-2609100526194406: a client whose load needs the memory a
// preemptible hold is holding takes it, rather than being refused once and
// served on a retry.
func TestAClientsLoadEvictsAPreemptibleHoldInsteadOfBeingRefused(t *testing.T) {
	l := newFakeLauncher()
	p := newTestPool(t, l, graceModels(), PoolOptions{MaxResidentBytes: graceBudget})

	h := holdSoftly(t, p, "org/a")

	_, release, err := p.Acquire(context.Background(), "org/b")
	if err != nil {
		t.Fatalf("a client's load was refused the memory a preemptible hold was holding: %v", err)
	}
	defer release()

	select {
	case <-h.yielded:
	default:
		t.Error("the client was served without the holder's context being cancelled")
	}
	if ids := residentIDs(p); !slices.Equal(ids, []string{"org/b"}) {
		t.Errorf("resident = %v, want the preempted model replaced by the client's", ids)
	}
	if n := p.Residency().Refusals; n != 0 {
		t.Errorf("Refusals = %d, want 0: the client was served rather than refused", n)
	}
	if n := p.Waiting(); n != 0 {
		t.Errorf("Waiting() = %d once the client has its model, want 0", n)
	}
}

// The existing behaviour, unchanged: an ordinary client's hold is nobody
// else's to take, so a load that needs its memory is refused.
func TestAnOrdinaryHoldIsRefusedRatherThanPreempted(t *testing.T) {
	l := newFakeLauncher()
	p := newTestPool(t, l, graceModels(), PoolOptions{MaxResidentBytes: graceBudget})

	_, release, err := p.Acquire(context.Background(), "org/a")
	if err != nil {
		t.Fatalf("Acquire(org/a): %v", err)
	}
	defer release()

	_, _, err = p.Acquire(context.Background(), "org/b")
	var noRoom *NoRoomError
	if !errors.As(err, &noRoom) {
		t.Fatalf("Acquire(org/b) = %v, want the no-room refusal: an ordinary hold is not preemptible", err)
	}
	if ids := residentIDs(p); !slices.Equal(ids, []string{"org/a"}) {
		t.Errorf("resident = %v, want the held model untouched", ids)
	}
	if n := p.Residency().Refusals; n != 1 {
		t.Errorf("Refusals = %d after a curable refusal, want 1", n)
	}
}

// A preemptible caller does not preempt: the self-test never takes a model
// from anyone, itself included.
func TestASoftHoldDoesNotPreemptAnotherSoftHold(t *testing.T) {
	l := newFakeLauncher()
	p := newTestPool(t, l, graceModels(), PoolOptions{MaxResidentBytes: graceBudget})

	h := holdSoftly(t, p, "org/a")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err := p.Acquire(WithSoftHold(ctx, cancel), "org/b")
	var noRoom *NoRoomError
	if !errors.As(err, &noRoom) {
		t.Fatalf("a preemptible load took a preemptible hold's memory: %v", err)
	}
	select {
	case <-h.yielded:
		t.Error("the first holder was asked to yield for a load that is itself preemptible")
	default:
	}
}

// A holder that does not let go costs the client exactly what it cost before:
// the ordinary refusal. Nothing is torn down for it, and the model it would
// not give up is still resident with its server running.
func TestAPreemptibleHoldThatNeverYieldsEndsInTheOrdinaryRefusal(t *testing.T) {
	l := newFakeLauncher()
	p := newTestPool(t, l, graceModels(), PoolOptions{
		MaxResidentBytes: graceBudget,
		// The bound the wait for a yield is held to; short, so the test is.
		DrainWait: 100 * time.Millisecond,
	})

	// Acquired softly and never released: the yield is heard and ignored.
	_, release, err := p.Acquire(WithSoftHold(context.Background(), func() {}), "org/a")
	if err != nil {
		t.Fatalf("Acquire(org/a): %v", err)
	}
	defer release()

	_, _, err = p.Acquire(context.Background(), "org/b")
	var noRoom *NoRoomError
	if !errors.As(err, &noRoom) {
		t.Fatalf("Acquire(org/b) = %v, want the no-room refusal once the holder would not let go", err)
	}
	if ids := residentIDs(p); !slices.Equal(ids, []string{"org/a"}) {
		t.Errorf("resident = %v, want the model that was never given up still held", ids)
	}
	select {
	case <-l.procFor("org/a").stopped:
		t.Error("the model server was stopped under a holder that still holds it")
	default:
	}
}

// A hold is taken only when taking it seats the load. Here it does not: a
// client is holding the other model, so freeing the preemptible one still
// leaves the machine short, and the run is left alone rather than cancelled
// for a load that is refused anyway (iss-2609190031063965).
func TestAHoldIsNotTakenWhenTakingItWouldStillNotFit(t *testing.T) {
	l := newFakeLauncher()
	// LoadCost is 1.2x: the two small models cost 240 each and the large one
	// 480, so the budget holds both small ones and the large one fits only if
	// both go.
	src := &fakeSource{models: map[string]int64{"org/a": 200, "org/b": 200, "org/big": 400}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 500})

	// A client's request in flight on org/a: not the pool's to take.
	_, release, err := p.Acquire(context.Background(), "org/a")
	if err != nil {
		t.Fatalf("Acquire(org/a): %v", err)
	}
	defer release()
	h := holdSoftly(t, p, "org/b")

	_, _, err = p.Acquire(context.Background(), "org/big")
	var noRoom *NoRoomError
	if !errors.As(err, &noRoom) {
		t.Fatalf("Acquire(org/big) = %v, want the no-room refusal: it does not fit even with the hold given up", err)
	}
	select {
	case <-h.yielded:
		t.Error("a hold was taken for a load that could not be seated by taking it")
	default:
	}
}

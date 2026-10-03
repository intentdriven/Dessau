package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/selftest"
	"github.com/intentdriven/Dessau/internal/toolprobe"
)

// Under steady traffic the old version never falls idle on its own: requests
// overlap, so one is always in flight. While a swap waits, new requests are
// refused, the ones in flight finish, and the update lands
// (iss-2610031317470004, at the maintainer's answer).
func TestAnUpdateLandsUnderSteadyTraffic(t *testing.T) {
	a, h := newStagedApp(t)
	a.drainWait = 3 * time.Second

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	refusedUpdating := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_, release, err := a.Pool.Acquire(context.Background(), "org/repo")
			if err != nil {
				if errors.Is(err, runtime.ErrUpdating) {
					mu.Lock()
					refusedUpdating++
					mu.Unlock()
				}
				time.Sleep(5 * time.Millisecond)
				continue
			}
			wg.Add(1)
			go func() { defer wg.Done(); time.Sleep(60 * time.Millisecond); release() }()
			time.Sleep(20 * time.Millisecond)
		}
	}()
	t.Cleanup(func() { close(stop); wg.Wait() })

	time.Sleep(100 * time.Millisecond) // traffic is flowing and overlapping
	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV2 {
		t.Errorf("the update did not land under steady traffic: commit %s", m.Commit)
	}
	mu.Lock()
	defer mu.Unlock()
	if refusedUpdating == 0 {
		t.Error("no request was refused as updating while the swap drained the model")
	}
}

// An update abandoned because the old version never fell idle lifts its
// drain: the model serves new requests again rather than staying refused.
func TestAnAbandonedUpdateLiftsItsDrain(t *testing.T) {
	a, h := newStagedApp(t)
	a.drainWait = 200 * time.Millisecond
	_, release, err := a.Pool.Acquire(context.Background(), "org/repo")
	if err != nil {
		t.Fatal(err)
	}
	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	release()
	if m, _ := a.Registry.Get("org/repo"); m.Commit == commitV2 {
		t.Fatal("the update landed although a request held the old version throughout")
	}
	_, release2, err := a.Pool.Acquire(context.Background(), "org/repo")
	if err != nil {
		t.Fatalf("after the update was abandoned the model still refuses requests: %v", err)
	}
	release2()
}

// The self-test and the tool-call probe read a model being updated as one to
// come back to, not as a load that failed.
func TestIdleWorkReadsAnUpdateAsNotNow(t *testing.T) {
	a, _ := newStagedApp(t)
	undrain := a.Pool.Drain("org/repo")
	defer undrain()
	if _, _, err := (selfTestServer{a}).Acquire(context.Background(), "org/repo"); !errors.Is(err, selftest.ErrNotNow) {
		t.Errorf("self-test: err = %v, want ErrNotNow", err)
	}
	if _, _, err := (toolProbeSources{a}).Acquire(context.Background(), "org/repo"); !errors.Is(err, toolprobe.ErrGone) {
		t.Errorf("tool-call probe: err = %v, want ErrGone", err)
	}
}

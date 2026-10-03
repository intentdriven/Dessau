package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/runtime"
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

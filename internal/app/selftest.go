package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// selfTestSource is the identity the self-test's loads carry in the pool's
// load-waiter queue (runtime.WithSource), so its place there is its own and
// never a client's.
const selfTestSource = "dessau-self-test"

// selfTestServer is what the self-test sees of the app: the registry's ready
// models, the pool's ordinary Acquire — never a second launcher, so the
// budget, the pins, the served window and the eviction rules are the pool's
// own — and the pool's view of what is going on.
//
// It holds none of the app's locks across any call (adr-2609091239058072):
// each method takes the pool's or the registry's own lock for the length of
// one read and hands back a copy.
type selfTestServer struct{ a *App }

// Ready is every ready model but one whose last load failed under the
// provenance in force: the record on it stands until that moves or a person
// retries by hand, and idle work is neither (iss-2609211334570516).
func (s selfTestServer) Ready() []string {
	models := s.a.Registry.Ready()
	rule := s.a.Config().EffectiveChatRule()
	ids := make([]string, 0, len(models))
	for _, m := range models {
		// The self-test sends a chat request, so a model that cannot chat is
		// not one it loads (iss-2610031010371709).
		if m.LoadFailed() || !m.CanChat(rule) {
			continue
		}
		ids = append(ids, m.RepoID)
	}
	return ids
}

// Acquire is the pool's ordinary Acquire with two tags on the context: the
// self-test's identity in the queue for memory, and the run's own way of being
// told to let go. The second is what makes the hold a soft one — a client
// whose load needs the memory takes the model and the pool cancels the run,
// rather than the client being refused and the run finding out from the
// refusal count.
func (s selfTestServer) Acquire(ctx context.Context, repoID string) (selftest.Upstream, func(), error) {
	poolCtx := runtime.WithSoftHold(runtime.WithSource(ctx, selfTestSource), selftest.YieldFrom(ctx))
	up, release, err := s.a.Pool.Acquire(poolCtx, repoID)
	if errors.Is(err, runtime.ErrUpdating) {
		return selftest.Upstream{}, nil, fmt.Errorf("%w: %w", selftest.ErrNotNow, err)
	}
	if err != nil {
		return selftest.Upstream{}, nil, err
	}
	return selftest.Upstream{BaseURL: up.BaseURL, Transport: up.Transport, ModelArg: up.ModelArg}, release, nil
}

func (s selfTestServer) Activity() selftest.Activity {
	res := s.a.Pool.Residency()
	act := selftest.Activity{
		Waiting:     s.a.Pool.Waiting(),
		Downloading: len(s.a.Downloading()),
		Refusals:    res.Refusals,
	}
	act.Requests, act.LastRequest = s.a.Clients.Snapshot()
	for _, m := range res.Models {
		act.Models = append(act.Models, selftest.ModelActivity{
			RepoID: m.RepoID, InFlight: m.InFlight, LastUsed: m.LastUsed,
		})
	}
	return act
}

// Unload stops a model the self-test loaded, and never a pinned one: a model
// pinned during the run stays loaded and pinned (iss-2610032241098944); see
// unloadUnpinned.
func (s selfTestServer) Unload(repoID string) error {
	return s.a.unloadUnpinned(repoID, selftest.ErrPinned)
}

// Pinned is the pool's pin, the one the context probe's candidates and both
// idle jobs' unloads read (isPinned).
func (s selfTestServer) Pinned(repoID string) bool { return s.a.isPinned(repoID) }

func (s selfTestServer) Concurrency() int { return s.a.Pool.DecodeConcurrency() }

// Fits says whether the model can be loaded beside what the pool holds
// without evicting anything: its charge, worked out where every charge is
// (chargeOf), against the budget less what is resident and what is still
// exiting. A model already resident fits. The figures are a snapshot, and the
// pool decides for itself at the load; this only keeps the self-test from
// asking for a load it knows would take something out.
func (s selfTestServer) Fits(repoID string) bool {
	res := s.a.Pool.Residency()
	key := config.FoldRepoID(repoID)
	var used int64
	for _, m := range res.Models {
		if config.FoldRepoID(m.RepoID) == key {
			return true
		}
		used += m.Charge
	}
	used += res.ExitingBytes
	m, err := s.a.Registry.Get(repoID)
	if err != nil {
		return false
	}
	size := chargedSize(m)
	if size <= 0 {
		return false
	}
	return used+s.a.chargeOf(m, size) <= s.a.Pool.MemoryBudget()
}

package contextprobe

import (
	"net/http"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/registry"
)

// lowerServed puts a lower served window into force for the one candidate:
// the app's candidate and provenance read it, and the gateway's served-window
// check refuses above it.
func lowerServed(src *fakeSources, gw *fakeGateway, window int64) {
	src.mu.Lock()
	src.cands[0].Served = window
	src.prov = registry.Provenance{Runtime: "0.31.3", BudgetBytes: 1, DecodeConcurrency: 4, ServedContext: window}
	src.mu.Unlock()
	gw.mu.Lock()
	gw.accept = window
	gw.mu.Unlock()
}

// sweptPast reports whether the gateway has seen a step of at least size
// tokens.
func sweptPast(gw *fakeGateway, size int) func() bool {
	return sweptPastSince(gw, 0, size)
}

// sweptPastSince is sweptPast counting only the steps from the from'th on.
func sweptPastSince(gw *fakeGateway, from, size int) func() bool {
	return func() bool {
		seen := gw.seen()
		for _, s := range seen[min(from, len(seen)):] {
			if len(s) > size*4 {
				return true
			}
		}
		return false
	}
}

// assertHeldUnder fails unless the measurement fits the served window in
// force and is stamped with the provenance it was made under.
func assertHeldUnder(t *testing.T, m *registry.Measurement, served int64) {
	t.Helper()
	if m.Window > served {
		t.Errorf("saved window %d is above the served window %d in force", m.Window, served)
	}
	if m.ServedContext != served {
		t.Errorf("measurement stamped with served window %d, want %d", m.ServedContext, served)
	}
}

// A run that yields keeps its bounds for a resume, but they were made under
// the provenance in force when it began. When the served window is lowered
// meanwhile, the resumed run must not bisect to them and save a window above
// the new served window stamped as current (iss-2610032241096901).
func TestAResumedProbeDropsBoundsMadeUnderAnotherProvenance(t *testing.T) {
	gw := newFakeGateway(t, 131072, http.StatusBadRequest)
	gw.delay = 30 * time.Millisecond
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	pool := newFakePool("org/m")
	r := runner(t, pool, p)
	r.SetEnabled(true)
	waitFor(t, "the sweep to pass 16,384 tokens", sweptPast(gw, 32768))
	pool.mu.Lock()
	pool.inFlight["org/other"] = 1
	pool.mu.Unlock()
	waitFor(t, "the probe to yield", func() bool { return r.Status().Job == "" })
	if src.result("org/m") != nil {
		t.Fatal("a yielded probe wrote a figure")
	}
	// The yielded step's handler may still be reading its prompt; the steps
	// counted from here on are the resumed run's alone.
	waitFor(t, "the gateway to finish the yielded step", func() bool { return gw.active() == 0 })
	lowerServed(src, gw, 8192)
	from := len(gw.seen())
	pool.mu.Lock()
	delete(pool.inFlight, "org/other")
	pool.mu.Unlock()
	waitFor(t, "a measurement after resuming", func() bool { return src.result("org/m") != nil })
	assertHeldUnder(t, src.result("org/m"), 8192)
	// The resume itself starts over: no step after the move is sized from
	// the old bounds. Without the check on resume, the check before saving
	// would still throw the figure away and a fresh run save a right one, so
	// only the steps show that the resume did not bisect between bounds the
	// settings in force do not serve.
	if sweptPastSince(gw, from, 8192)() {
		t.Error("the resumed run sent a step above the served window in force: it resumed from bounds made under the old one")
	}
}

// The bounds' ceiling and their provenance come from one read. Here the
// candidate still carries the served window from before a raise while the
// provenance already has the raised one, which is what a raise landing
// between the two reads looks like: a ceiling from the candidate would stop
// the sweep at the old window and save that figure, bound by the served
// window, stamped with the raised one.
func TestAProbesCeilingComesFromTheProvenanceItIsStampedWith(t *testing.T) {
	gw := newFakeGateway(t, 131072, http.StatusBadRequest)
	cand := model
	cand.Served = 8192
	src := newFakeSources(gw.srv.URL, cand)
	src.prov = registry.Provenance{Runtime: "0.31.3", BudgetBytes: 1, DecodeConcurrency: 4, ServedContext: 131072}
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	m := src.result("org/m")
	if m.ServedContext != 131072 {
		t.Fatalf("measurement stamped with served window %d, want 131072", m.ServedContext)
	}
	if m.Bound == registry.BoundServedWindow && m.Window <= m.ServedContext/2 {
		t.Errorf("saved window %d bound by the served window, stamped with a served window of %d: the ceiling came from another read", m.Window, m.ServedContext)
	}
}

// The same move inside one run, with no yield between: the bounds were made
// under the provenance in force when the run began, so the figure they give
// is not saved as current under another.
func TestAProbeDoesNotSaveAFigureAcrossAProvenanceChange(t *testing.T) {
	gw := newFakeGateway(t, 131072, http.StatusBadRequest)
	gw.delay = 30 * time.Millisecond
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "the sweep to pass 16,384 tokens", sweptPast(gw, 32768))
	lowerServed(src, gw, 8192)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	assertHeldUnder(t, src.result("org/m"), 8192)
}

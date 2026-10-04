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
	return func() bool {
		for _, s := range gw.seen() {
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
	lowerServed(src, gw, 8192)
	pool.mu.Lock()
	delete(pool.inFlight, "org/other")
	pool.mu.Unlock()
	waitFor(t, "a measurement after resuming", func() bool { return src.result("org/m") != nil })
	assertHeldUnder(t, src.result("org/m"), 8192)
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

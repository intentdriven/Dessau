package contextprobe

import (
	"net/http"
	"testing"
	"time"
)

// A pinned candidate is never picked, by the switch or from the queue: the
// probe stops its model between steps, and a pin promises nothing does
// (iss-2609211754251373).
func TestAPinnedModelIsNotPickedForAMeasurement(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	pinned := model
	pinned.Pinned = true
	src := newFakeSources(gw.srv.URL, pinned)
	p := probeOf(src, true)
	p.MeasureNow("org/m")
	if due := p.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("Due = %q for a pinned model, want none", due)
	}
	if q := p.Queued(); len(q) != 0 {
		t.Errorf("a pinned model stays queued: %v", q)
	}
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	time.Sleep(40 * time.Millisecond)
	if n := len(src.unloaded()); n != 0 {
		t.Errorf("the probe unloaded a pinned model %d times", n)
	}
	if n := len(gw.seen()); n != 0 {
		t.Errorf("%d requests were sent to a pinned model", n)
	}
}

// A pin that lands while a measurement is queued or running is met at the
// probe's next unload, which is refused: the run stops there, at once and
// without retrying, keeps what it verified, and is not marked incomplete, so
// it resumes once the model is unpinned.
func TestAPinThatLandsMidRunStopsTheMeasurement(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	src.unloadErr = ErrPinned
	p := probeOf(src, false)
	p.MeasureNow("org/m")
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "the run to stop", func() bool { return len(p.Queued()) == 0 })
	time.Sleep(30 * time.Millisecond)
	if n := len(src.unloaded()); n != 1 {
		t.Errorf("%d unload attempts on a pinned model, want the one refused", n)
	}
	if n := len(gw.seen()); n != 0 {
		t.Errorf("%d requests were sent after the unload was refused", n)
	}
	src.mu.Lock()
	incomplete := src.incomplete["org/m"]
	src.mu.Unlock()
	if incomplete {
		t.Error("a run stopped by a pin was marked incomplete")
	}
	p.mu.Lock()
	_, kept := p.bounds["org/m"]
	p.mu.Unlock()
	if !kept {
		t.Error("a run stopped by a pin dropped its bounds")
	}
}

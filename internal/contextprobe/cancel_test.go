package contextprobe

import (
	"testing"
	"time"
)

// Cancel forgets a model's measurement: it leaves the queue, its bisection
// is dropped, and it reads as an incomplete probe, which nothing retries but
// Measure now (maintainer's decision, 2026-10-03; iss-2610031818057157).
func TestCancelForgetsTheMeasurement(t *testing.T) {
	src := newFakeSources("http://127.0.0.1:1", model)
	p := probeOf(src, true)
	p.MeasureNow("org/m")
	p.mu.Lock()
	p.bounds["org/m"] = &bounds{lo: 4096, hi: 65536, loTokens: 4096}
	p.mu.Unlock()

	p.Cancel("org/m")

	if q := p.Queued(); len(q) != 0 {
		t.Errorf("still queued: %v", q)
	}
	p.mu.Lock()
	_, kept := p.bounds["org/m"]
	p.mu.Unlock()
	if kept {
		t.Error("the bisection was kept for a resume")
	}
	if !src.incomplete["org/m"] {
		t.Error("the cancelled measurement does not read as incomplete")
	}
	if due := p.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("the cancelled model is due again: %q", due)
	}
}

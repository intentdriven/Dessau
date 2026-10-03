package contextprobe

import (
	"context"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/selftest"
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

// A cancel that lands before the interrupted run has made its bounds still
// leaves nothing to resume from: the run that yields afterwards drops the
// bounds it made, and Measure now starts afresh (iss-2610032210290992).
func TestACancelBeforeTheRunMakesItsBoundsLeavesNoneBehind(t *testing.T) {
	src := newFakeSources("http://127.0.0.1:1", model)
	p := probeOf(src, true)
	p.MeasureNow("org/m")
	p.Cancel("org/m")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the run was interrupted as it began
	p.Run(&selftest.Session{
		Ctx: ctx, Hold: func(int) {}, Yielded: func() bool { return true },
		Report: func(string) {}, Fits: func(string) bool { return true },
	}, "org/m")
	p.mu.Lock()
	_, kept := p.bounds["org/m"]
	p.mu.Unlock()
	if kept {
		t.Error("the interrupted run kept bounds for a cancelled measurement")
	}
	p.MeasureNow("org/m")
	if q := p.Queued(); len(q) != 1 {
		t.Errorf("Measure now after a cancel queued %v, want the model", q)
	}
}

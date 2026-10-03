package app

import (
	"errors"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/contextprobe"
)

// A pinned model is never stopped by the context probe: the probe's unload
// is refused, its candidate says it is pinned so the pick passes it by, and
// Measure now says why, whichever way the pin spells the id
// (iss-2609211754251373).
func TestTheContextProbeNeverStopsAPinnedModel(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	c := a.Config()
	c.Models = map[string]config.ModelSettings{"ORG/M": {Pinned: true}}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	src := probeSources{a}
	if err := src.Unload("org/m"); !errors.Is(err, contextprobe.ErrPinned) {
		t.Errorf("the probe's unload of a pinned model = %v, want ErrPinned", err)
	}
	pinned := false
	for _, cand := range src.Candidates() {
		if cand.RepoID == "org/m" {
			pinned = cand.Pinned
		}
	}
	if !pinned {
		t.Error("the candidate for a pinned model does not say it is pinned")
	}
	if err := a.MeasureNow("org/m"); err == nil {
		t.Error("Measure now queued a pinned model")
	}
	if q := a.Probe.Queued(); len(q) != 0 {
		t.Errorf("queued %v for a pinned model", q)
	}
}

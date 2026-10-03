package app

import (
	"slices"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// Ticking a model's transcript box takes it off the debug-logging list: a
// save that marks a model as keeping no transcript disarms it, so a mark
// armed before the box was ticked does not launch it at debug
// (iss-2610032212269524; the 2026-09-20 decision). Other armed models keep
// their marks.
func TestMarkingAModelKeepsNoTranscriptDisarmsItsDebugLog(t *testing.T) {
	a := newTestApp(t)
	for _, id := range []string{"org/m", "org/other"} {
		if err := a.Pool.ArmDebugLog(id); err != nil {
			t.Fatal(err)
		}
	}
	cfg := a.Config()
	cfg.Models = map[string]config.ModelSettings{"ORG/M": {NoTranscript: true}}
	if err := a.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	armed := a.Pool.DebugArmed()
	if slices.Contains(armed, "org/m") {
		t.Errorf("the marked model is still armed for debug logging: %v", armed)
	}
	if !slices.Contains(armed, "org/other") {
		t.Errorf("an unmarked model lost its arming: %v", armed)
	}
}

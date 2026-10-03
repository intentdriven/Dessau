package gateway

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// Ticking a model's transcript box takes it off the debug-logging list: a
// settings save that marks a model as keeping no transcript disarms it, so a
// mark armed before the box was ticked does not launch it at debug
// (iss-2610032212269524; the 2026-09-20 decision). Other armed models keep
// their marks.
func TestMarkingAModelKeepsNoTranscriptDisarmsItsDebugLog(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	for _, id := range []string{"org/m", "org/other"} {
		if err := a.Pool.ArmDebugLog(id); err != nil {
			t.Fatal(err)
		}
	}
	body := `{"host":"127.0.0.1","port":11535,"api_key":"","decode_concurrency":1,"idle_timeout_sec":0,` +
		`"models":{"ORG/M":{"no_transcript":true}}}`
	resp, err := srv.Client().Post(srv.URL+"/api/settings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	armed := a.Pool.DebugArmed()
	if slices.Contains(armed, "org/m") {
		t.Errorf("the marked model is still armed for debug logging: %v", armed)
	}
	if !slices.Contains(armed, "org/other") {
		t.Errorf("an unmarked model lost its arming: %v", armed)
	}
}

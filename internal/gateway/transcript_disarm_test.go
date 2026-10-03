package gateway

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/app"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
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

// Arming debug logging waits for a settings save in progress, so a save that
// marks a model and an arm of it are ordered: the arm either lands first and
// is disarmed by the save, or reads the mark and is refused
// (iss-2610032221548858).
func TestArmingDebugLogWaitsForASettingsSave(t *testing.T) {
	a, err := app.New(app.Options{Paths: config.NewPaths(t.TempDir()), Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	if err := a.Registry.Put(registry.Model{RepoID: "org/m", State: registry.StateReady}); err != nil {
		t.Fatal(err)
	}
	c := &Control{App: a}
	mux := http.NewServeMux()
	c.Routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c.settingsMu.Lock() // a save in progress
	done := make(chan int, 1)
	go func() {
		resp, err := srv.Client().Post(srv.URL+"/api/models/debug-log", "application/json",
			strings.NewReader(`{"model":"org/m","armed":true}`))
		if err != nil {
			done <- 0
			return
		}
		resp.Body.Close()
		done <- resp.StatusCode
	}()
	select {
	case <-done:
		c.settingsMu.Unlock()
		t.Fatal("the arm did not wait for the settings save in progress")
	case <-time.After(200 * time.Millisecond):
	}
	c.settingsMu.Unlock()
	if code := <-done; code != http.StatusOK {
		t.Errorf("the arm after the save answered %d", code)
	}
}

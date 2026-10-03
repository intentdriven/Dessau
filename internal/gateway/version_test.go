package gateway

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
)

// The panel's state carries which version of a model is on disk, by its
// commit, but not the per-file hashes a check compares: the state is
// re-encoded on every event, and the hashes are not what the panel shows.
func TestThePanelStateCarriesTheCommitButNotTheFileHashes(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	const commit = "0123456789abcdef0123456789abcdef01234567"
	if err := a.Registry.Put(registry.Model{ChatTemplate: true,
		RepoID: "org/m", Path: a.Paths.ModelDir("org/m"), State: registry.StateReady,
		Commit:     commit,
		FileHashes: map[string]string{"config.json": "89abcdef0123456789abcdef0123456789abcdef"},
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), commit) {
		t.Error("the panel state does not carry the model's commit")
	}
	if strings.Contains(string(body), "file_hashes") {
		t.Error("the panel state carries every file's hash")
	}
	if m, _ := a.Registry.Get("org/m"); len(m.FileHashes) != 1 {
		t.Error("building the panel's view changed what the registry holds")
	}
}

// Saving any other setting never turns update checks on, and a save that
// turns them on, with an interval, is kept by the next unrelated save
// (itd-2610030857275099 criteria 1 and 10).
func TestAnUnrelatedSaveLeavesTheUpdateCheckAsItWas(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	unrelated := `{"host":"0.0.0.0","port":11535,"api_key":"","decode_concurrency":1,"idle_timeout_sec":0,"context_probe":true}`

	resp := postJSON(t, srv, "/api/settings", unrelated)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if a.Config().UpdateCheck {
		t.Fatal("an unrelated save turned update checks on")
	}

	resp = postJSON(t, srv, "/api/settings",
		`{"host":"0.0.0.0","port":11535,"api_key":"","decode_concurrency":1,"update_check_enabled":true,"update_check_interval_hours":6}`)
	resp.Body.Close()
	if c := a.Config(); !c.UpdateCheck || c.UpdateCheckIntervalHours != 6 {
		t.Fatalf("the save did not turn checks on every 6 hours: %v %d", c.UpdateCheck, c.UpdateCheckIntervalHours)
	}
	resp = postJSON(t, srv, "/api/settings", unrelated)
	resp.Body.Close()
	if c := a.Config(); !c.UpdateCheck || c.UpdateCheckIntervalHours != 6 {
		t.Errorf("an unrelated save changed the update check: %v %d", c.UpdateCheck, c.UpdateCheckIntervalHours)
	}

	resp = postJSON(t, srv, "/api/settings",
		`{"host":"0.0.0.0","port":11535,"api_key":"","decode_concurrency":1,"update_check_interval_hours":721}`)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "update_check_interval_hours") {
		t.Errorf("an interval past thirty days: status %d, %s", resp.StatusCode, body)
	}
}

// The panel's Update is a route of its own: refused for a model with nothing
// Dessau would run to update to, and for one that is not there.
func TestTheUpdateRouteRefusesWhereNoneIsOffered(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	const commit = "0123456789abcdef0123456789abcdef01234567"
	const newer = "fedcba9876543210fedcba9876543210fedcba98"
	if err := a.Registry.Put(registry.Model{ChatTemplate: true,
		RepoID: "org/m", Path: a.Paths.ModelDir("org/m"), State: registry.StateReady, Commit: commit,
		Update: &registry.UpdateCheck{Status: registry.UpdateRunsOwnCode, Commit: newer},
	}); err != nil {
		t.Fatal(err)
	}
	resp := postJSON(t, srv, "/api/models/update", `{"model":"org/m"}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("an update to a version Dessau will not run: status %d, want 409", resp.StatusCode)
	}
	resp = postJSON(t, srv, "/api/models/update", `{"model":"org/absent"}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("an update of a model that is not there: status %d, want 404", resp.StatusCode)
	}
	if len(a.Downloading()) != 0 {
		t.Error("a refused update started a download")
	}
}

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
	if err := a.Registry.Put(registry.Model{
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

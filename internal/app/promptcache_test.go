package app

import (
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/registry"
)

// shapedConfig declares a cache: 2 layers of 2 key-value heads of 128, which
// is 2,048 bytes a token of real cache and 10,240 charged.
const shapedConfig = `{"model_type":"qwen3","max_position_embeddings":40960,
  "num_hidden_layers":2,"num_key_value_heads":2,"head_dim":128}`

const shapedRealPerToken = 2 * 2 * 128 * 2 * 2

// The pool is handed the real cache a token costs beside its charge, so the
// launcher can bound the model server's prompt cache to one served window of
// it (iss-2610071035130302).
func TestTheResolvedModelCarriesTheRealCacheFigure(t *testing.T) {
	a := newBudgetApp(t, 128*gb, captureConfig())
	m := captureModel("org/long")
	m.KVBytesPerToken = 327680 / 5
	putModel(t, a, m)
	resolved, err := modelSource{a}.Resolve("org/long")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.KVBytesPerToken != 327680/5 {
		t.Errorf("the pool resolves %d real bytes a token, want %d", resolved.KVBytesPerToken, 327680/5)
	}
}

// A model carries the figure the moment its download completes, as it
// carries the charge, not only after the next startup rescan.
func TestADownloadRecordsTheRealCacheFigure(t *testing.T) {
	a := newTestApp(t)
	a.Hub.BaseURL = fakeHubWithConfig(t, shapedConfig).URL
	if err := a.Download("org/repo"); err != nil {
		t.Fatalf("Download: %v", err)
	}
	waitFor(t, "the model to become ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready()
	})
	m, _ := a.Registry.Get("org/repo")
	if m.KVChargePerToken != shapedRealPerToken*5 {
		t.Errorf("KVChargePerToken = %d, want %d", m.KVChargePerToken, shapedRealPerToken*5)
	}
	if m.KVBytesPerToken != shapedRealPerToken {
		t.Errorf("KVBytesPerToken = %d, want %d", m.KVBytesPerToken, shapedRealPerToken)
	}
}

// And an update records the new version's figure with the rest of its facts.
func TestAnUpdateRecordsTheRealCacheFigure(t *testing.T) {
	a, h := newStagedApp(t)
	h.set(func(h *versionedHub) {
		h.versions[commitV2]["config.json"] = []byte(shapedConfig)
		h.current = commitV2
	})
	a.Registry.SetUpdate("org/repo", commitV1, registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: commitV2, CheckedAt: time.Now()})
	if err := a.Update("org/repo"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	waitSettled(t, a)
	m, _ := a.Registry.Get("org/repo")
	if !m.Ready() || m.Commit != commitV2 {
		t.Fatalf("after the update: state %s, commit %s", m.State, m.Commit)
	}
	if m.KVBytesPerToken != shapedRealPerToken {
		t.Errorf("KVBytesPerToken = %d, want %d", m.KVBytesPerToken, shapedRealPerToken)
	}
}

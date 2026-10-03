package app

import (
	"context"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/registry"
)

// A failed update says why on the model, as a class the card has words for,
// and the next update that succeeds clears it (iss-2610031320392078).
func TestAFailedUpdateSaysWhyOnTheModel(t *testing.T) {
	for name, tc := range map[string]struct {
		breakIt func(a *App, h *versionedHub)
		class   string
	}{
		"hash mismatch": {func(a *App, h *versionedHub) { h.set(func(h *versionedHub) { h.corrupt = "model.safetensors" }) }, registry.UpdateFailedDownload},
		"mid-download":  {func(a *App, h *versionedHub) { h.set(func(h *versionedHub) { h.failOn = "config.json" }) }, registry.UpdateFailedDownload},
		"ships code": {func(a *App, h *versionedHub) {
			h.set(func(h *versionedHub) {
				h.versions[commitV2]["config.json"] = []byte(`{"model_type":"qwen3","model_file":"x.py"}`)
			})
		}, registry.UpdateFailedRefused},
		"no space": {func(a *App, h *versionedHub) { a.freeSpace = func(string) (int64, bool) { return 3, true } }, registry.UpdateFailedNoSpace},
	} {
		t.Run(name, func(t *testing.T) {
			a, h := newStagedApp(t)
			h.set(func(h *versionedHub) { h.current = commitV2 })
			tc.breakIt(a, h)
			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)
			if m, _ := a.Registry.Get("org/repo"); m.UpdateFailed != tc.class {
				t.Fatalf("UpdateFailed = %q, want %q", m.UpdateFailed, tc.class)
			}

			// Put right, the next update lands and the failure goes with the
			// old record.
			h.set(func(h *versionedHub) {
				h.corrupt, h.failOn = "", ""
				h.versions[commitV2] = v2Files()
			})
			a.freeSpace = func(string) (int64, bool) { return 0, false }
			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)
			m, _ := a.Registry.Get("org/repo")
			if m.Commit != commitV2 || m.UpdateFailed != "" {
				t.Errorf("after a successful update: commit %s, UpdateFailed %q", m.Commit, m.UpdateFailed)
			}
		})
	}
}

// An update the old version's requests outlast says so.
func TestAnUpdateTheRequestsOutlastSaysSo(t *testing.T) {
	a, h := newStagedApp(t)
	a.drainWait = 200 * time.Millisecond
	_, release, err := a.Pool.Acquire(context.Background(), "org/repo")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.UpdateFailed != registry.UpdateFailedBusy {
		t.Errorf("UpdateFailed = %q, want %q", m.UpdateFailed, registry.UpdateFailedBusy)
	}
}

package app

import (
	"os"
	"path/filepath"
	"testing"
)

// A link planted at the staging org folder's name after the update has
// checked it — while the new version downloads — does not carry the clean-up
// of a failed update into the served model's own folder
// (iss-2610031324593822).
func TestALinkReplantedMidUpdateNeverReachesTheServedModel(t *testing.T) {
	a, h := newStagedApp(t)
	before := h.hitsFor("config.json")
	block := make(chan struct{})
	h.set(func(h *versionedHub) {
		h.current = commitV2
		h.block = block
		h.failOn = "config.json"
	})
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	// The download has started: the staging folder for the model is there,
	// checked and opened.
	waitFor(t, "the download to start", func() bool { return h.hitsFor("config.json") > before })
	stagingOrg := filepath.Join(a.Paths.Models, stagingDirName, "org")
	if err := os.Rename(stagingOrg, stagingOrg+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "org"), stagingOrg); err != nil {
		t.Fatal(err)
	}
	close(block)
	waitSettled(t, a)

	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v1Files()) {
		t.Errorf("the served model's folder was reached through the re-planted link: %v", got)
	}
	if m, _ := a.Registry.Get("org/repo"); !m.Ready() || m.Commit != commitV1 {
		t.Errorf("the record changed: state %s, commit %s", m.State, m.Commit)
	}
}

// A link planted at the staging org folder's name just before either swap
// rename — after the last check of that name — pointing at another org's
// folder never moves the other org's model in as this one, and never removes
// either model's files.
func TestALinkPlantedJustBeforeASwapRenameMovesNoOtherModel(t *testing.T) {
	for _, step := range []string{"aside", "in"} {
		t.Run(step, func(t *testing.T) {
			a, h := newStagedApp(t)
			other := filepath.Join(a.Paths.Models, "org2", "repo")
			if err := os.MkdirAll(other, 0o755); err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(other, "config.json"), []byte(`{"model_type":"other"}`), 0o644)
			os.WriteFile(filepath.Join(other, "model.safetensors"), []byte("other-weights"), 0o644)
			otherBefore := filesIn(t, other)
			stagingOrg := filepath.Join(a.Paths.Models, stagingDirName, "org")
			a.beforeSwapRename = func(s string) {
				if s != step {
					return
				}
				if err := os.Rename(stagingOrg, stagingOrg+".moved"); err != nil {
					t.Error(err)
				}
				if err := os.Symlink(filepath.Join("..", "org2"), stagingOrg); err != nil {
					t.Error(err)
				}
			}
			h.set(func(h *versionedHub) { h.current = commitV2 })
			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)

			if got := filesIn(t, other); len(got) != len(otherBefore) || got["config.json"] != otherBefore["config.json"] {
				t.Errorf("the other org's model moved or changed: %v", got)
			}
			served := filesIn(t, a.Paths.ModelDir("org/repo"))
			if served["config.json"] == otherBefore["config.json"] {
				t.Errorf("the other org's model is served as org/repo: %v", served)
			}
			// The old version is never removed by a misdirected swap: it is
			// served, or left aside where nothing removes it.
			if !sameFiles(served, v1Files()) && !sameFiles(served, v2Files()) {
				aside := false
				filepath.WalkDir(a.Paths.Models, func(p string, d os.DirEntry, err error) error {
					if err == nil && !d.IsDir() && filepath.Base(p) == "model-00002-of-00002.safetensors" {
						aside = true
					}
					return nil
				})
				if !aside {
					t.Errorf("the old version is gone and the folder holds neither version whole: %v", served)
				}
			}
		})
	}
}

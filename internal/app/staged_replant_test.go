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

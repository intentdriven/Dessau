package app

import (
	"io"
	"log/slog"
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

// A link planted at the staged version's own name, or at the served model's,
// just before the rename that moves it, is never moved in or aside as though
// it were the folder: no other model is served as this one, and the old
// version is not removed.
func TestALinkAtTheMovedNameIsNeverMovedAsTheFolder(t *testing.T) {
	for _, tc := range []struct {
		step string
		// plant replaces the folder about to be moved with a link.
		plant func(t *testing.T, a *App, other string)
	}{
		// The staged version's own name now links to another org's model.
		{"in", func(t *testing.T, a *App, other string) {
			at := filepath.Join(a.Paths.Models, stagingDirName, "org", "repo")
			if err := os.Rename(at, at+".moved"); err != nil {
				t.Error(err)
			}
			if err := os.Symlink(other, at); err != nil {
				t.Error(err)
			}
		}},
		// The served model's name now links to the real folder, moved where
		// it stays reachable: moving the link aside would leave the update
		// to land with the old version still in place, unrecorded.
		{"aside", func(t *testing.T, a *App, other string) {
			at := a.Paths.ModelDir("org/repo")
			real := filepath.Join(a.Paths.Models, "org3", "repo")
			os.MkdirAll(filepath.Dir(real), 0o755)
			if err := os.Rename(at, real); err != nil {
				t.Error(err)
			}
			if err := os.Symlink(real, at); err != nil {
				t.Error(err)
			}
		}},
	} {
		t.Run(tc.step, func(t *testing.T) {
			a, h := newStagedApp(t)
			other := filepath.Join(a.Paths.Models, "org2", "repo")
			if err := os.MkdirAll(other, 0o755); err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(other, "config.json"), []byte(`{"model_type":"other"}`), 0o644)
			os.WriteFile(filepath.Join(other, "model.safetensors"), []byte("other-weights"), 0o644)
			a.beforeSwapRename = func(s string) {
				if s == tc.step {
					tc.plant(t, a, other)
				}
			}
			h.set(func(h *versionedHub) { h.current = commitV2 })
			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)

			if got := filesIn(t, other); got["config.json"] != `{"model_type":"other"}` {
				t.Errorf("the other org's model changed: %v", got)
			}
			if m, _ := a.Registry.Get("org/repo"); m.Commit == commitV2 {
				t.Error("a link was moved as the model's folder and the update recorded as landed")
			}
			// The real old version is never removed.
			found := false
			filepath.WalkDir(a.Paths.Models, func(p string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() && filepath.Base(p) == "model-00002-of-00002.safetensors" {
					found = true
				}
				return nil
			})
			if !found {
				t.Error("the old version was removed")
			}
		})
	}
}

// An old version a swap left aside, whose model's folder is now taken by
// something else, is kept at the next start rather than removed with the
// rest of the staging folder: it may be the only copy left.
func TestAnAsideCopyThatCannotGoBackIsKeptAtStart(t *testing.T) {
	models := t.TempDir()
	aside := filepath.Join(models, stagingDirName, "org", "repo"+asideSuffix+"abc")
	if err := os.MkdirAll(aside, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(aside, "model.safetensors"), []byte("the only copy"), 0o644)
	staged := filepath.Join(models, stagingDirName, "org", "other")
	os.MkdirAll(staged, 0o755)
	os.WriteFile(filepath.Join(staged, "x"), []byte("staged"), 0o644)
	occupant := filepath.Join(models, "org", "repo")
	os.MkdirAll(occupant, 0o755)
	os.WriteFile(filepath.Join(occupant, "planted"), []byte("not the model"), 0o644)

	recoverStaging(models, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if b, err := os.ReadFile(filepath.Join(aside, "model.safetensors")); err != nil || string(b) != "the only copy" {
		t.Errorf("the copy left aside was removed at start: %v", err)
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Errorf("a staged version was kept at start: %v", err)
	}
}

// The old version a completed swap left aside — the process died before it
// removed it — is removed at start: the model's folder holds a version that
// checks out, so the aside copy is not the last one.
func TestAStaleAsideCopyIsRemovedAtStart(t *testing.T) {
	models := t.TempDir()
	aside := filepath.Join(models, stagingDirName, "org", "repo"+asideSuffix+"abc")
	os.MkdirAll(aside, 0o755)
	os.WriteFile(filepath.Join(aside, "model.safetensors"), []byte("old"), 0o644)
	served := filepath.Join(models, "org", "repo")
	os.MkdirAll(served, 0o755)
	os.WriteFile(filepath.Join(served, "config.json"), []byte(`{"model_type":"qwen3"}`), 0o644)
	os.WriteFile(filepath.Join(served, "model.safetensors"), []byte("new"), 0o644)

	recoverStaging(models, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := os.Stat(aside); !os.IsNotExist(err) {
		t.Errorf("a stale aside copy was kept beside a version that checks out: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(served, "model.safetensors")); string(b) != "new" {
		t.Error("the served version changed")
	}
}

// Of several copies earlier swaps left aside, the newest is the one put back.
func TestTheNewestAsideCopyIsTheOnePutBack(t *testing.T) {
	models := t.TempDir()
	root, err := os.OpenRoot(models)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, attempt := range []string{"zz", "1abc", "zzzzzzzzzzz", "a"} {
		os.MkdirAll(filepath.Join(models, stagingDirName, "org", "repo"+asideSuffix+attempt), 0o755)
	}
	sorg, err := openStagingOrg(root, "org/repo")
	if err != nil {
		t.Fatal(err)
	}
	defer sorg.Close()
	left, ok := asideLeft(sorg, "org/repo")
	if want := "repo" + asideSuffix + "zzzzzzzzzzz"; !ok || filepath.Base(left) != want {
		t.Errorf("asideLeft = %q, %v; want %q", left, ok, want)
	}
}

// At start, of several copies left aside for a model whose folder is
// missing, the newest is put back; the older ones, beside a folder that now
// checks out, go. An aside whose name gives no model can never go back, and
// goes too.
func TestAStartPutsBackTheNewestAsideCopy(t *testing.T) {
	models := t.TempDir()
	write := func(attempt, weights string) {
		dir := filepath.Join(models, stagingDirName, "org", "repo"+asideSuffix+attempt)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"model_type":"qwen3"}`), 0o644)
		os.WriteFile(filepath.Join(dir, "model.safetensors"), []byte(weights), 0o644)
	}
	write("a", "oldest")
	write("zz", "newest")
	write("1b", "older")
	os.MkdirAll(filepath.Join(models, "org"), 0o755)
	junk := filepath.Join(models, stagingDirName, "org", asideSuffix+"x")
	os.MkdirAll(junk, 0o755)

	recoverStaging(models, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if b, _ := os.ReadFile(filepath.Join(models, "org", "repo", "model.safetensors")); string(b) != "newest" {
		t.Errorf("put back %q, want the newest", b)
	}
	if entries, _ := os.ReadDir(filepath.Join(models, stagingDirName, "org")); len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("left in the staging folder: %v", names)
	}
}

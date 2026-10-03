package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

const (
	commitV1 = "1111111111111111111111111111111111111111"
	commitV2 = "2222222222222222222222222222222222222222"
)

// versionedHub serves one repository, org/repo, at whichever commit is
// current, with each commit's own files. Weights are LFS objects (their
// sha256 in the listing); everything else is stored in git (its blob id).
// corrupt names a file whose body is served wrong at the current commit, and
// failOn a file whose request fails.
type versionedHub struct {
	srv *httptest.Server

	mu       sync.Mutex
	current  string
	versions map[string]map[string][]byte
	corrupt  string
	failOn   string
	hits     map[string]int
	block    chan struct{} // when set, file bodies wait for it
}

func newVersionedHub(t *testing.T, versions map[string]map[string][]byte, current string) *versionedHub {
	t.Helper()
	h := &versionedHub{current: current, versions: versions, hits: map[string]int{}}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		cur, files, corrupt, failOn, block := h.current, h.versions[h.current], h.corrupt, h.failOn, h.block
		h.mu.Unlock()
		switch {
		case r.URL.Path == "/api/models/org/repo":
			fmt.Fprintf(w, `{"id":"org/repo","sha":%q}`, cur)
		case strings.HasPrefix(r.URL.Path, "/api/models/org/repo/tree/"):
			commit := strings.TrimPrefix(r.URL.Path, "/api/models/org/repo/tree/")
			type lfs struct {
				OID  string `json:"oid"`
				Size int64  `json:"size"`
			}
			type entry struct {
				Path string `json:"path"`
				Type string `json:"type"`
				Size int64  `json:"size"`
				OID  string `json:"oid"`
				LFS  *lfs   `json:"lfs,omitempty"`
			}
			var out []entry
			for p, b := range h.versions[commit] {
				e := entry{Path: p, Type: "file", Size: int64(len(b)), OID: gitBlobID(b)}
				if strings.HasSuffix(p, ".safetensors") {
					sum := sha256.Sum256(b)
					e.LFS = &lfs{OID: hex.EncodeToString(sum[:]), Size: int64(len(b))}
				}
				out = append(out, e)
			}
			json.NewEncoder(w).Encode(out)
		case strings.HasPrefix(r.URL.Path, "/org/repo/resolve/"+cur+"/"):
			name := strings.TrimPrefix(r.URL.Path, "/org/repo/resolve/"+cur+"/")
			h.mu.Lock()
			h.hits[name]++
			h.mu.Unlock()
			if block != nil {
				select {
				case <-block:
				case <-r.Context().Done():
					return
				}
			}
			if name == failOn {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			b, ok := files[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if name == corrupt {
				b = []byte(strings.Repeat("z", len(b)))
			}
			w.Write(b)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *versionedHub) set(f func(h *versionedHub)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	f(h)
}

func (h *versionedHub) hitsFor(name string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hits[name]
}

func v1Files() map[string][]byte {
	return map[string][]byte{
		"config.json":                      []byte(`{"model_type":"qwen3","max_position_embeddings":40960}`),
		"model-00001-of-00002.safetensors": []byte("weights-one-v1"),
		"model-00002-of-00002.safetensors": []byte("weights-two-v1"),
		"tokenizer.json":                   []byte(`{"v":1}`),
	}
}

// v2Files changes the config at the same length, drops the second shard and
// renames the first: everything a mixed re-download got wrong.
func v2Files() map[string][]byte {
	return map[string][]byte{
		"config.json":       []byte(`{"model_type":"qwen3","max_position_embeddings":40961}`),
		"model.safetensors": []byte("weights-all-v2"),
		"tokenizer.json":    []byte(`{"v":1}`),
	}
}

// precheckLauncher is a launcher whose Precheck applies the model-code rule
// the shipping launcher applies, and that starts nothing.
type precheckLauncher struct{ toolLauncher }

func (l *precheckLauncher) Precheck(spec runtime.Spec) error {
	return registry.CheckModelCode(spec.ModelPath, os.Geteuid())
}

// newStagedApp is an app at v1 of org/repo, downloaded and ready.
func newStagedApp(t *testing.T) (*App, *versionedHub) {
	t.Helper()
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files()}, commitV1)
	l := &precheckLauncher{toolLauncher{servers: map[string]*mlxtest.Server{}}}
	a, err := New(Options{Paths: config.NewPaths(t.TempDir()), Config: config.Default(), Launcher: l})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Hub.BaseURL = h.srv.URL
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "v1 to be ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready() && m.Commit == commitV1
	})
	return a, h
}

// waitSettled waits for the in-flight download of org/repo to finish.
func waitSettled(t *testing.T, a *App) {
	t.Helper()
	waitFor(t, "the update to settle", func() bool { return len(a.Downloading()) == 0 })
}

// filesIn lists the regular files under dir, relative to it.
func filesIn(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		b, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	return out
}

func sameFiles(got map[string]string, want map[string][]byte) bool {
	if len(got) != len(want) {
		return false
	}
	for p, b := range want {
		if got[p] != string(b) {
			return false
		}
	}
	return true
}

// Update fetches the newer version beside the one being served, at exactly
// the marked commit, and swaps it in whole: the directory then holds that
// version's files and no other, the record names its commit, and a file the
// two versions share is linked rather than fetched again (criterion 7).
func TestAnUpdateStagesTheNewVersionAndSwapsItInWhole(t *testing.T) {
	a, h := newStagedApp(t)
	h.set(func(h *versionedHub) { h.current = commitV2 })
	a.Registry.SetUpdate("org/repo", commitV1, registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: commitV2, CheckedAt: time.Now()})

	if err := a.Update("org/repo"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	waitSettled(t, a)
	m, _ := a.Registry.Get("org/repo")
	if !m.Ready() || m.Commit != commitV2 {
		t.Fatalf("after the update: state %s, commit %s", m.State, m.Commit)
	}
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v2Files()) {
		t.Errorf("the model directory holds %v, want exactly the new version", got)
	}
	if h.hitsFor("tokenizer.json") != 1 {
		t.Errorf("a file both versions share was fetched %d times, want once (at the first download)", h.hitsFor("tokenizer.json"))
	}
	if m.Update == nil || m.Update.Status != registry.UpdateCurrent {
		t.Errorf("the mark outlived the update: %+v", m.Update)
	}
	if _, err := os.Stat(a.stagingRoot()); err == nil {
		if entries, _ := os.ReadDir(a.stagingDir("org/repo")); len(entries) > 0 {
			t.Error("the staged version was left behind")
		}
	}
}

// An update is fetched at the commit the check marked, not whatever the
// repository has moved on to since.
func TestAnUpdateFetchesTheMarkedCommit(t *testing.T) {
	a, h := newStagedApp(t)
	const v3 = "3333333333333333333333333333333333333333"
	h.set(func(h *versionedHub) {
		h.versions[v3] = map[string][]byte{"config.json": []byte(`{"model_type":"qwen3","max_position_embeddings":1}`), "model.safetensors": []byte("v3")}
		h.current = commitV2
	})
	a.Registry.SetUpdate("org/repo", commitV1, registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: commitV2, CheckedAt: time.Now()})
	if err := a.Update("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV2 {
		t.Errorf("commit = %s, want the marked %s", m.Commit, commitV2)
	}
}

// Every way an update can fail leaves the old version serving, its files and
// its record untouched, and the staged copy gone: a file that is not what the
// listing says, a request that fails mid-download, a version Dessau would not
// start, and a disk with no room for it (criterion 7).
func TestAFailedUpdateLeavesTheOldVersionServing(t *testing.T) {
	for name, breakIt := range map[string]func(a *App, h *versionedHub){
		"hash mismatch": func(a *App, h *versionedHub) { h.set(func(h *versionedHub) { h.corrupt = "model.safetensors" }) },
		"mid-download":  func(a *App, h *versionedHub) { h.set(func(h *versionedHub) { h.failOn = "config.json" }) },
		"ships code": func(a *App, h *versionedHub) {
			h.set(func(h *versionedHub) {
				h.versions[commitV2]["config.json"] = []byte(`{"model_type":"qwen3","model_file":"x.py"}`)
			})
		},
		"no space": func(a *App, h *versionedHub) { a.freeSpace = func(string) (int64, bool) { return 3, true } },
	} {
		t.Run(name, func(t *testing.T) {
			a, h := newStagedApp(t)
			before, _ := a.Registry.Get("org/repo")
			h.set(func(h *versionedHub) { h.current = commitV2 })
			breakIt(a, h)

			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)
			m, _ := a.Registry.Get("org/repo")
			if !m.Ready() || m.Commit != commitV1 || len(m.FileHashes) != len(before.FileHashes) {
				t.Errorf("the record changed: state %s, commit %s", m.State, m.Commit)
			}
			if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v1Files()) {
				t.Errorf("the served files changed: %v", got)
			}
			if entries, _ := os.ReadDir(a.stagingDir("org/repo")); len(entries) > 0 {
				t.Error("a failed update left its staged copy behind")
			}
		})
	}
}

// While the new version downloads, the old one stays ready and is still what
// a load resolves to: a re-download no longer takes a model out of service.
func TestAModelStaysReadyWhileItsUpdateDownloads(t *testing.T) {
	a, h := newStagedApp(t)
	block := make(chan struct{})
	h.set(func(h *versionedHub) { h.current = commitV2; h.block = block })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the update to reach the files", func() bool { return h.hitsFor("config.json") > 1 })
	m, _ := a.Registry.Get("org/repo")
	if !m.Ready() || m.Commit != commitV1 {
		t.Errorf("during the update: state %s, commit %s", m.State, m.Commit)
	}
	if _, err := (modelSource{a}).Resolve("org/repo"); err != nil {
		t.Errorf("the model being updated cannot be loaded: %v", err)
	}
	close(block)
	waitSettled(t, a)
}

// The swap waits for a request in flight to finish — Carol's answer is never
// cut off — and refuses new loads while it moves the directories. A request
// that outlasts the wait abandons the update and the old version keeps
// serving.
func TestTheSwapDrainsTheModelFirst(t *testing.T) {
	a, h := newStagedApp(t)
	_, release, err := a.Pool.Acquire(context.Background(), "org/repo")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	a.drainWait = 300 * time.Millisecond
	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV1 {
		t.Errorf("the swap went ahead under a request in flight: commit %s", m.Commit)
	}
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v1Files()) {
		t.Errorf("the served files changed under a request in flight: %v", got)
	}

	a.drainWait = 10 * time.Second
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the swap to wait on the request", func() bool { return a.isSwapping("org/repo") })
	if _, err := (modelSource{a}).Resolve("org/repo"); err == nil || !strings.Contains(err.Error(), "being updated") {
		t.Errorf("a load during the swap: err = %v, want it refused", err)
	}
	release()
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV2 {
		t.Errorf("the swap did not go ahead once the request finished: commit %s", m.Commit)
	}
}

// A re-download of a ready model takes the same staged path, so a repository
// that changed upstream — a config of the same length, a shard renamed and
// one dropped — leaves exactly the new version, never a mix
// (iss-2610030913179523).
func TestARedownloadNeverLeavesAMixedModel(t *testing.T) {
	a, h := newStagedApp(t)
	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v2Files()) {
		t.Errorf("the re-download left %v, want exactly the new version", got)
	}
}

// What was learned about the old files goes with them; what the operator set
// for the model stays.
func TestAnUpdateClearsFileFactsAndKeepsSettings(t *testing.T) {
	a, h := newStagedApp(t)
	cfg := a.Config()
	cfg.Models = map[string]config.ModelSettings{"org/repo": {Pinned: true}}
	if err := a.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	m, _ := a.Registry.Get("org/repo")
	m.ToolCalling = &registry.ToolCalling{Can: true, At: time.Now().Unix()}
	m.LoadFailure = &registry.LoadFailure{Reason: "it did not start", At: time.Now().Unix()}
	a.Registry.Put(m)

	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	m, _ = a.Registry.Get("org/repo")
	if m.Commit != commitV2 || m.ToolCalling != nil || m.LoadFailure != nil || m.Measured != nil {
		t.Errorf("facts about the old files outlived them: %+v", m)
	}
	if !a.Config().Models["org/repo"].Pinned {
		t.Error("the pin did not survive the update")
	}
}

// Update offers nothing for a version Dessau would not run, or one waiting
// for review.
func TestUpdateIsRefusedWhereNoneIsOffered(t *testing.T) {
	a, _ := newStagedApp(t)
	for _, status := range []string{registry.UpdateRunsOwnCode, registry.UpdateAwaitingReview} {
		a.Registry.SetUpdate("org/repo", commitV1, registry.UpdateCheck{Status: status, Commit: commitV2, CheckedAt: time.Now()})
		if err := a.Update("org/repo"); !errors.Is(err, ErrUpdateNotOffered) {
			t.Errorf("%s: err = %v, want ErrUpdateNotOffered", status, err)
		}
	}
}

// A swap the previous process died in the middle of is put right at the next
// start: an old version left aside with nothing in its place goes back, and
// staged copies are cleared.
func TestAnInterruptedSwapIsPutRightAtStart(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(paths.Models, stagingDirName)
	aside := filepath.Join(staging, "org", "lost"+asideSuffix)
	os.MkdirAll(aside, 0o755)
	os.WriteFile(filepath.Join(aside, "config.json"), []byte(`{"model_type":"qwen3"}`), 0o644)
	os.WriteFile(filepath.Join(aside, "model.safetensors"), []byte("w"), 0o644)
	os.MkdirAll(filepath.Join(staging, "org", "half"), 0o755)
	os.WriteFile(filepath.Join(staging, "org", "half", "model.safetensors.dessau-part"), []byte("w"), 0o644)

	a, err := New(Options{Paths: paths, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if _, err := os.Stat(filepath.Join(paths.Models, "org", "lost", "model.safetensors")); err != nil {
		t.Errorf("the old version left aside was not put back: %v", err)
	}
	if m, err := a.Registry.Get("org/lost"); err != nil || !m.Ready() {
		t.Errorf("the model put back is not served: %v", err)
	}
	if _, err := os.Stat(staging); err == nil {
		t.Error("the staging folder survived the start")
	}
}

// A decision model is updated only to a version a Dessau release reviewed,
// whatever the registry's mark says: a mark is a record, and is asked again
// before anything is fetched (review of step 2).
func TestADecisionModelIsUpdatedOnlyToAReviewedVersion(t *testing.T) {
	a, h := newStagedApp(t)
	h.set(func(h *versionedHub) { h.current = commitV2 })
	a.reviewed = func(string) (bool, []string) { return true, []string{commitV1} }
	a.Registry.SetUpdate("org/repo", commitV1, registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: commitV2, CheckedAt: time.Now()})
	if err := a.Update("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV1 {
		t.Errorf("a decision model was updated to an unreviewed version: %s", m.Commit)
	}
	if h.hitsFor("model.safetensors") != 0 {
		t.Error("an unreviewed version was fetched")
	}

	a.reviewed = func(string) (bool, []string) { return true, []string{commitV1, commitV2} }
	if err := a.Update("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if m, _ := a.Registry.Get("org/repo"); m.Commit != commitV2 {
		t.Errorf("a reviewed version was not taken: %s", m.Commit)
	}
}

// While a newer version is fetched beside a ready model, how far it has come
// is told on its own, for the card; the registry's figure stays the served
// version's.
func TestAnUpdatesProgressIsToldForTheCard(t *testing.T) {
	a, h := newStagedApp(t)
	if _, ok := a.UpdateProgress("org/repo"); ok {
		t.Error("a model with no update under way reports one")
	}
	block := make(chan struct{})
	h.set(func(h *versionedHub) { h.current = commitV2; h.block = block })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the update to start", func() bool { _, ok := a.UpdateProgress("org/repo"); return ok })
	if m, _ := a.Registry.Get("org/repo"); m.Progress != 100 {
		t.Errorf("the served version's progress moved to %v", m.Progress)
	}
	close(block)
	waitSettled(t, a)
	if _, ok := a.UpdateProgress("org/repo"); ok {
		t.Error("a finished update still reports progress")
	}
}

// A link planted in the staging folder is never followed: not by the
// removal that clears the folder before a download, not by the swap, and not
// by the recovery at start. Whatever it names, inside the models folder or
// outside it, is untouched (adversarial review of step 3).
func TestALinkPlantedInTheStagingFolderIsNeverFollowed(t *testing.T) {
	for name, plant := range map[string]func(t *testing.T, a *App, outside string){
		"the org inside staging links outside": func(t *testing.T, a *App, outside string) {
			os.MkdirAll(filepath.Join(a.Paths.Models, stagingDirName), 0o755)
			if err := os.Symlink(outside, filepath.Join(a.Paths.Models, stagingDirName, "org")); err != nil {
				t.Fatal(err)
			}
		},
		"the staging folder links outside": func(t *testing.T, a *App, outside string) {
			if err := os.Symlink(filepath.Dir(outside), filepath.Join(a.Paths.Models, stagingDirName)); err != nil {
				t.Fatal(err)
			}
			os.MkdirAll(filepath.Join(filepath.Dir(outside), "org"), 0o755)
			os.MkdirAll(filepath.Join(filepath.Dir(outside), "org", "repo"), 0o755)
			os.WriteFile(filepath.Join(filepath.Dir(outside), "org", "repo", "precious.txt"), []byte("keep"), 0o644)
		},
		"the org inside staging links to the served model's org": func(t *testing.T, a *App, outside string) {
			os.MkdirAll(filepath.Join(a.Paths.Models, stagingDirName), 0o755)
			if err := os.Symlink(filepath.Join(a.Paths.Models, "org"), filepath.Join(a.Paths.Models, stagingDirName, "org")); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			a, h := newStagedApp(t)
			outside := filepath.Join(t.TempDir(), "victim")
			os.MkdirAll(filepath.Join(outside, "repo"), 0o755)
			os.WriteFile(filepath.Join(outside, "repo", "precious.txt"), []byte("keep"), 0o644)
			plant(t, a, outside)

			h.set(func(h *versionedHub) { h.current = commitV2 })
			if err := a.Download("org/repo"); err != nil {
				t.Fatal(err)
			}
			waitSettled(t, a)
			if b, err := os.ReadFile(filepath.Join(outside, "repo", "precious.txt")); err != nil || string(b) != "keep" {
				t.Errorf("a file the planted link names was touched: %v", err)
			}
			if b, err := os.ReadFile(filepath.Join(filepath.Dir(outside), "org", "repo", "precious.txt")); err == nil && string(b) != "keep" {
				t.Error("a file the planted staging link names was changed")
			}
			// Whatever happened to the update, the model is served whole.
			got := filesIn(t, a.Paths.ModelDir("org/repo"))
			if !sameFiles(got, v1Files()) && !sameFiles(got, v2Files()) {
				t.Errorf("the model folder holds neither version whole: %v", got)
			}
			if m, _ := a.Registry.Get("org/repo"); !m.Ready() {
				t.Error("the model is no longer ready")
			}
		})
	}
}

// A retry never clears the only copy of a model: when a swap's folder is
// missing and the copy left aside is the one remaining, it is put back.
func TestARetryNeverClearsTheOnlyCopy(t *testing.T) {
	a, h := newStagedApp(t)
	aside := filepath.Join(a.Paths.Models, stagingDirName, "org", "repo"+asideSuffix)
	os.MkdirAll(filepath.Dir(aside), 0o755)
	if err := os.Rename(a.Paths.ModelDir("org/repo"), aside); err != nil {
		t.Fatal(err)
	}
	h.set(func(h *versionedHub) { h.current = commitV2 })
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	if got := filesIn(t, a.Paths.ModelDir("org/repo")); !sameFiles(got, v1Files()) {
		t.Errorf("the copy left aside was not put back: %v", got)
	}
}

// A first download retried at another commit keeps none of the earlier
// attempt's files the new version does not name: a shard left over would be
// loaded beside the new ones (iss-2610030913179523).
func TestARetriedFirstDownloadKeepsNoStaleFiles(t *testing.T) {
	h := newVersionedHub(t, map[string]map[string][]byte{commitV1: v1Files(), commitV2: v2Files()}, commitV2)
	a, err := New(Options{Paths: config.NewPaths(t.TempDir()), Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Hub.BaseURL = h.srv.URL
	dir := a.Paths.ModelDir("org/repo")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "model-00002-of-00002.safetensors"), []byte("left by an attempt at v1"), 0o644)
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the model to be ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready()
	})
	if got := filesIn(t, dir); !sameFiles(got, v2Files()) {
		t.Errorf("the model folder holds %v, want exactly the version downloaded", got)
	}
}

// The swapping mark is lifted in the same step as the new record is written:
// once loads are admitted again, the record describes the new files.
func TestTheNewRecordLandsBeforeLoadsResume(t *testing.T) {
	a, h := newStagedApp(t)
	h.set(func(h *versionedHub) { h.current = commitV2 })
	stop := make(chan struct{})
	var bad []string
	var mu sync.Mutex
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if !a.isSwapping("org/repo") {
				m, _ := a.Registry.Get("org/repo")
				b, _ := os.ReadFile(filepath.Join(a.Paths.ModelDir("org/repo"), "config.json"))
				if strings.Contains(string(b), "40961") && m.Commit != commitV2 {
					mu.Lock()
					bad = append(bad, m.Commit)
					mu.Unlock()
				}
			}
		}
	}()
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitSettled(t, a)
	close(stop)
	mu.Lock()
	defer mu.Unlock()
	if len(bad) > 0 {
		t.Errorf("loads were admitted while the new files carried the old record (%d observations)", len(bad))
	}
}

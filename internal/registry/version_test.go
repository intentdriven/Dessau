package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	aCommit  = "0123456789abcdef0123456789abcdef01234567"
	aBlobID  = "89abcdef0123456789abcdef0123456789abcdef"
	aLFSHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// The version a download fetched survives a restart: the commit and each
// file's hash are written to the index and read back unchanged.
func TestTheDownloadedVersionIsRecordedAndReadBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"config.json": aBlobID, "model.safetensors": aLFSHash}
	if err := r.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit, FileHashes: files}); err != nil {
		t.Fatal(err)
	}
	files["config.json"] = "changed after the Put"

	r2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := r2.Get("org/m")
	if err != nil {
		t.Fatal(err)
	}
	if !m.VersionKnown() {
		t.Fatal("a model downloaded at a recorded commit reads as version unknown")
	}
	if m.Commit != aCommit {
		t.Errorf("Commit = %q, want %q", m.Commit, aCommit)
	}
	if got := m.FileHashes["config.json"]; got != aBlobID {
		t.Errorf("config.json reads back as %q, want %q: the registry must hold its own copy", got, aBlobID)
	}
	if got := m.FileHashes["model.safetensors"]; got != aLFSHash {
		t.Errorf("model.safetensors reads back as %q, want %q", got, aLFSHash)
	}
}

// A model whose version Dessau never recorded — everything downloaded before
// versions were, and every directory a rescan adopts — reads as version
// unknown rather than as some version.
func TestAModelWithNoRecordedCommitIsVersionUnknown(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/old", State: StateReady}); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/old")
	if m.VersionKnown() {
		t.Error("a model with no recorded commit claims a known version")
	}

	models := t.TempDir()
	writeModelDir(t, models, "org", "adopted", 64)
	if err := r.Rescan(models); err != nil {
		t.Fatal(err)
	}
	m, err = r.Get("org/adopted")
	if err != nil {
		t.Fatal(err)
	}
	if m.VersionKnown() {
		t.Error("a directory the rescan adopted claims a known version")
	}
}

// A rescan re-derives what it can read from the directory and keeps what it
// cannot: the version came from the Hub, not from the disk, so a restart must
// not forget it.
func TestARescanKeepsTheRecordedVersion(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	models := t.TempDir()
	writeModelDir(t, models, "org", "m", 64)
	if err := r.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit, FileHashes: map[string]string{"config.json": aBlobID}}); err != nil {
		t.Fatal(err)
	}
	if err := r.Rescan(models); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/m")
	if m.Commit != aCommit || m.FileHashes["config.json"] != aBlobID {
		t.Errorf("the rescan lost the recorded version: %q %v", m.Commit, m.FileHashes)
	}
}

// registry.json can be edited by hand or left corrupt, and the version is
// published to the panel and compared with the Hub's answer, so what is read
// back is held to the shapes a download writes: a commit id, and hashes of
// files with plain relative paths. A commit that is not one is cleared with
// its hashes — the model is version unknown, which offers Update — and a bad
// hash entry is dropped on its own.
func TestAPlantedVersionIsHeldToTheShapesADownloadWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	planted := `[
	  {"repo_id":"org/badcommit","state":"ready","commit":"main","file_hashes":{"config.json":"` + aBlobID + `"}},
	  {"repo_id":"org/badfiles","state":"ready","commit":"` + aCommit + `","file_hashes":{
	    "config.json":"` + aBlobID + `",
	    "../../escape":"` + aBlobID + `",
	    "/abs":"` + aBlobID + `",
	    "weights.safetensors":"<script>",
	    "":"` + aBlobID + `",
	    "tokenizer.json":"` + strings.ToUpper(aBlobID) + `"}}
	]`
	if err := os.WriteFile(path, []byte(planted), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/badcommit")
	if m.VersionKnown() || m.FileHashes != nil {
		t.Errorf("a commit that is not one was kept: %q %v", m.Commit, m.FileHashes)
	}
	m, _ = r.Get("org/badfiles")
	if m.Commit != aCommit {
		t.Errorf("a good commit was lost beside bad entries: %q", m.Commit)
	}
	if len(m.FileHashes) != 1 || m.FileHashes["config.json"] != aBlobID {
		t.Errorf("FileHashes = %v, want only the well-formed config.json entry", m.FileHashes)
	}
}

// The index is read whole and bounded, so one repository listing tens of
// thousands of files must not be able to fill it: past the bound the version
// is kept as its commit alone, which is still a version.
func TestAVersionWithTooManyFilesKeepsItsCommitAlone(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for i := 0; i <= MaxVersionFiles; i++ {
		files[fmt.Sprintf("shard-%05d.safetensors", i)] = aLFSHash
	}
	if err := r.Put(Model{RepoID: "org/many", State: StateReady, Commit: aCommit, FileHashes: files}); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/many")
	if m.Commit != aCommit {
		t.Errorf("Commit = %q, want it kept", m.Commit)
	}
	if m.FileHashes != nil {
		t.Errorf("%d file hashes were recorded past the bound of %d", len(m.FileHashes), MaxVersionFiles)
	}
}

// However many hostile versions are recorded, the index still opens: each is
// bounded far below the read limit, and a name the index would have to
// escape is not recorded at all (adversarial review of step 1).
func TestHostileVersionsCannotFillTheIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("a", MaxVersionPathBytes-12)
	for n := 0; n < 40; n++ {
		files := map[string]string{}
		for i := 0; i < MaxVersionFiles/2; i++ {
			files[fmt.Sprintf("%04d<&>%s.json", i, long)] = aLFSHash
			files[fmt.Sprintf("%04d-%s", i, long)] = aLFSHash
		}
		if err := r.Put(Model{RepoID: fmt.Sprintf("org/m%02d", n), State: StateReady, Commit: aCommit, FileHashes: files}); err != nil {
			t.Fatal(err)
		}
		// And a version just inside the per-model bound, which is recorded.
		files = map[string]string{}
		for i := 0; (i+1)*(len(long)+16+64) <= MaxVersionBytes; i++ {
			files[fmt.Sprintf("%04d-%s.json", i, long)] = aLFSHash
		}
		if err := r.Put(Model{RepoID: fmt.Sprintf("org/k%02d", n), State: StateReady, Commit: aCommit, FileHashes: files}); err != nil {
			t.Fatal(err)
		}
		if m, _ := r.Get(fmt.Sprintf("org/k%02d", n)); len(m.FileHashes) == 0 {
			t.Fatal("a version inside the bound was not recorded")
		}
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() > maxRegistryBytes/4 {
		t.Errorf("eighty versions cost %d bytes of an index read whole under %d", fi.Size(), maxRegistryBytes)
	}
	if _, err := Open(path); err != nil {
		t.Fatalf("the index no longer opens: %v", err)
	}
	m, _ := r.Get("org/m00")
	if m.Commit != aCommit || m.FileHashes != nil {
		t.Errorf("a version past the bound should keep its commit alone, got %q and %d hashes", m.Commit, len(m.FileHashes))
	}
}

// A name the index would have to escape is not recorded.
func TestAVersionRecordsOnlyPlainNames(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"model-00001-of-00002.safetensors": aLFSHash,
		"sub/dir/tokenizer_config.json":    aBlobID,
		"weird<name>.json":                 aBlobID,
		"amp&.json":                        aBlobID,
		"spaced name.json":                 aBlobID,
		"naïve.json":                       aBlobID,
	}
	if err := r.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit, FileHashes: files}); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/m")
	if len(m.FileHashes) != 2 || m.FileHashes["sub/dir/tokenizer_config.json"] != aBlobID {
		t.Errorf("FileHashes = %v, want the two plain names only", m.FileHashes)
	}
}

// What a check found is recorded on the model and read back, and only
// against the version it compared: a check that finished after the model was
// downloaded again describes files that are no longer there.
func TestACheckIsRecordedAgainstTheVersionItCompared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit}); err != nil {
		t.Fatal(err)
	}
	const newer = "fedcba9876543210fedcba9876543210fedcba98"
	at := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	if err := r.SetUpdate("org/m", aCommit, UpdateCheck{Status: UpdateAvailable, Commit: newer, CheckedAt: at}); err != nil {
		t.Fatal(err)
	}
	r2, _ := Open(path)
	m, _ := r2.Get("org/m")
	if m.Update == nil || m.Update.Status != UpdateAvailable || m.Update.Commit != newer || !m.Update.CheckedAt.Equal(at) {
		t.Fatalf("Update = %+v", m.Update)
	}
	if err := r.SetUpdate("org/m", newer, UpdateCheck{Status: UpdateCurrent, CheckedAt: at}); !errors.Is(err, ErrVersionMoved) {
		t.Errorf("a check of another version was recorded: err = %v", err)
	}
	if err := r.SetUpdate("org/absent", aCommit, UpdateCheck{Status: UpdateCurrent, CheckedAt: at}); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// A planted check record is held to what a check writes: a status this build
// does not know, or a newer version that is not a commit, is dropped; and a
// check time in the future — which would put the next check off for as long
// as it says — is not believed.
func TestAPlantedCheckIsHeldToWhatACheckWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	planted := `[
	  {"repo_id":"org/a","state":"ready","commit":"` + aCommit + `","update":{"status":"<b>new</b>","commit":"` + aCommit + `","checked_at":"2026-10-03T12:00:00Z"}},
	  {"repo_id":"org/b","state":"ready","commit":"` + aCommit + `","update":{"status":"available","commit":"main","checked_at":"2026-10-03T12:00:00Z"}},
	  {"repo_id":"org/c","state":"ready","commit":"` + aCommit + `","update":{"status":"current","checked_at":"2999-01-01T00:00:00Z"}},
	  {"repo_id":"org/d","state":"ready","update":{"status":"current","checked_at":"2026-10-03T12:00:00Z"}}
	]`
	if err := os.WriteFile(path, []byte(planted), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"org/a", "org/b", "org/d"} {
		if m, _ := r.Get(id); m.Update != nil {
			t.Errorf("%s: a planted check was kept: %+v", id, m.Update)
		}
	}
	if m, _ := r.Get("org/c"); m.Update != nil && m.Update.CheckedAt.After(time.Now().Add(48*time.Hour)) {
		t.Errorf("a check time in the future was believed: %v", m.Update.CheckedAt)
	}
}

// The rule the model server loads code by, applied to the bytes of a
// config.json a check fetched rather than to a file on disk.
func TestAConfigThatNamesModelCodeIsRecognisedFromItsBytes(t *testing.T) {
	for body, want := range map[string]bool{
		`{"model_type":"qwen3"}`:                  false,
		`{"model_file":null}`:                     false,
		`{"model_file":"modeling.py"}`:            true,
		`{"model_file":""}`:                       true,
		`{"Model_File":"x.py"}`:                   false,
		`{"nested":{"model_file":"x.py"}}`:        false,
		`{"model_file":"x.py","model_file":null}`: false,
		`{"model_file":null,"model_file":"x.py"}`: true,
	} {
		got, err := ConfigNamesModelCode([]byte(body))
		if err != nil || got != want {
			t.Errorf("ConfigNamesModelCode(%s) = %v, %v; want %v", body, got, err, want)
		}
	}
	if _, err := ConfigNamesModelCode([]byte(`[1,2]`)); err == nil {
		t.Error("a config that is not an object was read as one")
	}
}

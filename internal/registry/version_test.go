package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

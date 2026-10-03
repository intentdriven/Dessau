package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A failed update's reason is recorded on the model as a class, read back
// after a restart, and cleared by the next successful download, whose fresh
// record carries none (iss-2610031320392078).
func TestAFailedUpdateIsRecordedAndClearedByTheNextSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetUpdateFailure("Org/M", UpdateFailedNoSpace); err != nil {
		t.Fatal(err)
	}
	r2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r2.Get("org/m"); m.UpdateFailed != UpdateFailedNoSpace {
		t.Fatalf("UpdateFailed = %q after a restart, want %q", m.UpdateFailed, UpdateFailedNoSpace)
	}
	if err := r2.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit}); err != nil {
		t.Fatal(err)
	}
	if m, _ := r2.Get("org/m"); m.UpdateFailed != "" {
		t.Errorf("a successful download kept the failure %q", m.UpdateFailed)
	}
	if err := r2.SetUpdateFailure("org/m", "the disk said /Users/alice is full"); err == nil {
		t.Error("a reason that is not one of the classes was recorded")
	}
	if err := r2.SetUpdateFailure("org/absent", UpdateFailedDownload); err == nil {
		t.Error("a failure was recorded for a model that is not there")
	}
}

// A class this build does not know, written into registry.json by hand, is
// dropped when the file is read: the panel shows only a class's own words.
func TestAPlantedUpdateFailureIsDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/m", State: StateReady, Commit: aCommit}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetUpdateFailure("org/m", UpdateFailedBusy); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	planted := strings.Replace(string(raw), `"update_failed": "busy"`, `"update_failed": "<img src=x>"`, 1)
	if planted == string(raw) {
		t.Fatalf("the record does not carry update_failed as expected:\n%s", raw)
	}
	if err := os.WriteFile(path, []byte(planted), 0o600); err != nil {
		t.Fatal(err)
	}
	r2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r2.Get("org/m"); m.UpdateFailed != "" {
		t.Errorf("a planted class was kept: %q", m.UpdateFailed)
	}
}

package selftest

import (
	"os"
	"path/filepath"
	"testing"
)

// The results file is written by the same primitive as Gropius's own log and
// its statistics store (internal/applog), so it refuses what they refuse. A
// link under the file's name is the older half of that and was already held;
// a file this account's own writers could not have created is the half the
// three writers had drifted apart on (iss-2609091714393599), and it is held
// here the way internal/archtest holds it for the other two.
func TestAPlantedResultsFileIsRefused(t *testing.T) {
	run := Run{Kind: KindRun, Model: "org/a", Outcome: OutcomeOK}

	t.Run("a link to somewhere else", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(target, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(dir, FileName)); err != nil {
			t.Fatal(err)
		}
		f := &file{path: filepath.Join(dir, FileName), maxBytes: DefaultMaxBytes, log: testLogger()}
		f.write(run)
		b, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) != 0 {
			t.Errorf("%d bytes of a result were written through a planted link", len(b))
		}
	})

	t.Run("a file another account could read", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, FileName)
		const planted = "{\"kind\":\"run\",\"model\":\"planted\"}\n"
		if err := os.WriteFile(path, []byte(planted), 0o600); err != nil {
			t.Fatal(err)
		}
		// Not a mode this account's own writers can produce: they create at
		// 0600 and a umask only takes bits away.
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
		f := &file{path: path, maxBytes: DefaultMaxBytes, log: testLogger()}
		f.write(run)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != planted {
			t.Errorf("a result was appended to a file the self-test did not make (mode 0644): %q", string(b))
		}
	})
}

// The cap is held by rotation rather than by truncation now, and what a reader
// sees is unchanged: one file, under the cap, holding the newest runs.
func TestTheResultsFileIsStartedAgainAtItsCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	f := &file{path: path, maxBytes: 400, log: testLogger()}
	t.Cleanup(f.close)
	for i := range 40 {
		f.write(Run{Kind: KindRun, Model: "org/a", At: int64(i), Outcome: OutcomeOK})
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 400 {
		t.Errorf("the results file is %d bytes, past its 400-byte cap", info.Size())
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("the results file is mode %04o, want 0600", info.Mode().Perm())
	}
	// No numbered predecessors: the self-test keeps one file, as its reference
	// page says. Rotation with nothing kept is what starts it again.
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != FileName {
		t.Errorf("the results directory holds %v, want %s alone", ents, FileName)
	}
	runs, err := ReadResults(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) == 0 {
		t.Fatal("the results file holds nothing after being started again")
	}
	if runs[len(runs)-1].At != 39 {
		t.Errorf("the newest run is at %d, want the last one written", runs[len(runs)-1].At)
	}
}

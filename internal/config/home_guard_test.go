package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// A test binary never resolves this account's own Dessau directory: a test
// that would have read or written the real ~/Library is refused with the
// reason, and one that points HOME at a directory of its own gets that
// directory (iss-2609200823589977). The lifecycle tests wrote into the real
// ~/Library until PR 83 gave them a fixture; this is what catches the next one.
func TestATestCannotResolveTheRealAccountDirectory(t *testing.T) {
	t.Setenv("HOME", startHome)
	if dir, err := InstalledRoot(); err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Errorf("InstalledRoot under the HOME this test binary started with = %q, %v; want a refusal naming HOME", dir, err)
	}
	if dir, err := AccountHome(); err == nil {
		t.Errorf("AccountHome under the starting HOME = %q, want a refusal", dir)
	}

	fake := t.TempDir()
	t.Setenv("HOME", fake)
	dir, err := InstalledRoot()
	if err != nil {
		t.Fatalf("InstalledRoot under a HOME the test chose: %v", err)
	}
	if want := filepath.Join(fake, "Library", "Application Support", "Dessau"); dir != want {
		t.Errorf("InstalledRoot = %q, want %q", dir, want)
	}
}

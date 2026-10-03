package archtest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Dessau serves from one macOS account, and its data root is that account's
// own: ~/Library/Application Support/Dessau, or wherever DESSAU_ROOT points.
// No code path selects a machine-wide root that several accounts write to,
// and nothing in the tree offers to make one. Every other account reaches the
// server over the network and never needs the model files.
//
// The machine-wide root was a mode of its own once, with an installer target,
// a directory mode, an adoption path for settings and a branch in uninstall.
// This scan is what keeps it gone: a revival has to delete this test in a diff
// somebody reviews, rather than reappear one comment or one target at a time.
//
// The subject is everything the repository says in the present tense — code,
// tests, scripts, the Makefile, the docs and the README. CHANGELOG.md is
// history and is left out, as are the records under .abcd/, which the walker
// skips with every other dot-directory.
func TestNoMachineWideDataRoot(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	self := filepath.Join(root, "internal", "archtest", "one_account_root_test.go")
	// The one file that names the system directory on purpose, and why.
	allowed := map[string]string{
		"internal/archtest/arch_test.go": "/Users/" + "Shared",
	}

	// Spelled so that this file does not match its own scan.
	markers := []string{
		"/Users/" + "Shared",
		"install" + "-shared",
		"Shared" + "Root",
		"AdoptShared" + "Config",
		"shared" + "-cache",
		"shared" + " cache",
		"shared" + " root",
	}
	scanned := map[string]bool{
		".go": true, ".md": true, ".sh": true, ".swift": true,
		".html": true, ".js": true, ".css": true, ".plist": true,
	}

	walkRepoFiles(t, root, walkOptions{}, func(path string, d fs.DirEntry) error {
		if path == self {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "CHANGELOG.md" {
			return nil
		}
		if !scanned[filepath.Ext(path)] && d.Name() != "Makefile" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(b))
		for _, m := range markers {
			// arch_test.go pins the absolute-path hook's exemption for the
			// system directory, which the changelog's history still names.
			if allowed[rel] == m {
				continue
			}
			if i := strings.Index(lower, strings.ToLower(m)); i >= 0 {
				line := 1 + strings.Count(lower[:i], "\n")
				t.Errorf("%s:%d mentions %q — Dessau serves from one account and keeps no machine-wide data root", rel, line, m)
			}
		}
		return nil
	})
}

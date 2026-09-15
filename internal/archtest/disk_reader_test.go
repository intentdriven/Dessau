package archtest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Free disk has one reader, in the capability package, and every figure the
// panel shows — the search tab's note, the Models tab's roll-up — comes from
// it through capability.Assess. A second statfs anywhere else would be a
// second answer to how much room the downloads have (itd-2609091903463596).
func TestFreeDiskHasOneReader(t *testing.T) {
	root := repoRootDir(t)
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(filepath.ToSlash(rel), "internal/capability/") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "Statfs") {
			t.Errorf("%s reads the disk itself; free disk is read once, in internal/capability", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

package archtest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The served window has one home, App.ServedWindow in internal/app: the
// operator's setting when they have made one, and otherwise the largest
// window that fits the memory budget at the concurrency in force, capped at
// what the model declares. The derivation needs the budget, the concurrency,
// the weights and the cache charge together, and only the app holds them —
// so the reader of the explicit setting, config.Config.ServedContextSetting,
// is called from internal/app alone, and nobody else turns a declared window
// into a served one. A second reader is how a fresh install came to charge
// every model its declared window while the panel showed the same figure as
// the default (iss-2609202048576967).
func TestTheServedWindowHasOneHome(t *testing.T) {
	root := repoRootDir(t)
	walkRepoFiles(t, root, walkOptions{}, func(path string, d fs.DirEntry) error {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "internal/app/") || strings.HasPrefix(rel, "internal/config/") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), ".ServedContextSetting(") {
			t.Errorf("%s reads the served-context setting itself; the served window is resolved once, by App.ServedWindow in internal/app", rel)
		}
		return nil
	})
}

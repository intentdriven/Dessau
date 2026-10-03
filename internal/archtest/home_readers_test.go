package archtest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// homeReaders is every non-test file that reads the account's home directory
// itself, and why it may. Everything else resolves the account's Dessau
// directory through internal/config, whose one reader refuses inside a test
// binary while HOME is still the real one (iss-2609200823589977). A new reader
// of HOME added anywhere else is a way round that guard, and this is where it
// is noticed.
var homeReaders = map[string]string{
	"internal/config/config.go":       "userSupportDir: the guarded reader itself",
	"internal/lifecycle/install.go":   "liveInstallEnv, behind liveEnvGuard, which refuses inside a test",
	"internal/lifecycle/uninstall.go": "liveUninstallEnv, behind liveEnvGuard",
	"internal/lifecycle/update.go":    "liveUpdateEnv, behind liveEnvGuard",
	"internal/lifecycle/doctor.go":    "liveDoctorEnv: reads only, to say where things are",
	"internal/lifecycle/swap.go":      "redacting the home path out of a log line; reads nothing under it",
}

func TestOnlyTheKnownReadersResolveTheHomeDirectory(t *testing.T) {
	root := repoRootDir(t)
	found := map[string]bool{}
	walkRepoFiles(t, root, walkOptions{}, func(path string, d fs.DirEntry) error {
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(b)
		if strings.Contains(src, "os.UserHomeDir(") || strings.Contains(src, `Getenv("HOME")`) {
			rel, _ := filepath.Rel(root, path)
			found[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	var extra []string
	for rel := range found {
		if _, ok := homeReaders[rel]; !ok {
			extra = append(extra, rel)
		}
	}
	sort.Strings(extra)
	for _, rel := range extra {
		t.Errorf("%s reads the home directory itself. Resolve the account's Dessau directory through "+
			"internal/config, whose reader refuses the real home inside a test (iss-2609200823589977), "+
			"or add the file to homeReaders with the reason it may", rel)
	}
	for rel := range homeReaders {
		if !found[rel] {
			t.Errorf("homeReaders names %s, which no longer reads the home directory; drop it so the "+
				"list stays the whole of who does", rel)
		}
	}
}

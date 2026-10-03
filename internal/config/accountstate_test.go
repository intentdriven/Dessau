package config

import (
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"testing"
)

// An installation is one folder, and everything this account keeps is in it:
// config.json and registry.json sit in the root, EnsureDirs creates exactly
// the layout's directories, and the settings file is 0600.
func TestTheLayoutIsOneFolder(t *testing.T) {
	root := t.TempDir()
	p := NewPaths(root)
	if p.Config != filepath.Join(root, "config.json") {
		t.Errorf("Config = %q, want %q", p.Config, filepath.Join(root, "config.json"))
	}
	if p.State != filepath.Join(root, "registry.json") {
		t.Errorf("State = %q, want %q", p.State, filepath.Join(root, "registry.json"))
	}
	if err := p.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	if err := Save(p.Config, Default()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	want := []string{"bin", "config.json", "hf", "logs", "models", "python", "venv"}
	if len(names) != len(want) {
		t.Fatalf("the root holds %v, want exactly %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("the root holds %v, want exactly %v", names, want)
		}
	}
	fi, err := os.Stat(p.Config)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("config.json mode = %v, want 0600", fi.Mode().Perm())
	}
}

// privateToThisAccount is the rule for a secret file read back (the server's
// TLS private key): owned by this account, one link, mode 0600. Ownership
// alone is not enough — a hard link to a file this account owns passes a uid
// check while its content is whatever the linked file holds, and a file left
// group-writable is one another account could have written.
func TestPrivateToThisAccountRefusesALinkedOrOpenFile(t *testing.T) {
	stat := func(t *testing.T, path string) os.FileInfo {
		t.Helper()
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return fi
	}
	t.Run("hard link", func(t *testing.T) {
		dir := t.TempDir()
		original := filepath.Join(dir, "original.json")
		if err := Save(original, Default()); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(original, filepath.Join(dir, "linked.json")); err != nil {
			t.Skipf("this filesystem does not support hard links: %v", err)
		}
		if err := privateToThisAccount(stat(t, original)); err == nil {
			t.Error("a file with two links passed: a linked file is not this account's own")
		}
	})
	t.Run("group-writable", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "open.json")
		if err := Save(path, Default()); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o664); err != nil {
			t.Fatal(err)
		}
		if err := privateToThisAccount(stat(t, path)); err == nil {
			t.Error("a group-writable file passed: another account could have written it")
		}
	})
}

// A test running under one uid cannot create a file owned by another, so the
// uid half of the rule is checked against the one uid available; the two halves
// a test CAN reach — the link count and the mode — are exercised above.
func TestPrivateToThisAccountAcceptsThisAccountsOwnFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := Save(path, Default()); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("no stat information on this platform")
	}
	if int(st.Uid) != os.Getuid() {
		t.Fatalf("a file this account just wrote is owned by uid %d", st.Uid)
	}
	if err := privateToThisAccount(fi); err != nil {
		t.Errorf("a 0600 file this account just wrote was refused: %v", err)
	}
}

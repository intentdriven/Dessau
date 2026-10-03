package registry

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// writeModelConfig makes a model directory holding config.json with body.
func writeModelConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The pinned model server imports and runs the file a model's config.json
// names in model_file, on the condition `config.get("model_file") is not
// None`, read from the top level of the file and nowhere else
// (iss-2610030709283687). The check is that condition, exactly: any value
// that is not null refuses, an empty string included; null, an absent key, a
// key nested in another object and a key spelled in another case do not.
// Duplicate keys go to the last one, as Python's json module takes them.
func TestCheckModelCodeMirrorsTheModelServersOwnCondition(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		refuse bool
	}{
		{"a file name", `{"model_type":"llama","model_file":"model.py"}`, true},
		{"a path out of the directory", `{"model_type":"llama","model_file":"../../elsewhere.py"}`, true},
		{"an empty string", `{"model_type":"llama","model_file":""}`, true},
		{"a number", `{"model_type":"llama","model_file":0}`, true},
		{"false", `{"model_type":"llama","model_file":false}`, true},
		{"an object", `{"model_type":"llama","model_file":{}}`, true},
		{"the key spelled with an escape", `{"model_type":"llama","model_file":"model.py"}`, true},
		{"null", `{"model_type":"llama","model_file":null}`, false},
		{"absent", `{"model_type":"llama"}`, false},
		{"nested, not top-level", `{"model_type":"llava","text_config":{"model_file":"model.py"}}`, false},
		{"another case", `{"model_type":"llama","Model_File":"model.py"}`, false},
		{"duplicate keys, the last null", `{"model_type":"llama","model_file":"model.py","model_file":null}`, false},
		{"duplicate keys, the last a file", `{"model_type":"llama","model_file":null,"model_file":"model.py"}`, true},
		{"a JSON null document", `null`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckModelCode(writeModelConfig(t, c.body), os.Geteuid())
			if c.refuse && !errors.Is(err, ErrModelCode) {
				t.Errorf("CheckModelCode = %v, want ErrModelCode", err)
			}
			if !c.refuse && err != nil {
				t.Errorf("CheckModelCode = %v, want nil", err)
			}
		})
	}
}

// The refusal's own words are the reason the card and an entitled client
// are shown: plain, and free of anything path-shaped, which the registry's
// bound on a stored reason would otherwise blank.
func TestTheModelCodeRefusalIsPlainAndStorable(t *testing.T) {
	msg := ErrModelCode.Error()
	if !strings.Contains(msg, "ships its own code") || !strings.Contains(msg, "Dessau does not run") {
		t.Errorf("the refusal reads %q, want it to say the model ships its own code, which Dessau does not run", msg)
	}
	if !plausibleLoadFailure(&LoadFailure{Reason: msg}) {
		t.Errorf("the refusal %q would not survive the registry's bound on a recorded reason", msg)
	}
}

// What the check cannot read, it does not vouch for. A config.json that is
// missing is the model server's own failure to load and runs nothing, so it
// is reported as missing for the caller to treat as such; one that is there
// but is not something this reader accepts — a link, a FIFO, an oversized
// file, JSON the model server's parser takes and Go's does not — is an error
// that is not ErrModelCode and not ErrNotExist.
func TestCheckModelCodeReportsWhatItCannotRead(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		if err := CheckModelCode(t.TempDir(), os.Geteuid()); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("CheckModelCode on a directory without config.json = %v, want ErrNotExist", err)
		}
	})
	unreadable := map[string]func(t *testing.T) string{
		"a symlink": func(t *testing.T) string {
			target := writeModelConfig(t, `{"model_type":"llama","model_file":"model.py"}`)
			dir := t.TempDir()
			if err := os.Symlink(filepath.Join(target, "config.json"), filepath.Join(dir, "config.json")); err != nil {
				t.Fatal(err)
			}
			return dir
		},
		"a FIFO": func(t *testing.T) string {
			dir := t.TempDir()
			if err := syscall.Mkfifo(filepath.Join(dir, "config.json"), 0o644); err != nil {
				t.Fatal(err)
			}
			return dir
		},
		"oversized": func(t *testing.T) string {
			return writeModelConfig(t, `{"model_type":"llama","pad":"`+strings.Repeat("x", maxManifestJSON)+`"}`)
		},
		"NaN, which Python's parser accepts": func(t *testing.T) string {
			return writeModelConfig(t, `{"model_type":"llama","rope_theta":NaN,"model_file":"model.py"}`)
		},
		"a number out of float range": func(t *testing.T) string {
			return writeModelConfig(t, `{"model_type":"llama","rope_theta":1e400,"model_file":"model.py"}`)
		},
		"not an object": func(t *testing.T) string {
			return writeModelConfig(t, `["model_file"]`)
		},
	}
	for name, mk := range unreadable {
		t.Run(name, func(t *testing.T) {
			err := CheckModelCode(mk(t), os.Geteuid())
			if err == nil || errors.Is(err, ErrModelCode) || errors.Is(err, fs.ErrNotExist) {
				t.Errorf("CheckModelCode = %v, want an error that is neither ErrModelCode nor ErrNotExist", err)
			}
		})
	}
}

// In the shared cache a model's files belong to whichever account downloaded
// them, and the owner of a file — or of the directory it sits in — can
// replace it after the check has read it and before the model server does.
// So a model is loaded only from files the account running Dessau owns: the
// model's directory, and config.json as the handle the check read it through
// saw it. Anything else is refused with the plain reason, whatever the file
// says, since its content is not this account's to vouch for.
func TestCheckModelCodeRefusesFilesAnotherAccountOwns(t *testing.T) {
	other := os.Geteuid() + 1
	for name, body := range map[string]string{
		"a plain configuration":   `{"model_type":"llama"}`,
		"one naming a model_file": `{"model_type":"llama","model_file":"model.py"}`,
	} {
		t.Run("the directory, "+name, func(t *testing.T) {
			err := CheckModelCode(writeModelConfig(t, body), other)
			if !errors.Is(err, ErrOtherAccount) {
				t.Errorf("CheckModelCode for files another account owns = %v, want ErrOtherAccount", err)
			}
		})
		// The directory is this account's and config.json is not: the owner
		// is read off the handle the bytes came from, not off a second stat
		// of the path, which a swap between the two could answer.
		t.Run("config.json alone, "+name, func(t *testing.T) {
			dirsOnly := func(fi fs.FileInfo) bool { return fi.IsDir() }
			err := checkModelCode(writeModelConfig(t, body), dirsOnly)
			if !errors.Is(err, ErrOtherAccount) {
				t.Errorf("checkModelCode with config.json another account's = %v, want ErrOtherAccount", err)
			}
		})
	}
	t.Run("this account's own", func(t *testing.T) {
		if err := CheckModelCode(writeModelConfig(t, `{"model_type":"llama"}`), os.Geteuid()); err != nil {
			t.Errorf("CheckModelCode for this account's own files = %v, want nil", err)
		}
	})
	t.Run("a directory reached through a link", func(t *testing.T) {
		real := writeModelConfig(t, `{"model_type":"llama"}`)
		link := filepath.Join(t.TempDir(), "model")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		if err := CheckModelCode(link, os.Geteuid()); err != nil {
			t.Errorf("CheckModelCode through this account's own link = %v, want nil", err)
		}
		if err := CheckModelCode(link, other); !errors.Is(err, ErrOtherAccount) {
			t.Errorf("CheckModelCode through a link another account owns = %v, want ErrOtherAccount", err)
		}
	})
}

// The refusal's own words are the reason the card and an entitled client are
// shown, so they are plain and storable, like the model_file refusal's.
func TestTheOtherAccountRefusalIsPlainAndStorable(t *testing.T) {
	msg := ErrOtherAccount.Error()
	if !strings.Contains(msg, "belong to another account") || !strings.Contains(msg, "Dessau does not load") {
		t.Errorf("the refusal reads %q, want it to say the files belong to another account, which Dessau does not load", msg)
	}
	if !plausibleLoadFailure(&LoadFailure{Reason: msg}) {
		t.Errorf("the refusal %q would not survive the registry's bound on a recorded reason", msg)
	}
}

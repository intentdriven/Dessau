package lifecycle

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// uninstallFixture lays down an installation in a temporary directory: a
// bundle, the private runtime, the settings, the registry, the logs, the
// statistics, the per-user link, and a model that must survive all of it.
func uninstallFixture(t *testing.T) (Env, UninstallEnv, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	env, out, errOut := testEnv()

	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	root := filepath.Join(home, "Library", "Application Support", "Dessau")
	paths := config.NewPaths(root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{paths.Config, paths.State, paths.UV(), paths.VenvPython(),
		filepath.Join(paths.Logs, "dessau.log"), filepath.Join(paths.Stats, "2026-09.json"),
		filepath.Join(paths.Python, "cpython-3.12", "bin", "python3"),
		filepath.Join(paths.Models, "mlx-community", "a-model", "weights.safetensors"),
		filepath.Join(paths.HFCache, "blob"),
	} {
		writeFileAt(t, f, "x")
	}

	bundle := bundleAt(t, filepath.Join(home, "Applications", "DessauServer.app"), "installed")
	binary := filepath.Join(bundle, binaryInBundle)
	link, err := linkCommand(home, binary)
	if err != nil {
		t.Fatal(err)
	}

	return env, UninstallEnv{
		Paths:    paths,
		Home:     home,
		Bundles:  []string{bundle},
		Link:     link,
		Binary:   binary,
		Terminal: true,
		Firewall: func(string) error { return nil },
	}, out, errOut
}

func writeFileAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// Uninstall removes the application, the runtime, the settings, the registry,
// the logs and the link — and leaves the models, which are the expensive
// thing, saying how much space they take and what removes them.
func TestUninstallRemovesTheApplicationAndLeavesTheModels(t *testing.T) {
	env, ue, out, _ := uninstallFixture(t)

	if code := runUninstall(env, nil, ue); code != ExitOK {
		t.Fatalf("exit = %d, want %d\n%s", code, ExitOK, out)
	}

	for _, gone := range []string{ue.Bundles[0], ue.Paths.Venv, ue.Paths.Python, ue.Paths.Bin,
		ue.Paths.Config, ue.Paths.State, ue.Paths.Logs, ue.Paths.Stats, ue.Link} {
		if exists(gone) {
			t.Errorf("%s is still there", redact(gone, ue.Home))
		}
	}
	for _, kept := range []string{ue.Paths.Models, ue.Paths.HFCache} {
		if !exists(kept) {
			t.Errorf("%s was removed; the downloaded models are what uninstall leaves", redact(kept, ue.Home))
		}
	}
	got := out.String()
	if !strings.Contains(got, "--purge") {
		t.Errorf("the output does not name the flag that removes the models:\n%s", got)
	}
	if !strings.Contains(got, "B") {
		t.Errorf("the output does not state the size of what it left:\n%s", got)
	}
}

// --purge without a terminal and without --yes deletes NOTHING and names the
// flag. Under a piped bootstrap standard input is the rest of the installer, so
// there is nobody to ask and nothing to read.
func TestPurgeRefusesWithoutATerminalUnlessToldYes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		terminal bool
		args     []string
		deletes  bool
	}{
		{"no terminal, no --yes", false, []string{"--purge"}, false},
		{"no terminal, --yes", false, []string{"--purge", "--yes"}, true},
		{"a terminal", true, []string{"--purge"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, ue, out, errOut := uninstallFixture(t)
			ue.Terminal = tc.terminal

			code := runUninstall(env, tc.args, ue)
			modelsGone := !exists(ue.Paths.Models)
			if modelsGone != tc.deletes {
				t.Errorf("models removed = %v, want %v", modelsGone, tc.deletes)
			}
			if tc.deletes {
				return
			}
			if code == ExitOK {
				t.Errorf("a refused purge exited %d, which reads as having done the work", code)
			}
			if exists(ue.Bundles[0]) == false {
				t.Error("a refused purge removed the application; it must delete nothing at all")
			}
			if !strings.Contains(errOut.String(), "--yes") {
				t.Errorf("the refusal does not name the flag that would answer it:\n%s%s", out, errOut)
			}
		})
	}
}

// The firewall entry is machine-wide state with no per-account route, so its
// removal is the one authorisation panel. A refusal leaves everything else
// removed and reports the entry as the one thing remaining, with the command.
func TestUninstallSurvivesARefusedAuthorisation(t *testing.T) {
	env, ue, out, _ := uninstallFixture(t)
	asked := 0
	ue.Firewall = func(string) error {
		asked++
		return errors.New("the authorisation was declined")
	}

	if code := runUninstall(env, nil, ue); code != ExitOK {
		t.Fatalf("exit = %d, want %d: a declined panel is not a failed uninstall", code, ExitOK)
	}
	if asked != 1 {
		t.Errorf("the panel was raised %d times, want exactly 1", asked)
	}
	if exists(ue.Bundles[0]) {
		t.Error("a refused authorisation stopped the rest of the removal")
	}
	got := out.String()
	if !strings.Contains(got, "--remove") {
		t.Errorf("the output does not name the command that removes the firewall entry:\n%s", got)
	}
}

// A bundle in the machine-wide applications directory is the copy EVERY account
// on this Mac launches, and removing it removes it for all of them. The
// asymmetry is named in the code; the output has to name it too, because the
// person running uninstall is the only one who will find out otherwise.
func TestUninstallSaysWhenItRemovedTheCopyEveryAccountLaunches(t *testing.T) {
	env, ue, out, _ := uninstallFixture(t)

	// A stand-in for the machine-wide directory, so this test never touches the
	// real one.
	apps := filepath.Join(t.TempDir(), "Applications")
	machineWide := bundleAt(t, filepath.Join(apps, "DessauServer.app"), "everybody's")
	ue.SystemApplications = apps
	ue.Bundles = []string{machineWide}

	if code := runUninstall(env, nil, ue); code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if exists(machineWide) {
		t.Fatal("the bundle was not removed")
	}
	if !strings.Contains(out.String(), "every account") {
		t.Errorf("removing the copy every account on this Mac launches was reported as an ordinary removal:\n%s", out)
	}

	// And a per-account bundle is not described that way.
	env, ue, out, _ = uninstallFixture(t)
	if code := runUninstall(env, nil, ue); code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if strings.Contains(out.String(), "every account") {
		t.Errorf("this account's own bundle was reported as everybody's:\n%s", out)
	}
}

// DESSAU_ROOT is never a deletion path. A directory any local account can
// pre-create as a symlink would otherwise choose what is deleted, so the
// removal acts on the fixed locations this account's install uses — and says so
// rather than leaving a reader to expect the variable to be honoured.
func TestDessauRootIsNeverADeletionPath(t *testing.T) {
	decoy := t.TempDir()
	witness := filepath.Join(decoy, "models", "someone-elses-data")
	writeFileAt(t, witness, "not ours to delete")

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DESSAU_ROOT", decoy)

	// The live resolution is what this case is about — which root uninstall
	// picks when the environment names another one — so the guard that keeps
	// every other test off this path is opened deliberately, for this test
	// only.
	allowLiveEnvInTest(t)

	env, _, _ := testEnv()
	ue, err := liveUninstallEnv(env)
	if err != nil {
		t.Fatal(err)
	}
	// The bundle locations and the panel are the two things a test may not
	// exercise for real; everything the case is about — which root is resolved,
	// and what is said about the one that was not — is left live.
	ue.Bundles = nil
	ue.Firewall = func(string) error { return nil }
	ue.Terminal = false

	out := &bytes.Buffer{}
	env.Out = out
	if code := runUninstall(env, nil, ue); code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !exists(witness) {
		t.Fatal("uninstall deleted what DESSAU_ROOT named")
	}
	if !strings.Contains(out.String(), "DESSAU_ROOT") {
		t.Errorf("the output does not name the root it did not remove:\n%s", out)
	}
}

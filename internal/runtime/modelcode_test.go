package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/registry"
)

// markerLauncher is an ExecLauncher whose interpreter leaves a file behind
// when it runs, so a test can tell whether any process was started at all.
func markerLauncher(t *testing.T) (*ExecLauncher, string) {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "ran")
	return stubbedLauncher(t, "touch '"+marker+"'"), marker
}

// modelDirWithConfig makes a model directory whose config.json is body.
func modelDirWithConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func ran(marker string) bool {
	_, err := os.Stat(marker)
	return err == nil
}

// A model whose config.json names a model_file is refused before any process
// starts: the pinned model server would import and run that file under this
// account (iss-2610030709283687). The refusal is the model's own load
// failure, not a broken installation — a NotReadyError carrying the plain
// reason, which stands until a person retries — so it goes the way every
// other load failure goes: onto the model's card and to an entitled client.
func TestTheLauncherRefusesAModelThatShipsItsOwnCode(t *testing.T) {
	l, marker := markerLauncher(t)
	dir := modelDirWithConfig(t, `{"model_type":"llama","model_file":"model.py"}`)
	spec := Spec{RepoID: "org/code", ModelPath: dir, Port: 1}

	for name, call := range map[string]func() error{
		"Precheck": func() error { return l.Precheck(spec) },
		"Launch": func() error {
			p, err := l.Launch(context.Background(), spec)
			if p != nil {
				<-p.Done()
				t.Error("Launch returned a process for a model that ships its own code")
			}
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			var notReady *NotReadyError
			if !errors.As(err, &notReady) {
				t.Fatalf("%s = %v, want a NotReadyError", name, err)
			}
			if !errors.Is(err, registry.ErrModelCode) {
				t.Errorf("%s = %v, want it to wrap registry.ErrModelCode", name, err)
			}
			if notReady.Reason != registry.ErrModelCode.Error() || notReady.Transient {
				t.Errorf("%s reason = %q (transient %v), want %q standing until a person retries",
					name, notReady.Reason, notReady.Transient, registry.ErrModelCode.Error())
			}
			if strings.Contains(err.Error(), dir) {
				t.Errorf("%s = %q, which carries the model directory's path", name, err)
			}
		})
	}
	if ran(marker) {
		t.Fatal("a process was started for a model that ships its own code")
	}
}

// A config.json that is there but that Go's reader does not accept is
// refused too: the model server's parser takes some of what Go's refuses
// (NaN, a number past float range), so a check that waved those through
// would be one the file could step round. The refusal is transient — it
// says nothing about the next start — and names no path.
func TestTheLauncherRefusesAConfigItCannotRead(t *testing.T) {
	l, marker := markerLauncher(t)
	dir := modelDirWithConfig(t, `{"model_type":"llama","rope_theta":NaN,"model_file":"model.py"}`)
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/nan", ModelPath: dir, Port: 1})
	if p != nil {
		<-p.Done()
	}
	var notReady *NotReadyError
	if !errors.As(err, &notReady) || !notReady.Transient || notReady.Reason == "" || strings.Contains(notReady.Reason, "/") {
		t.Fatalf("Launch = %v (%+v), want a transient NotReadyError with a path-free reason", err, notReady)
	}
	if ran(marker) {
		t.Fatal("a process was started for a model whose config.json could not be read")
	}
}

// A model_file that is null, or absent, launches exactly as before: the
// model server runs nothing from the directory in either case.
func TestTheLauncherStartsAModelThatShipsNoCode(t *testing.T) {
	for name, body := range map[string]string{
		"model_file null":   `{"model_type":"llama","model_file":null}`,
		"model_file absent": `{"model_type":"llama"}`,
	} {
		t.Run(name, func(t *testing.T) {
			l, marker := markerLauncher(t)
			dir := modelDirWithConfig(t, body)
			launchAndWait(t, l, Spec{RepoID: "org/plain", ModelPath: dir, Port: 1})
			if !ran(marker) {
				t.Error("the model server was not started")
			}
		})
	}
}

// A model whose files belong to another account is refused before any
// process starts. In the shared cache the owner of a model's directory or of
// its config.json can replace that file after the check has read it and
// before the model server does, so only this account's own files are loaded:
// Dessau serves from one account, and other accounts reach it over the
// network. The refusal is the model's own load failure, with the plain reason,
// standing until a person retries, and it names no path.
func TestTheLauncherRefusesAModelAnotherAccountOwns(t *testing.T) {
	l, marker := markerLauncher(t)
	l.modelOwner = os.Geteuid() + 1
	dir := modelDirWithConfig(t, `{"model_type":"llama"}`)
	spec := Spec{RepoID: "org/theirs", ModelPath: dir, Port: 1}

	for name, call := range map[string]func() error{
		"Precheck": func() error { return l.Precheck(spec) },
		"Launch": func() error {
			p, err := l.Launch(context.Background(), spec)
			if p != nil {
				<-p.Done()
				t.Error("Launch returned a process for a model another account owns")
			}
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			var notReady *NotReadyError
			if !errors.As(err, &notReady) || !errors.Is(err, registry.ErrOtherAccount) {
				t.Fatalf("%s = %v, want a NotReadyError wrapping registry.ErrOtherAccount", name, err)
			}
			if notReady.Reason != registry.ErrOtherAccount.Error() || notReady.Transient {
				t.Errorf("%s reason = %q (transient %v), want %q standing until a person retries",
					name, notReady.Reason, notReady.Transient, registry.ErrOtherAccount.Error())
			}
			if strings.Contains(err.Error(), dir) {
				t.Errorf("%s = %q, which carries the model directory's path", name, err)
			}
		})
	}
	if ran(marker) {
		t.Fatal("a process was started for a model another account owns")
	}
}

// A model directory with no config.json is refused, not launched. The model
// server cannot load a model without one, so nothing is lost; and in the
// shared cache a model directory is writable by every account in the group,
// so a config.json that is missing when the check looks is one another
// account can create before the model server looks — with a model_file in it.
func TestTheLauncherRefusesAModelWithNoConfig(t *testing.T) {
	l, marker := markerLauncher(t)
	dir := t.TempDir()
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/bare", ModelPath: dir, Port: 1})
	if p != nil {
		<-p.Done()
	}
	var notReady *NotReadyError
	if !errors.As(err, &notReady) || notReady.Reason == "" || strings.Contains(notReady.Reason, "/") {
		t.Fatalf("Launch = %v (%+v), want a NotReadyError with a path-free reason", err, notReady)
	}
	if ran(marker) {
		t.Fatal("a process was started for a model with no config.json")
	}
}

// The pool takes the launcher's refusal as the model's load failure: the
// caller gets the NotReadyError itself, not a LaunchError the gateway would
// turn into "could not be started"; the observer is told the load failed with
// the reason, which is what records it on the model; nothing is launched;
// and the model in memory is not evicted for a load that was never going to
// happen.
func TestThePoolTakesAModelRefusalAsTheModelsLoadFailure(t *testing.T) {
	obs := &recordingObserver{}
	l := newFakeLauncher()
	l.refuseFor = "org/code"
	src := &fakeSource{models: map[string]int64{"org/a": 100, "org/code": 100}}
	// loadCost is 1.2x, so a 200-byte budget fits exactly one model.
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 200, Observer: obs})

	_, release, err := p.Acquire(context.Background(), "org/a")
	if err != nil {
		t.Fatal(err)
	}
	release()

	_, _, err = p.Acquire(context.Background(), "org/code")
	var notReady *NotReadyError
	var launchErr *LaunchError
	if !errors.As(err, &notReady) || errors.As(err, &launchErr) {
		t.Fatalf("Acquire = %v, want the launcher's NotReadyError and no LaunchError", err)
	}
	if !strings.Contains(err.Error(), "ships its own code") {
		t.Errorf("Acquire = %q, want the plain reason", err)
	}
	if got := awaitLoad(t, obs, "org/code"); !got.failed || got.interrupted || got.reason != registry.ErrModelCode.Error() {
		t.Errorf("the observer was told %+v, want the model's own failure with the plain reason", got)
	}
	l.mu.Lock()
	launched := append([]string(nil), l.launched...)
	l.mu.Unlock()
	for _, id := range launched {
		if id == "org/code" {
			t.Error("the pool launched a model the launcher refused")
		}
	}
	resident := false
	for _, r := range p.Resident() {
		if r.RepoID == "org/a" {
			resident = true
		}
	}
	if !resident {
		t.Error("a resident model was evicted to make room for a load the launcher refused")
	}
}

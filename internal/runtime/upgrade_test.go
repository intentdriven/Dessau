package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// A runtime installed from another lock is not this build's: the marker names
// the lock's own hash, so the old pin's install — and the same pin installed
// with another dependency set — is reprovisioned rather than trusted
// (spc-2610030846273729 step 1).
func TestARuntimeFromAnotherLockIsReprovisioned(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	writeInterpreter(t, paths, 0o755)
	p := &Provisioner{Paths: paths}
	if !p.Installed() {
		t.Fatal("a runtime installed from this lock reads as not installed")
	}
	if err := os.Remove(filepath.Join(paths.Venv, mlxMarkerName())); err != nil {
		t.Fatal(err)
	}
	for _, old := range []string{".dessau-mlx-0.31.3", ".dessau-mlx-" + mlxLMVersion, ".dessau-mlx-" + mlxLMVersion + "-0123456789abcdef"} {
		if err := os.WriteFile(filepath.Join(paths.Venv, old), []byte("ok"), 0o644); err != nil {
			t.Fatal(err)
		}
		if p.Installed() {
			t.Errorf("a runtime marked %s reads as this lock's", old)
		}
	}
}

// The lock carries the signed-off set and nothing else: the three pins, and
// every line hash-locked (DECISIONS 2026-10-03, dependency sign-off).
func TestTheLockIsTheSignedOffSet(t *testing.T) {
	lock := string(mlxRequirements)
	for _, pin := range []string{"mlx-lm==0.32.0", "mlx==0.32.3", "mlx-metal==0.32.3", "mlx-vlm==0.7.4"} {
		if !strings.Contains(lock, "\n"+pin+" \\\n") {
			t.Errorf("the lock does not pin %s", pin)
		}
	}
	if mlxLMVersion != "0.32.0" {
		t.Errorf("mlxLMVersion = %s, but the lock pins mlx-lm 0.32.0", mlxLMVersion)
	}
	n := 0
	for _, line := range strings.Split(lock, "\n") {
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, " ") {
			continue
		}
		n++
		if !strings.Contains(line, "==") || !strings.HasSuffix(line, " \\") {
			t.Errorf("a requirement that is not an exact, hashed pin: %q", line)
		}
	}
	if n != 57 {
		t.Errorf("the lock holds %d packages; the signed-off set is 57", n)
	}
}

// No OpenTelemetry configuration reaches a child: the runtime carries
// opentelemetry-api, and an exporter or endpoint set where Dessau was started
// is the one way it could send anything (adr-2609201008476813).
func TestNoOpenTelemetryConfigurationReachesAChild(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector.example")
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("otel_metrics_exporter", "otlp")
	t.Setenv("DESSAU_TEST_KEEPS", "1")
	paths := config.NewPaths(t.TempDir())
	l := &ExecLauncher{Paths: paths, LogDir: paths.Logs}
	p := &Provisioner{Paths: paths}
	for name, env := range map[string][]string{
		"model server": l.childEnv(), "install check": childEnviron(), "uv": p.uvEnv(),
	} {
		kept := false
		for _, kv := range env {
			if strings.HasPrefix(strings.ToUpper(kv), "OTEL_") {
				t.Errorf("%s: %s reaches the child", name, kv)
			}
			kept = kept || kv == "DESSAU_TEST_KEEPS=1"
		}
		if !kept {
			t.Errorf("%s: the rest of the environment was dropped too", name)
		}
	}
}

// An install clears every marker before it touches the venv, so a venv in the
// middle of an install, or one a failed install left, is not one an older
// lock's marker still vouches for (adversarial review of step 1).
func TestAnInstallClearsEveryMarkerFirst(t *testing.T) {
	venv := t.TempDir()
	for _, m := range []string{".dessau-mlx-0.31.3", ".dessau-mlx-0.32.0-0123456789abcdef", mlxMarkerName()} {
		if err := os.WriteFile(filepath.Join(venv, m), []byte("ok"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(venv, "pyvenv.cfg")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := removeMLXMarkers(venv); err != nil {
		t.Fatal(err)
	}
	left, _ := filepath.Glob(filepath.Join(venv, ".dessau-mlx-*"))
	if len(left) != 0 {
		t.Errorf("markers left: %v", left)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("the clear removed more than markers: %v", err)
	}
}

// Precheck tells a runtime that is not there yet from one that is there and
// not trusted: only the first is ErrRuntimeNotInstalled, so only the first
// lets a staged version fall back to the model check (iss-2610031317475284).
func TestPrecheckTellsAMissingRuntimeFromAnUntrustedOne(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	l := &ExecLauncher{Paths: paths, LogDir: paths.Logs}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"model_type":"qwen3"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := Spec{RepoID: "org/m", ModelPath: dir}
	if err := l.Precheck(spec); !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Errorf("no interpreter: err = %v, want ErrRuntimeNotInstalled", err)
	}
	if err := l.PrecheckModel(spec); err != nil {
		t.Errorf("the model half on a clean model: %v", err)
	}
	writeInterpreter(t, paths, 0o775)
	if err := l.Precheck(spec); err == nil || errors.Is(err, ErrRuntimeNotInstalled) {
		t.Errorf("a group-writable interpreter: err = %v, want a refusal that is not 'not installed'", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"model_type":"qwen3","model_file":"x.py"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := l.PrecheckModel(spec); err == nil {
		t.Error("the model half accepted a model that ships its own code")
	}
}

package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// What the live server's child wrote on 2026-09-21 while its httpd went on
// answering /health: the generate thread died on the first request, so no
// completion ever came back and the pool waited its whole readiness timeout.
// The frames carry the venv's absolute paths, as a real traceback does.
const glmOCRTraceback = `Exception in thread Thread-1 (_generate):
Traceback (most recent call last):
  File "/Users/alice/Library/Application Support/Dessau/venv/lib/python3.12/site-packages/mlx_lm/utils.py", line 188, in _get_classes
    arch = importlib.import_module(f"mlx_lm.models.{model_type}")
ModuleNotFoundError: No module named 'mlx_lm.models.glm_ocr'

During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File "/Users/alice/Library/Application Support/Dessau/venv/lib/python3.12/site-packages/mlx_lm/server.py", line 695, in _generate
    self.model_provider.load_default()
 2026-09-21 15:51:35,267 - INFO - Starting httpd at 127.0.0.1 on port 59269...
  File "/Users/alice/Library/Application Support/Dessau/venv/lib/python3.12/site-packages/mlx_lm/utils.py", line 191, in _get_classes
    raise ValueError(msg)
ValueError: Model type glm_ocr not supported.
`

// A load whose child has already said it cannot load the model ends at
// once, with the child's own reason, rather than at the readiness timeout:
// the pool reads the log it captures per model while it waits
// (iss-2609211334570516).
func TestAFatalLineInTheChildLogEndsTheLoadWaitAtOnce(t *testing.T) {
	l := newFakeLauncher()
	l.loadDelayFor["org/ocr"] = time.Hour // the completion never comes back
	logPath := filepath.Join(t.TempDir(), "org@ocr.log")
	l.logPathFor["org/ocr"] = logPath
	src := &fakeSource{models: map[string]int64{"org/ocr": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30, ReadyTimeout: 30 * time.Second})

	// Written a moment after the launch, as the real child writes it on its
	// first request: the wait is watching the file, not reading it once.
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = os.WriteFile(logPath, []byte(glmOCRTraceback), 0o600)
	}()
	started := time.Now()
	_, _, err := p.Acquire(context.Background(), "org/ocr")
	took := time.Since(started)
	if err == nil {
		t.Fatal("a model whose child raised on load was handed back as ready")
	}
	var notReady *NotReadyError
	if !errors.As(err, &notReady) {
		t.Fatalf("Acquire error = %T %v, want a NotReadyError", err, err)
	}
	if took > 10*time.Second {
		t.Errorf("the load wait took %s, want it ended by the log line well inside the 30s readiness timeout", took)
	}
	if !strings.Contains(err.Error(), "ValueError: Model type glm_ocr not supported.") {
		t.Errorf("the reason does not carry the child's own line: %q", err)
	}
	if strings.Contains(err.Error(), "within") {
		t.Errorf("the reason is the readiness timeout's, not the child's: %q", err)
	}
	if strings.Contains(err.Error(), "/Users/") {
		t.Errorf("the reason carries a local path: %q", err)
	}
	// And the failed server is out of the pool, its memory on its way back.
	if res := p.Resident(); len(res) != 0 {
		t.Errorf("the failed model is still resident: %+v", res)
	}
}

// Only a traceback's terminal line of the kinds that mean the model cannot
// load ends the wait; the child's ordinary chatter, a bare warning, or an
// exception the child recovers from, does not. Whatever line is taken is
// bounded and stripped of anything path-shaped before it goes anywhere.
func TestOnlyAFatalTracebackLineIsReadAsALoadFailure(t *testing.T) {
	cases := []struct {
		name, log, want string
	}{
		{"the live traceback", glmOCRTraceback, "ValueError: Model type glm_ocr not supported."},
		{"a missing module", "Traceback (most recent call last):\n  File \"x.py\", line 1\nModuleNotFoundError: No module named 'mlx_vlm'\n",
			"ModuleNotFoundError: No module named 'mlx_vlm'"},
		{"an import error", "Traceback (most recent call last):\nImportError: cannot import name 'foo'\n", "ImportError: cannot import name 'foo'"},
		{"chatter alone", "UserWarning: mlx_lm.server is not recommended for production\n2026-09-21 - INFO - Starting httpd at 127.0.0.1 on port 1\n", ""},
		{"a ValueError with no traceback", "ValueError: stray\n", ""},
		{"a broken pipe the child survives", "Traceback (most recent call last):\n  File \"x.py\"\nBrokenPipeError: [Errno 32] Broken pipe\n", ""},
		{"a path in the message", "Traceback (most recent call last):\nValueError: bad weights at /Users/alice/models/x/model.safetensors here\n",
			"ValueError: bad weights at <path> here"},
		{"an overlong line", "Traceback (most recent call last):\nValueError: " + strings.Repeat("x", 1000) + "\n",
			"ValueError: " + strings.Repeat("x", maxFatalLineBytes-len("ValueError: "))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "child.log")
			if err := os.WriteFile(path, []byte(c.log), 0o600); err != nil {
				t.Fatal(err)
			}
			got, ok := fatalLoadLine(path)
			if got != c.want || ok != (c.want != "") {
				t.Errorf("fatalLoadLine = %q, %v; want %q", got, ok, c.want)
			}
		})
	}
	if got, ok := fatalLoadLine(filepath.Join(t.TempDir(), "absent.log")); ok || got != "" {
		t.Errorf("a log that does not exist yet read as %q, %v", got, ok)
	}
}

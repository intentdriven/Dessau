package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// A start keeps an aside copy whenever the model's folder is one the rescan
// would refuse — here, a version whose download left a part file — even if
// the folder looks like a model at a glance: the aside may be the only copy
// the rescan would serve (iss-2610031807030740).
func TestAnAsideCopyIsKeptBesideAFolderTheRescanWouldRefuse(t *testing.T) {
	models := t.TempDir()
	aside := filepath.Join(models, stagingDirName, "org", "repo"+asideSuffix+"abc")
	os.MkdirAll(aside, 0o755)
	os.WriteFile(filepath.Join(aside, "config.json"), []byte(`{"model_type":"qwen3"}`), 0o644)
	os.WriteFile(filepath.Join(aside, "model.safetensors"), []byte("old"), 0o644)
	served := filepath.Join(models, "org", "repo")
	os.MkdirAll(served, 0o755)
	os.WriteFile(filepath.Join(served, "config.json"), []byte(`{"model_type":"qwen3"}`), 0o644)
	os.WriteFile(filepath.Join(served, "model.safetensors"), []byte("new"), 0o644)
	os.WriteFile(filepath.Join(served, "model-2.safetensors.dessau-part"), []byte("half"), 0o644)

	recoverStaging(models, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := os.Stat(filepath.Join(aside, "model.safetensors")); err != nil {
		t.Errorf("the aside copy was removed beside a folder the rescan refuses: %v", err)
	}
}

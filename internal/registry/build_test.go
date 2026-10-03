package registry

import (
	"os"
	"path/filepath"
	"testing"
)

// A build belongs to the model HuggingFace names as its origin when its own
// tags say, once, that it is quantised from it; the origin is folded by the
// one rule for model ids. No such tag, two of them, or one naming no valid
// model id, and the build stands on its own (itd-2610030932551549).
func TestABuildOfIsTheOneOriginItsTagsName(t *testing.T) {
	cases := []struct {
		tags []string
		want string
	}{
		{[]string{"mlx", "base_model:quantized:Qwen/Qwen3-8B", "base_model:Qwen/Qwen3-8B"}, "qwen/qwen3-8b"},
		{[]string{"mlx", "base_model:Qwen/Qwen3-8B"}, ""},
		{[]string{"base_model:quantized:Qwen/Qwen3-8B", "base_model:quantized:Other/Model"}, ""},
		{[]string{"base_model:quantized:Qwen/Qwen3-8B", "base_model:quantized:qwen/QWEN3-8B"}, "qwen/qwen3-8b"},
		{[]string{"base_model:quantized:not-a-repo-id"}, ""},
		{[]string{"base_model:quantized:../x"}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := (Model{Tags: c.tags}).BuildOf(); got != c.want {
			t.Errorf("BuildOf(%v) = %q, want %q", c.tags, got, c.want)
		}
	}
}

// The precision of a build is read from its own config.json — the
// quantization block mlx-lm writes — never from its name, and re-derived at
// every rescan like the other facts of the directory.
func TestQuantizationBitsComeFromTheConfig(t *testing.T) {
	for body, want := range map[string]int{
		`{"model_type":"qwen3","quantization":{"group_size":64,"bits":4}}`:        4,
		`{"model_type":"qwen3","quantization_config":{"group_size":64,"bits":8}}`: 8,
		`{"model_type":"qwen3"}`:                             0,
		`{"model_type":"qwen3","quantization":{"bits":"4"}}`: 0,
		`{"model_type":"qwen3","quantization":{"bits":4.5}}`: 0,
		`{"model_type":"qwen3","quantization":{"bits":99}}`:  0,
		`{"model_type":"qwen3","quantization":{"bits":0}}`:   0,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := ReadModelFacts(dir).QuantizationBits; got != want {
			t.Errorf("%s: bits = %d, want %d", body, got, want)
		}
	}

	models := t.TempDir()
	dir := writeModelDirWithConfig(t, models, "org", "m", `{"model_type":"qwen3","quantization":{"bits":4}}`, 64)
	_ = dir
	r, err := Open(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Rescan(models); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/m"); m.QuantizationBits != 4 {
		t.Errorf("the rescan read %d bits", m.QuantizationBits)
	}
}

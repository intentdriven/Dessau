package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
)

// buildsGateway serves a 4-bit and an 8-bit build HuggingFace labels as
// quantised from one origin, and a model with no such label.
func buildsGateway(t *testing.T) (*httptest.Server, *mlxtest.Server) {
	t.Helper()
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m", Reply: "OK"})
	t.Cleanup(fake.Close)
	models := &stubModels{models: []registry.Model{
		{RepoID: "mlx-community/Qwen3-8B-4bit", State: registry.StateReady, Bytes: 4 << 30, QuantizationBits: 4,
			Tags: []string{"mlx", "base_model:quantized:Qwen/Qwen3-8B"}},
		{RepoID: "mlx-community/Qwen3-8B-8bit", State: registry.StateReady, Bytes: 8 << 30, QuantizationBits: 8,
			Tags: []string{"mlx", "base_model:quantized:qwen/qwen3-8b"}},
		{RepoID: "org/alone", State: registry.StateReady, Bytes: 1 << 30, Tags: []string{"mlx"}},
	}}
	g := New(Options{Config: config.Default(), Pool: &stubPool{srv: fake}, Models: models})
	srv := httptest.NewServer(g.Handler())
	t.Cleanup(srv.Close)
	return srv, fake
}

// Two builds labelled as quantised from one model each keep their own name
// and carry that origin as build_of, with their precision and size; a model
// with no such label carries no build_of and still its own facts
// (itd-2610030932551549 criteria 1 and 2).
func TestBuildsOfOneModelShareTheirOriginAndKeepTheirNames(t *testing.T) {
	srv, _ := buildsGateway(t)
	byID := map[string]map[string]any{}
	for _, e := range allModelEntries(t, srv) {
		byID[e["id"].(string)] = e
	}
	four, eight, alone := byID["mlx-community/Qwen3-8B-4bit"], byID["mlx-community/Qwen3-8B-8bit"], byID["org/alone"]
	if four == nil || eight == nil || alone == nil {
		t.Fatalf("an entry lost its own name: %v", byID)
	}
	if four["build_of"] != "qwen/qwen3-8b" || eight["build_of"] != "qwen/qwen3-8b" {
		t.Errorf("build_of = %v and %v, want one origin", four["build_of"], eight["build_of"])
	}
	if four["quantization_bits"] != float64(4) || eight["quantization_bits"] != float64(8) {
		t.Errorf("quantization_bits = %v and %v", four["quantization_bits"], eight["quantization_bits"])
	}
	if four["size_bytes"] != float64(4<<30) || alone["size_bytes"] != float64(1<<30) {
		t.Errorf("size_bytes = %v and %v", four["size_bytes"], alone["size_bytes"])
	}
	if _, ok := alone["build_of"]; ok {
		t.Errorf("a model with no label carries build_of %v", alone["build_of"])
	}
	if _, ok := alone["quantization_bits"]; ok {
		t.Errorf("a model that declares no precision carries one: %v", alone["quantization_bits"])
	}
}

// A group's name is not a model name: a request naming the origin, in full
// or by its short name, is refused as not served, and every build stays
// callable by its own name (criterion 5).
func TestAGroupNameIsNotAModelName(t *testing.T) {
	srv, _ := buildsGateway(t)
	for _, name := range []string{"qwen/qwen3-8b", "Qwen/Qwen3-8B", "Qwen3-8B"} {
		resp := post(t, srv, "/v1/chat/completions", map[string]any{
			"model": name, "messages": []map[string]string{{"role": "user", "content": "hi"}},
		}, nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("a request naming the group %q: status %d, want 404", name, resp.StatusCode)
		}
	}
	resp := post(t, srv, "/v1/chat/completions", map[string]any{
		"model": "mlx-community/Qwen3-8B-4bit", "messages": []map[string]string{{"role": "user", "content": "hi"}},
	}, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("a build called by its own name: status %d", resp.StatusCode)
	}
}

// The size the list publishes is bounded where it leaves the machine, as the
// context length is: a figure past any real model's size is not published.
func TestAnAbsurdSizeIsNotPublished(t *testing.T) {
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m", Reply: "OK"})
	defer fake.Close()
	models := &stubModels{models: []registry.Model{
		{RepoID: "org/huge", State: registry.StateReady, Bytes: 9007199254740993},
	}}
	g := New(Options{Config: config.Default(), Pool: &stubPool{srv: fake}, Models: models})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()
	if e := firstModelEntry(t, srv); e["size_bytes"] != nil {
		t.Errorf("size_bytes = %v, want it withheld", e["size_bytes"])
	}
}

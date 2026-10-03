package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/app"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
)

// A chat request for a model that ships its own code is refused before any
// process starts, and the operator is told why in plain words on both of
// the surfaces a load failure reaches: the error a client on this Mac
// receives, and the model's card in the control panel's state
// (iss-2610030709283687). The composition is the real one — app, pool, the
// real launcher over a stand-in interpreter that leaves a marker if it is
// ever run.
func TestAModelThatShipsItsOwnCodeIsRefusedWithThePlainReason(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	marker := filepath.Join(t.TempDir(), "ran")
	if err := os.MkdirAll(filepath.Dir(paths.VenvPython()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.VenvPython(), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	a, err := app.New(app.Options{Paths: paths, Config: config.Default()})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	const id = "org/code"
	dir := paths.ModelDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"model_type":"llama","model_file":"model.py"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.Registry.Put(registry.Model{
		RepoID: id, Path: dir, State: registry.StateReady, Bytes: 1 << 20, Progress: 100,
		ContextLength: 131072, KVChargePerToken: 64, ChatTemplate: true,
	}); err != nil {
		t.Fatal(err)
	}

	g := New(Options{ConfigFunc: a.Config, Pool: a.Pool, Models: a.Registry, ServedWindow: a.ServedWindow})
	api := httptest.NewServer(g.Handler())
	t.Cleanup(api.Close)
	ctrl := &Control{App: a}
	mux := http.NewServeMux()
	ctrl.Routes(mux)
	panel := httptest.NewServer(mux)
	t.Cleanup(panel.Close)

	resp, err := api.Client().Post(api.URL+"/v1/chat/completions", "application/json",
		bytes.NewReader([]byte(`{"model":"org/code","messages":[{"role":"user","content":"hi"}]}`)))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	if msg := errorMessage(t, string(body)); !strings.Contains(msg, "ships its own code, which Dessau does not run") {
		t.Errorf("a client on this Mac was told %q, want the plain reason", msg)
	}

	deadline := time.Now().Add(10 * time.Second)
	var reason string
	for time.Now().Before(deadline) && reason == "" {
		for _, m := range fetchState(t, panel).Models {
			if m.RepoID == id && m.LoadFailure != nil {
				reason = m.LoadFailure.Reason
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(reason, "ships its own code, which Dessau does not run") {
		t.Errorf("the panel's state carries load_failure.reason %q, want the plain reason", reason)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a process was started for a model that ships its own code")
	}
}

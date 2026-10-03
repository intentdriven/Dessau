package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// newCodeModelApp is an App over the real launcher, whose interpreter is a
// shell script that leaves a marker when it runs, with one ready chat model
// whose config.json names a model_file. The marker is how the test knows no
// process was ever started.
func newCodeModelApp(t *testing.T) (*App, string, string) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	marker := filepath.Join(t.TempDir(), "ran")
	if err := os.MkdirAll(filepath.Dir(paths.VenvPython()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.VenvPython(), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{Paths: paths, Config: config.Default()})
	if err != nil {
		t.Fatalf("New: %v", err)
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
		RepoID: id, Path: dir, State: registry.StateReady, Bytes: 1 << 20,
		ContextLength: 131072, KVChargePerToken: 64, ChatTemplate: true,
	}); err != nil {
		t.Fatal(err)
	}
	return a, id, marker
}

// Idle work goes through the pool, so a model that ships its own code is
// refused to it exactly as to a client, before any process starts. The
// refusal is recorded on the model as its load failure, so the context probe
// and the self-test then leave it alone rather than queueing it again and
// again (the 2026-09-21 stuck-probe line), and a later request is refused
// with the reason at once, from the record, without reading the file again.
func TestAModelThatShipsItsOwnCodeIsRefusedToIdleWorkAndThenLeftAlone(t *testing.T) {
	a, id, marker := newCodeModelApp(t)

	_, _, err := selfTestServer{a}.Acquire(context.Background(), id)
	var notReady *runtime.NotReadyError
	if !errors.As(err, &notReady) || !strings.Contains(err.Error(), "ships its own code") {
		t.Fatalf("the self-test's Acquire = %v, want a NotReadyError saying the model ships its own code", err)
	}
	waitFor(t, "the refusal to be recorded on the model", func() bool {
		m, err := a.Registry.Get(id)
		return err == nil && m.LoadFailed()
	})
	m, _ := a.Registry.Get(id)
	if m.LoadFailure.Reason != registry.ErrModelCode.Error() || m.LoadFailure.Transient {
		t.Errorf("LoadFailure = %+v, want the plain reason, standing until a person retries", m.LoadFailure)
	}
	for _, c := range (probeSources{a}).Candidates() {
		if c.RepoID == id {
			t.Error("the context probe still counts the refused model as a candidate")
		}
	}
	for _, r := range (selfTestServer{a}).Ready() {
		if r == id {
			t.Error("the self-test still counts the refused model as ready")
		}
	}
	_, _, err = a.Pool.Acquire(context.Background(), id)
	if !errors.As(err, &notReady) || !strings.Contains(err.Error(), "ships its own code") {
		t.Errorf("a later Acquire = %v, want the recorded reason", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a process was started for a model that ships its own code")
	}
}

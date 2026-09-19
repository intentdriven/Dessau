package main

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/pairing"
)

func testIdentity(t *testing.T) (*pairing.Identity, *pairing.Registry) {
	t.Helper()
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "server-key.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	return id, pairing.NewRegistry(func() config.Config { return cfg })
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

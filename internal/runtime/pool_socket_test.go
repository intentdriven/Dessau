package runtime

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The pool reaches each model server over a socket in a directory only this
// account can open, and hands every caller that socket's transport with the
// Upstream: the readiness probe, the gateway, the self-test and the tool-call
// probe all reach the server through it, and through nothing else. When the
// pool closes, the directory goes with it.
func TestThePoolReachesItsModelServerOverItsPrivateSocket(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := NewPool(PoolOptions{
		Launcher:         l,
		Models:           src,
		MaxResidentBytes: 1 << 30,
		ReadyTimeout:     5 * time.Second,
	})
	closed := false
	t.Cleanup(func() {
		if !closed {
			p.Close()
		}
	})

	up, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	spec := l.specFor("org/m")
	if spec.Socket == "" {
		t.Fatal("the model server was launched with no socket")
	}
	dir := filepath.Dir(spec.Socket)
	if err := checkSocketDir(dir, os.Geteuid()); err != nil {
		t.Errorf("the socket is not in a private directory: %v", err)
	}
	if up.Transport == nil {
		t.Fatal("the Upstream carries no transport to reach its model server with")
	}

	before := l.serverFor("org/m").Completions()
	body := []byte(`{"model":"` + up.ModelArg + `","messages":[{"role":"user","content":"hi"}]}`)
	req, err := http.NewRequest(http.MethodPost, up.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Transport: up.Transport, Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("the Upstream's transport did not reach the model server: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if after := l.serverFor("org/m").Completions(); after != before+1 {
		t.Errorf("the request did not reach the server on the socket: completions %d -> %d", before, after)
	}
	release()

	closed = true
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the socket directory outlived the pool: %v", err)
	}
}

package gateway

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// releasePool is a pool that records what it was asked to release and answers
// with whatever refusal a test gives it.
type releasePool struct {
	stubPool
	mu       sync.Mutex
	released []string
	refuse   error
}

func (p *releasePool) Release(repoID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.refuse != nil {
		return p.refuse
	}
	p.released = append(p.released, repoID)
	return nil
}

func (p *releasePool) releasedIDs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.released...)
}

func unloadGateway(t *testing.T, key string) (*Gateway, *releasePool) {
	t.Helper()
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m"})
	t.Cleanup(fake.Close)
	cfg := config.Default()
	cfg.APIKey = key
	models := &stubModels{models: []registry.Model{
		{RepoID: "org/warm", State: registry.StateReady, Path: "/models/org/warm"},
	}}
	pool := &releasePool{stubPool: stubPool{srv: fake}}
	return New(Options{Config: cfg, Pool: pool, Models: models}), pool
}

// unloadRequest is a POST to the unload route from addr, with a JSON body.
func unloadRequest(addr, body string, hdrs map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:11535/v1/dessau/unload", strings.NewReader(body))
	req.RemoteAddr = addr
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	return req
}

func serveUnload(h http.Handler, r *http.Request) (int, string) {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	b, _ := io.ReadAll(w.Result().Body)
	return w.Code, string(b)
}

const (
	thisMac  = "127.0.0.1:50000"
	aLANHost = "192.168.1.77:50000"
)

// On a keyless install a program on this Mac may unload an idle model, and
// the answer says so (itd-2610031024247803 criterion 1).
func TestAProgramOnThisMacMayUnload(t *testing.T) {
	g, pool := unloadGateway(t, "")
	code, body := serveUnload(g.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil))
	if code != http.StatusOK || !strings.Contains(body, `"unloaded"`) {
		t.Fatalf("status %d, %s", code, body)
	}
	if got := pool.releasedIDs(); len(got) != 1 || got[0] != "org/warm" {
		t.Errorf("released %v, want the model by its own name", got)
	}
}

// Another device on a keyless network is refused whatever is loaded, and the
// refusal is the same answer for a model that exists and one that does not:
// it reveals nothing (criterion 2).
func TestAnotherDeviceOnAKeylessNetworkIsRefusedAndLearnsNothing(t *testing.T) {
	g, pool := unloadGateway(t, "")
	c1, b1 := serveUnload(g.Handler(), unloadRequest(aLANHost, `{"model":"org/warm"}`, nil))
	c2, b2 := serveUnload(g.Handler(), unloadRequest(aLANHost, `{"model":"org/absent"}`, nil))
	if c1 != http.StatusForbidden || c2 != http.StatusForbidden {
		t.Errorf("statuses %d and %d, want 403", c1, c2)
	}
	if b1 != b2 {
		t.Errorf("the refusal differs by model:\n%s\n%s", b1, b2)
	}
	if strings.Contains(b1, "warm") {
		t.Errorf("the refusal names a model: %s", b1)
	}
	if len(pool.releasedIDs()) != 0 {
		t.Error("a refused caller unloaded a model")
	}
}

// With a key, a program elsewhere may unload with it and is refused without
// it; a paired client, admitted on its own key, may unload (criterion 3).
func TestAKeyHolderAndAPairedClientMayUnload(t *testing.T) {
	g, pool := unloadGateway(t, "bh_secret")
	if code, _ := serveUnload(g.Handler(), unloadRequest(aLANHost, `{"model":"org/warm"}`, nil)); code != http.StatusUnauthorized {
		t.Errorf("without the key: status %d, want 401", code)
	}
	if code, body := serveUnload(g.Handler(), unloadRequest(aLANHost, `{"model":"org/warm"}`,
		map[string]string{"Authorization": "Bearer bh_secret"})); code != http.StatusOK {
		t.Errorf("with the key: status %d, %s", code, body)
	}
	// What pairedOnly does for a paired client's request: admitted on its
	// own key, as a keyed one is.
	paired := withAdmittedKeyed(unloadRequest(aLANHost, `{"model":"org/warm"}`, nil), true)
	if code, body := serveUnload(g.routes(), paired); code != http.StatusOK {
		t.Errorf("a paired client: status %d, %s", code, body)
	}
	if n := len(pool.releasedIDs()); n != 2 {
		t.Errorf("%d unloads, want the two admitted", n)
	}
}

// A page in a browser on this Mac is refused and nothing changes
// (criterion 4): a cross-site fetch says so in Sec-Fetch-Site, a form post
// cannot send JSON, and a DNS-rebound page carries its own Host.
func TestAWebPageOnThisMacCannotUnload(t *testing.T) {
	for _, key := range []string{"", "bh_secret"} {
		g, pool := unloadGateway(t, key)
		for name, req := range map[string]*http.Request{
			"cross-site fetch": unloadRequest(thisMac, `{"model":"org/warm"}`,
				map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"}),
			"same-site page": unloadRequest(thisMac, `{"model":"org/warm"}`,
				map[string]string{"Sec-Fetch-Site": "same-site"}),
			"form post": unloadRequest(thisMac, `model=org/warm`,
				map[string]string{"Content-Type": "application/x-www-form-urlencoded"}),
			"rebound host": func() *http.Request {
				r := unloadRequest(thisMac, `{"model":"org/warm"}`, nil)
				r.Host = "evil.example"
				return r
			}(),
		} {
			code, _ := serveUnload(g.Handler(), req)
			if code < 400 {
				t.Errorf("key %q, %s: status %d, want a refusal", key, name, code)
			}
		}
		if n := len(pool.releasedIDs()); n != 0 {
			t.Errorf("key %q: a web page unloaded %d model(s)", key, n)
		}
	}
}

// A model the pool will not let go — pinned, busy, loading, inside its grace,
// or not loaded — is refused at once with 409 and the reason (criteria 5 and
// 6).
func TestAProtectedModelIsRefusedAtOnce(t *testing.T) {
	for _, refusal := range []error{runtime.ErrPinned, runtime.ErrBusy, runtime.ErrLoading, runtime.ErrInGrace, runtime.ErrNotLoaded} {
		g, pool := unloadGateway(t, "")
		pool.refuse = fmt.Errorf("org/warm: %w", refusal)
		code, body := serveUnload(g.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil))
		if code != http.StatusConflict || !strings.Contains(body, refusal.Error()) {
			t.Errorf("%v: status %d, %s", refusal, code, body)
		}
	}
}

// A body that names no model, or a model this server does not serve, is a
// client error, and a malformed one says so.
func TestAnUnloadNamesAServedModel(t *testing.T) {
	g, _ := unloadGateway(t, "")
	for body, want := range map[string]int{
		`{}`:                       http.StatusBadRequest,
		`not json`:                 http.StatusBadRequest,
		`{"model":"org/absent"}`:   http.StatusNotFound,
		`{"model":"warm"}`:         http.StatusOK, // the short name, as a chat request may use
		strings.Repeat("x", 1<<20): http.StatusBadRequest,
	} {
		code, out := serveUnload(g.Handler(), unloadRequest(thisMac, body, nil))
		if code != want {
			t.Errorf("%.40s: status %d, want %d (%s)", body, code, want, out)
		}
	}
}

// No client can delete a model through /v1, the way OpenAI's API deletes
// models: there is no DELETE route, and nothing is unloaded or removed
// (criterion 9).
func TestNoClientCanDeleteAModelThroughTheAPI(t *testing.T) {
	g, pool := unloadGateway(t, "")
	for _, path := range []string{"/v1/models/org/warm", "/v1/models/org%2Fwarm", "/v1/dessau/unload"} {
		req := httptest.NewRequest(http.MethodDelete, "http://127.0.0.1:11535"+path, nil)
		req.RemoteAddr = thisMac
		code, _ := serveUnload(g.Handler(), req)
		if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
			t.Errorf("DELETE %s: status %d", path, code)
		}
	}
	if len(pool.releasedIDs()) != 0 {
		t.Error("a DELETE unloaded a model")
	}
	_ = errors.New
}

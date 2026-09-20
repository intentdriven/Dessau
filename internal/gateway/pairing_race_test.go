package gateway

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/app"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/pairing"
)

// THE FINDING THIS TEST EXISTS FOR (iss-2609190110117982).
//
// App.Config() returns the configuration by value, and a struct copy shares its
// maps. So a handler that writes cfg.Clients writes the RUNNING paired set —
// while the registry reads it on every handshake and every request, and the
// panel iterates it on every poll. A Go map is not concurrency-safe, and this
// is not a race report but a fatal error that takes the process with it, from
// an endpoint nobody has to authenticate to.
//
// The pairing endpoint is exercised end to end here, rather than pairInto
// alone, because the mutation being tested is the handler's read of the live
// configuration and not the validation.
func TestPairingDoesNotWriteTheRunningPairedSet(t *testing.T) {
	stored := config.Default()
	spki := strings.Repeat("A", 43) + "="
	stored.Clients = map[string]config.Client{spki: {Name: "Bob", SPKI: spki, PairedAt: 1}}

	srv, a, reg := newPairingServer(t, stored)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	// The two readers of the live map: the TLS admission path and the panel.
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					reg.Lookup(spki)
					_ = pairedClients(a.Config(), reg)
				}
			}
		}()
	}

	k := newClientKey(t)
	body := `{"name":"Bob's iPad","public_key":"` + base64.StdEncoding.EncodeToString(k.spki) + `"}`
	for range 40 {
		resp := postJSON(t, srv, "/pair", body)
		resp.Body.Close()
	}
	close(stop)
	wg.Wait()

	if got := len(a.Config().Clients); got != 2 {
		t.Errorf("the paired set holds %d clients after one new pairing, want 2", got)
	}
}

// A pairing the server could not save must not be admitted by the live
// registry either (iss-2609190110110994). The answer was already moved behind
// the save; the WRITE has to move with it, or a 500 is full access to the TLS
// port with no API key until the process restarts.
func TestAPairingThatCouldNotBeSavedIsNotAdmitted(t *testing.T) {
	srv, a, reg := newPairingServer(t, config.Default())

	// Make the save fail the way a full disk or a permission would: the
	// directory config.json is written into stops being writable.
	dir := filepath.Dir(a.Paths.Config)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	k := newClientKey(t)
	resp := postJSON(t, srv, "/pair",
		`{"name":"Bob","public_key":"`+base64.StdEncoding.EncodeToString(k.spki)+`"}`)
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Skip("the save succeeded, so there is no failed save to check (running as root?)")
	}
	if _, ok := reg.Lookup(k.fingerprint()); ok {
		t.Error("a pairing the server refused is admitted by the live registry — a 500 on this " +
			"endpoint is full access to the TLS port with no API key")
	}
	if got := len(a.Config().Clients); got != 0 {
		t.Errorf("the running paired set holds %d clients after a pairing that was refused", got)
	}
}

// The pairing endpoint is not a form a web page may post.
//
// A POST with a CORS-simple Content-Type sends no preflight, so without this
// any page anybody on the LAN visits can enrol an attacker's key — and the
// attacker never needs to read the answer, because the paired lookup is on the
// key and they can sign their own certificate for it. That is a wider window
// than adr-2609182357322050 accepted (iss-2609190110118690).
func TestThePairingEndpointIsNotAFormAPageCanPost(t *testing.T) {
	cfg := config.Default()
	id, err := pairing.LoadIdentity(t.TempDir()+"/k.pem", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{Identity: id}
	k := newClientKey(t)
	body := `{"name":"Bob","public_key":"` + base64.StdEncoding.EncodeToString(k.spki) + `"}`

	cases := map[string]func(*http.Request){
		"a CORS-simple content type": func(r *http.Request) {
			r.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		},
		"a form content type": func(r *http.Request) {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		},
		"no content type at all": func(r *http.Request) {},
		"a cross-origin page": func(r *http.Request) {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", "https://example.invalid")
		},
	}
	for what, decorate := range cases {
		r := httptest.NewRequest("POST", "/pair", strings.NewReader(body))
		decorate(r)
		if _, err := readPairRequest(r); err == nil {
			t.Errorf("the pairing endpoint accepted %s", what)
		}
	}

	// And an ordinary client still pairs.
	r := httptest.NewRequest("POST", "/pair", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	req, err := readPairRequest(r)
	if err != nil {
		t.Fatalf("an ordinary pairing was refused: %v", err)
	}
	w := httptest.NewRecorder()
	if _, ok := ctrl.pairInto(&cfg, req, w); !ok {
		t.Fatalf("an ordinary pairing was refused: %d %s", w.Code, w.Body)
	}
	_ = id
}

// A client that pairs again with the key it already holds changes nothing, so
// nothing is written. Without this, an unauthenticated endpoint is a remote
// fsync loop on the operator's settings file (iss-2609190110241925).
func TestRepairingWithAnUnchangedRowSavesNothing(t *testing.T) {
	srv, a, _ := newPairingServer(t, config.Default())
	k := newClientKey(t)
	body := `{"name":"Bob's iPad","public_key":"` + base64.StdEncoding.EncodeToString(k.spki) + `"}`

	resp := postJSON(t, srv, "/pair", body)
	resp.Body.Close()
	if got := a.Config().Clients[k.fingerprint()].Name; got != "Bob's iPad" {
		t.Fatalf("the first pairing did not take: %+v", a.Config().Clients)
	}
	written := settingsFileWrittenAt(t, a)

	for range 5 {
		resp := postJSON(t, srv, "/pair", body)
		resp.Body.Close()
	}
	if got := settingsFileWrittenAt(t, a); !got.Equal(written) {
		t.Errorf("five unchanged pairings rewrote the settings file (was %v, now %v); an endpoint nobody "+
			"has to authenticate to is an fsync loop on the operator's file", written, got)
	}

	// A pairing that CHANGES something still writes.
	resp = postJSON(t, srv, "/pair",
		`{"name":"Bob's iPad Pro","public_key":"`+base64.StdEncoding.EncodeToString(k.spki)+`"}`)
	resp.Body.Close()
	if got := a.Config().Clients[k.fingerprint()].Name; got != "Bob's iPad Pro" {
		t.Errorf("a renaming pairing was swallowed: the client is still %q", got)
	}
}

// newPairingServer is a control plane with the pairing endpoint mounted where
// cmd/dessau mounts it: outside the loopback-only guard, because a client that
// has not paired is not on this Mac.
func newPairingServer(t *testing.T, cfg config.Config) (*httptest.Server, *app.App, *pairing.Registry) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	a, err := app.New(app.Options{Paths: paths, Config: cfg})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "server-key.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	reg := pairing.NewRegistry(a.Config)
	ctrl := &Control{App: a, Identity: id, Clients: reg}
	mux := http.NewServeMux()
	ctrl.Routes(mux)
	mux.Handle("/pair", ctrl.PairHandler())
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, a, reg
}

// settingsFileWrittenAt is when config.json was last written.
func settingsFileWrittenAt(t *testing.T, a *app.App) time.Time {
	t.Helper()
	info, err := os.Stat(a.Paths.Config)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}

// selfSignedFor is a certificate a client mints for its own key, which is what
// an attacker holding a paired key would present.
func selfSignedFor(t *testing.T, k clientKey) tls.Certificate {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(7),
		Subject:      pkix.Name{CommonName: "whatever it likes"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.priv.PublicKey, k.priv)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k.priv}
}

// The minted certificate is not what admits a client: the paired lookup is on
// the KEY, so a client holding that key is admitted under a certificate it
// signed itself. That is sound — possession of the private key is proved by
// the handshake — and it is written down here because it is what makes every
// finding above as sharp as it is (adr-2609182357322050, decision 3).
func TestAPairedKeyIsWhatAdmits_NotTheMintedCertificate(t *testing.T) {
	var reg *pairing.Registry
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("served")) })
	srv := newPairedServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pairedOnly(reg, mux, nil, nil).ServeHTTP(w, r)
	}))
	reg = srv.registry

	k := newClientKey(t)
	srv.pair(t, k, "Bob's iPad") // minted, and then deliberately not used
	selfSigned := selfSignedFor(t, k)
	resp, err := srv.client(selfSigned).Get("https://" + srv.tlsAddr + "/v1/models")
	if err != nil {
		t.Fatalf("a paired key was refused under its own certificate: %v — if this is now the intended "+
			"behaviour, the ADR and the docs page say otherwise", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("a paired key under its own certificate got %d", resp.StatusCode)
	}
	if _, err := x509.ParseCertificate(selfSigned.Certificate[0]); err != nil {
		t.Fatal(err)
	}
}

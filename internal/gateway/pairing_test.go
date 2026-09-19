package gateway

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Gropius/internal/app"
	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/pairing"
)

// clientKey is a P-256 keypair standing in for one a client made in its
// Keychain, with its public key in the shape the pairing call carries it.
type clientKey struct {
	priv *ecdsa.PrivateKey
	spki []byte
}

func newClientKey(t *testing.T) clientKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return clientKey{priv: k, spki: der}
}

func (c clientKey) fingerprint() string { return pairing.Fingerprint(c.spki) }

// pairedServer stands the whole arrangement up: an identity, a registry over a
// live configuration, a TLS listener with the paired check, and the pairing
// endpoint on a plain one.
type pairedServer struct {
	identity *pairing.Identity
	registry *pairing.Registry
	cfg      *config.Config
	tlsAddr  string
}

func (p *pairedServer) pair(t *testing.T, k clientKey, name string) tls.Certificate {
	t.Helper()
	pub, err := pairing.PublicKeyFromSPKI(k.spki)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := p.identity.MintLeaf(pub, name)
	if err != nil {
		t.Fatal(err)
	}
	if p.cfg.Clients == nil {
		p.cfg.Clients = map[string]config.Client{}
	}
	p.cfg.Clients[k.fingerprint()] = config.Client{
		Name: name, SPKI: k.fingerprint(), PairedAt: time.Now().Unix(),
	}
	return tls.Certificate{Certificate: [][]byte{leaf}, PrivateKey: k.priv}
}

func newPairedServer(t *testing.T, handler http.Handler) *pairedServer {
	t.Helper()
	cfg := config.Default()
	p := &pairedServer{cfg: &cfg}
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "server-key.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	p.identity = id
	p.registry = pairing.NewRegistry(func() config.Config { return *p.cfg })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: handler}
	go srv.Serve(tls.NewListener(ln, p.registry.TLSConfig(id)))
	t.Cleanup(func() { srv.Close() })
	p.tlsAddr = ln.Addr().String()
	return p
}

func (p *pairedServer) client(cert tls.Certificate) *http.Client {
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true, // the pin is the client's business; this test is the server's
		},
	}}
}

// THE OBLIGATION adr-2609182357322050 leaves: a revoked client is refused on
// its NEXT REQUEST — not at its next connection.
//
// This test drives both requests down one reused connection deliberately. A
// version that closed and reopened between them would pass over the bug it
// exists to find: every handshake-level hook Go offers is per connection, and a
// chat client holds one open because it streams.
func TestARevokedClientIsRefusedOnItsNextRequestOverAConnectionItAlreadyHad(t *testing.T) {
	var reg *pairing.Registry
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "served")
	})
	var srv *pairedServer
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pairedOnly(reg, mux, nil, nil).ServeHTTP(w, r)
	})
	srv = newPairedServer(t, handler)
	reg = srv.registry

	k := newClientKey(t)
	cert := srv.pair(t, k, "Bob's iPad")
	hc := srv.client(cert)

	first, err := hc.Get("https://" + srv.tlsAddr + "/v1/models")
	if err != nil {
		t.Fatalf("a paired client was refused: %v", err)
	}
	body, _ := io.ReadAll(first.Body)
	first.Body.Close()
	if first.StatusCode != http.StatusOK || string(body) != "served" {
		t.Fatalf("a paired client got %d %q", first.StatusCode, body)
	}

	// Revoke, without touching the connection.
	delete(srv.cfg.Clients, k.fingerprint())

	second, err := hc.Get("https://" + srv.tlsAddr + "/v1/models")
	if err != nil {
		// A transport error is an acceptable refusal, but the usual case is a
		// status, because the connection is still up.
		return
	}
	defer second.Body.Close()
	if second.StatusCode == http.StatusOK {
		t.Error("a revoked client was served on a connection it already had open — " +
			"the check is per connection, and the request path is not checking again")
	}
}

// RequireAnyClientCert verifies no chain, so a certificate's common name is
// whatever the presenter wrote in it. The lookup is on the key and on nothing
// else.
func TestASelfSignedLeafCarryingAPairedNameIsRefused(t *testing.T) {
	var reg *pairing.Registry
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "served") })
	var srv *pairedServer
	srv = newPairedServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pairedOnly(reg, mux, nil, nil).ServeHTTP(w, r)
	}))
	reg = srv.registry
	srv.pair(t, newClientKey(t), "Bob's iPad")

	// An impostor mints its own certificate with the same name.
	impostor := newClientKey(t)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Bob's iPad"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &impostor.priv.PublicKey, impostor.priv)
	if err != nil {
		t.Fatal(err)
	}
	hc := srv.client(tls.Certificate{Certificate: [][]byte{der}, PrivateKey: impostor.priv})
	resp, err := hc.Get("https://" + srv.tlsAddr + "/v1/models")
	if err != nil {
		return // refused at the handshake, which is the earliest and best place
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("a self-signed certificate carrying a paired client's name was served — " +
			"the lookup is reading the name rather than the key")
	}
}

// The pairing endpoint is unauthenticated by design, so everything it takes is
// bounded before it is parsed and nothing malformed is ever written.
func TestThePairingEndpointRefusesWhatItCannotStore(t *testing.T) {
	cfg := config.Default()
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{Identity: id}
	good := newClientKey(t)
	goodKey := base64.StdEncoding.EncodeToString(good.spki)

	cases := map[string]string{
		"an empty name":            `{"name":"","public_key":"` + goodKey + `"}`,
		"a name with a newline":    `{"name":"Bob\nAdmin","public_key":"` + goodKey + `"}`,
		"an overlong name":         `{"name":"` + strings.Repeat("b", config.MaxClientNameBytes+1) + `","public_key":"` + goodKey + `"}`,
		"no key":                   `{"name":"Bob"}`,
		"a key that is not base64": `{"name":"Bob","public_key":"!!!!"}`,
		"a key of the wrong kind":  `{"name":"Bob","public_key":"` + base64.StdEncoding.EncodeToString([]byte("hello")) + `"}`,
		"not JSON":                 `{`,
	}
	for what, body := range cases {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/pair", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		req, err := readPairRequest(r)
		if err != nil {
			continue // refused before it was even shaped, which is a refusal
		}
		if _, ok := ctrl.pairInto(&cfg, req, w); ok {
			t.Errorf("the pairing endpoint accepted %s", what)
		}
		if len(cfg.Clients) != 0 {
			t.Fatalf("%s was written to the paired set", what)
		}
	}
}

// The ceiling is what stops an unauthenticated endpoint growing config.json
// until the next start cannot read it.
func TestThePairingEndpointStopsAtTheCeiling(t *testing.T) {
	cfg := config.Default()
	cfg.Clients = map[string]config.Client{}
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{Identity: id}
	for range config.MaxClients {
		k := newClientKey(t)
		cfg.Clients[k.fingerprint()] = config.Client{Name: "c", SPKI: k.fingerprint(), PairedAt: 1}
	}
	k := newClientKey(t)
	body := `{"name":"one too many","public_key":"` + base64.StdEncoding.EncodeToString(k.spki) + `"}`
	w := httptest.NewRecorder()
	_, _ = ctrl.pairInto(&cfg, mustPairRequest(t, body), w)
	if w.Code == http.StatusOK {
		t.Error("the pairing endpoint paired a client over the ceiling")
	}
	if len(cfg.Clients) != config.MaxClients {
		t.Errorf("the paired set holds %d, want %d", len(cfg.Clients), config.MaxClients)
	}
}

// A client that pairs twice with the same key updates its row rather than
// making a second one nobody can tell apart.
func TestPairingTwiceWithOneKeyUpdatesTheSameClient(t *testing.T) {
	cfg := config.Default()
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{Identity: id}
	k := newClientKey(t)
	body := func(name string) string {
		return `{"name":"` + name + `","public_key":"` + base64.StdEncoding.EncodeToString(k.spki) + `"}`
	}
	for _, name := range []string{"Bob's iPad", "Bob's iPad Pro"} {
		w := httptest.NewRecorder()
		if _, ok := ctrl.pairInto(&cfg, mustPairRequest(t, body(name)), w); !ok {
			t.Fatalf("pairing %q was refused: %d %s", name, w.Code, w.Body)
		}
	}
	if len(cfg.Clients) != 1 {
		t.Fatalf("the paired set holds %d clients, want one", len(cfg.Clients))
	}
	if got := cfg.Clients[k.fingerprint()].Name; got != "Bob's iPad Pro" {
		t.Errorf("the client is named %q, want the name it paired with last", got)
	}
}

// The answer carries what the client needs and says what the fingerprint in it
// is for.
func TestPairingAnswersWithTheLeafAndTheServersFingerprint(t *testing.T) {
	cfg := config.Default()
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{Identity: id}
	k := newClientKey(t)
	w := httptest.NewRecorder()
	answer, ok := ctrl.pairInto(&cfg, mustPairRequest(t, `{"name":"Bob","public_key":"`+base64.StdEncoding.EncodeToString(k.spki)+`"}`), w)
	if !ok {
		t.Fatalf("pairing was refused: %d %s", w.Code, w.Body)
	}
	der, err := base64.StdEncoding.DecodeString(answer.Leaf)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("the answer's leaf is not a certificate: %v", err)
	}
	if got := pairing.FingerprintOf(leaf); got != k.fingerprint() {
		t.Errorf("the minted leaf carries %q, want the client's own key %q", got, k.fingerprint())
	}
	if answer.Fingerprint != id.Fingerprint() {
		t.Errorf("the answer names %q as the server's fingerprint, want %q", answer.Fingerprint, id.Fingerprint())
	}
	if answer.TLSPort != cfg.EffectiveTLSPort() {
		t.Errorf("the answer names port %d, want %d", answer.TLSPort, cfg.EffectiveTLSPort())
	}
}

// A pairing the server could not record must not answer as though it had.
//
// The client takes the 200 and the certificate in it as proof that it is
// paired. If the save failed, every request it then makes is refused as
// unpaired, with nothing on either side to explain it (iss-2609190100210971).
func TestAPairingThatCouldNotBeSavedIsNotAnsweredWithACertificate(t *testing.T) {
	cfg := config.Default()
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{Identity: id}
	k := newClientKey(t)
	w := httptest.NewRecorder()
	answer, ok := ctrl.pairInto(&cfg, mustPairRequest(t, `{"name":"Bob","public_key":"`+base64.StdEncoding.EncodeToString(k.spki)+`"}`), w)
	if !ok {
		t.Fatalf("an ordinary pairing was refused: %d %s", w.Code, w.Body)
	}
	if answer.Leaf == "" {
		t.Fatal("the answer carries no certificate")
	}
	// Nothing has been written yet: the caller writes it, and only after the
	// save it has not made yet succeeds.
	if w.Body.Len() != 0 {
		t.Errorf("the validation step wrote a %d-byte answer before anything was saved: %s", w.Body.Len(), w.Body)
	}
}

// mustPairRequest is a well-formed pairing request, so a test about what
// pairInto does with the CONTENT does not have to restate what readPairRequest
// checks about the caller.
func mustPairRequest(t *testing.T, body string) pairRequest {
	t.Helper()
	r := httptest.NewRequest("POST", "/pair", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	req, err := readPairRequest(r)
	if err != nil {
		t.Fatalf("a well-formed request was refused before it was read: %v", err)
	}
	return req
}

// A bound that only stops READING is not a bound on what is accepted
// (iss-2609190200099532). A streaming decoder stops at the first complete JSON
// value, so a request whose opening bytes are a well-formed pairing and whose
// remainder is anything at all used to pair — on an endpoint that is
// unauthenticated by design and writes the settings file. The refusal has to be
// the endpoint's answer, not a side effect of running out of bytes to read.
func TestThePairingEndpointRefusesABodyOverItsLimit(t *testing.T) {
	srv, a, _ := newPairingServer(t, config.Default())
	k := newClientKey(t)
	within := `{"name":"Bob's iPad","public_key":"` + base64.StdEncoding.EncodeToString(k.spki) + `"}`
	over := within + strings.Repeat("x", maxPairBodyBytes)

	resp := postJSON(t, srv, "/pair", over)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("a pairing body over the limit answered %d, want 413", resp.StatusCode)
	}
	if got := len(a.Config().Clients); got != 0 {
		t.Errorf("a pairing body over the limit paired %d clients; it must write nothing", got)
	}

	// And the bound is on the size and on nothing else: the same pairing within
	// it is still accepted, so the refusal above is not this test passing for
	// the wrong reason.
	ok := postJSON(t, srv, "/pair", within)
	defer ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("an ordinary pairing answered %d, want 200", ok.StatusCode)
	}
	if got := len(a.Config().Clients); got != 1 {
		t.Errorf("an ordinary pairing left %d clients paired, want one", got)
	}
}

// The three lines a pairing writes, at the level the server ships at
// (iss-2609190200097326).
//
// The operator's only account of who may connect is the panel and this log. A
// line written below the shipped level is a line that does not exist for them,
// and a line that carried the client's whole key fingerprint would put the one
// value the panel comparison rests on into a file that gets copied into bug
// reports. So: every pairing event is written at the shipped level, names the
// client by the name it chose where there is one to trust, and carries the
// short fingerprint and never the whole of it.
func TestEveryPairingEventIsLoggedAtTheShippedLevelByNameAndNeverByKey(t *testing.T) {
	var logged bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{
		Level: config.Default().SlogLevel(),
	}))

	paths := config.NewPaths(t.TempDir())
	a, err := app.New(app.Options{Paths: paths, Config: config.Default()})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "server-key.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	reg := pairing.NewRegistry(a.Config)
	ctrl := &Control{App: a, Identity: id, Clients: reg, Log: log}
	mux := http.NewServeMux()
	ctrl.Routes(mux)
	mux.Handle("/pair", ctrl.PairHandler())
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	const name = "Bob's iPad"
	k := newClientKey(t)
	resp := postJSON(t, srv, "/pair", `{"name":"`+name+`","public_key":"`+
		base64.StdEncoding.EncodeToString(k.spki)+`"}`)
	var answer pairAnswer
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	revoked := postJSON(t, srv, "/api/clients/revoke", `{"fingerprint":"`+k.fingerprint()+`"}`)
	if revoked.StatusCode != http.StatusOK {
		t.Fatalf("the revocation answered %d", revoked.StatusCode)
	}
	revoked.Body.Close()

	// The client does not know yet. It still holds the certificate and the
	// connection it already had, and its next request is the one that is
	// refused — which is exactly the event an operator who has just revoked
	// somebody is watching for.
	der, err := base64.StdEncoding.DecodeString(answer.Leaf)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	h := pairedOnly(reg, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("a revoked client was served")
	}), log, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("a revoked client's request answered %d, want 403", w.Code)
	}

	lines := map[string]string{}
	for _, line := range strings.Split(logged.String(), "\n") {
		for _, msg := range []string{"client paired", "client revoked", "refused a request"} {
			if strings.Contains(line, `msg="`+msg) || strings.Contains(line, "msg="+msg) {
				lines[msg] = line
			}
		}
	}
	for _, msg := range []string{"client paired", "client revoked", "refused a request"} {
		if lines[msg] == "" {
			t.Fatalf("nothing at the shipped level says %q; the whole log is:\n%s", msg, logged.String())
		}
	}
	for _, msg := range []string{"client paired", "client revoked"} {
		if !strings.Contains(lines[msg], name) {
			t.Errorf("%q does not name the client: %s", msg, lines[msg])
		}
	}
	for msg, line := range lines {
		if !strings.Contains(line, shortFingerprint(k.fingerprint())) {
			t.Errorf("%q carries no fingerprint, so nothing tells two clients of one name apart: %s", msg, line)
		}
		if strings.Contains(line, k.fingerprint()) {
			t.Errorf("%q carries the client's whole key fingerprint: %s", msg, line)
		}
	}
}

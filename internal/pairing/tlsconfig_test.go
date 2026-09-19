package pairing_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"maps"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/pairing"
)

// handshakeServer is TLSConfig standing on its own: the server's identity, a
// registry over a paired set this test moves, a TLS listener built from
// nothing but Registry.TLSConfig, and a count of the requests that got
// through.
//
// The count is the point. Everything downstream of this listener — the
// per-request check, the refusal log line in internal/gateway/pairing.go — is
// bounded by who reaches it, and what bounds that is VerifyConnection and
// nothing else. A test that only asserted a 403 would pass just as happily
// with the handshake wide open.
type handshakeServer struct {
	identity *pairing.Identity
	mu       sync.Mutex
	clients  map[string]config.Client
	addr     string
	served   atomic.Int64
}

func newHandshakeServer(t *testing.T) *handshakeServer {
	t.Helper()
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "server-key.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	s := &handshakeServer{identity: id, clients: map[string]config.Client{}}
	reg := pairing.NewRegistry(s.config)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.served.Add(1)
		io.WriteString(w, "served")
	})}
	go srv.Serve(tls.NewListener(ln, reg.TLSConfig(id)))
	t.Cleanup(func() { srv.Close() })
	s.addr = ln.Addr().String()
	return s
}

// config is the live view the registry reads on every handshake. The paired
// set is copied out under the lock so that revoking in the middle of a test is
// not a race with a handshake in flight.
func (s *handshakeServer) config() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := config.Default()
	c.Clients = maps.Clone(s.clients)
	return c
}

// key is a client keypair with a certificate this server minted for it. The
// pairing itself is a separate step, so that a key holding a perfectly valid
// minted leaf can still be an unpaired one.
type key struct {
	priv *ecdsa.PrivateKey
	spki []byte
	cert tls.Certificate
}

func (s *handshakeServer) newKey(t *testing.T, name string) key {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := pairing.PublicKeyFromSPKI(spki)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := s.identity.MintLeaf(pub, name)
	if err != nil {
		t.Fatal(err)
	}
	return key{
		priv: priv,
		spki: spki,
		cert: tls.Certificate{Certificate: [][]byte{leaf}, PrivateKey: priv},
	}
}

func (k key) fingerprint() string { return pairing.Fingerprint(k.spki) }

func (s *handshakeServer) pair(k key, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[k.fingerprint()] = config.Client{
		Name: name, SPKI: k.fingerprint(), PairedAt: time.Now().Unix(),
	}
}

func (s *handshakeServer) revoke(k key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, k.fingerprint())
}

// get makes a request on a CONNECTION OF ITS OWN, so that what it reports is
// the handshake and never a session this test opened earlier.
func (s *handshakeServer) get(certs ...tls.Certificate) (*http.Response, error) {
	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			Certificates:       certs,
			InsecureSkipVerify: true, // the client's pin is the client's business; this is the server's side
		},
	}
	defer tr.CloseIdleConnections()
	c := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	return c.Do(mustGet("https://" + s.addr + "/v1/models"))
}

func mustGet(url string) *http.Request {
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}
	return r
}

// A key this server has not paired is refused at the HANDSHAKE, so no request
// of its is ever served — and none is ever logged (iss-2609190225574067).
//
// The certificate here is one this server itself minted, which is the honest
// version of the test: under RequireAnyClientCert the leaf proves nothing at
// all, so what refuses this client has to be the paired-set lookup in
// VerifyConnection. Relax that to a plain RequireAnyClientCert and every peer
// on the network is admitted with a certificate it signed itself, which is the
// unbounded key space the gateway's rate-limited refusal line rests on not
// existing.
func TestAnUnpairedKeyIsRefusedAtTheHandshake(t *testing.T) {
	s := newHandshakeServer(t)
	k := s.newKey(t, "Carol's laptop") // minted, never paired

	resp, err := s.get(k.cert)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("an unpaired client completed a handshake and was answered %d", resp.StatusCode)
	}
	if n := s.served.Load(); n != 0 {
		t.Errorf("an unpaired client reached the handler %d time(s): the refusal is happening "+
			"per request, so every peer on the network is a request this server has to answer", n)
	}
}

// A client presenting no certificate at all does not get in either. This is
// ClientAuth's own doing rather than VerifyConnection's, and it is asserted
// here so that a change to either is a failing test.
func TestAHandshakeWithoutAClientCertificateIsRefused(t *testing.T) {
	s := newHandshakeServer(t)

	resp, err := s.get()
	if err == nil {
		resp.Body.Close()
		t.Fatalf("a client presenting no certificate was answered %d", resp.StatusCode)
	}
	if n := s.served.Load(); n != 0 {
		t.Errorf("a client presenting no certificate reached the handler %d time(s)", n)
	}
}

// The other half of the same check: a paired key is admitted. Without this the
// refusals above would pass on a server that refused everybody.
func TestAPairedKeyIsAdmittedAtTheHandshake(t *testing.T) {
	s := newHandshakeServer(t)
	k := s.newKey(t, "Bob's iPad")
	s.pair(k, "Bob's iPad")

	resp, err := s.get(k.cert)
	if err != nil {
		t.Fatalf("a paired client was refused: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "served" {
		t.Fatalf("a paired client got %d %q", resp.StatusCode, body)
	}
	if n := s.served.Load(); n != 1 {
		t.Errorf("a paired client reached the handler %d time(s), want 1", n)
	}
}

// Revocation is read live, so a client taken off the paired set does not get
// through a NEW handshake — the connection it held is the request path's
// business (internal/gateway/pairing.go), and this is the other one.
func TestARevokedKeyIsRefusedAtItsNextHandshake(t *testing.T) {
	s := newHandshakeServer(t)
	k := s.newKey(t, "Bob's iPad")
	s.pair(k, "Bob's iPad")

	resp, err := s.get(k.cert)
	if err != nil {
		t.Fatalf("a paired client was refused before it was revoked: %v", err)
	}
	resp.Body.Close()

	s.revoke(k)

	resp, err = s.get(k.cert)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("a revoked client completed a fresh handshake and was answered %d", resp.StatusCode)
	}
	if n := s.served.Load(); n != 1 {
		t.Errorf("the handler was reached %d time(s), want 1: the revoked client got through", n)
	}
}

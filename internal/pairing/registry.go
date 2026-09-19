package pairing

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"sync"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
)

// Registry answers the one question both the TLS handshake and every request
// ask: is the key this client is presenting one this server has paired?
//
// The paired set itself is read live from the configuration, so a revocation
// saved by the panel is in force for the next lookup without anything being
// restarted or invalidated. What the registry owns is the OTHER half, the half
// that is deliberately not in the configuration: when each client was last
// heard from. That changes on every request, and writing it would fsync the
// operator's settings file once per chat completion and race every save they
// make, so it lives here and the panel reads it from here.
type Registry struct {
	cfg  func() config.Config
	mu   sync.Mutex
	seen map[string]time.Time
	now  func() time.Time
}

// NewRegistry builds a registry over a live view of the settings.
func NewRegistry(cfg func() config.Config) *Registry {
	return &Registry{cfg: cfg, seen: map[string]time.Time{}, now: time.Now}
}

// Lookup is the paired set as it stands, without recording a sighting.
func (r *Registry) Lookup(spki string) (config.Client, bool) {
	c, ok := r.cfg().Clients[spki]
	return c, ok
}

// Sight is Lookup, and records that this client was heard from now. It is what
// the request path calls, so that "last seen" means the last request and not
// the last connection.
func (r *Registry) Sight(spki string) (config.Client, bool) {
	c, ok := r.Lookup(spki)
	if !ok {
		return config.Client{}, false
	}
	r.mu.Lock()
	r.seen[spki] = r.now()
	r.mu.Unlock()
	return c, true
}

// LastSeen is when a client was last heard from in this run, and whether it has
// been heard from at all. A server that has just started has heard from nobody,
// and the panel says so rather than showing a time it does not have.
func (r *Registry) LastSeen(spki string) (time.Time, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.seen[spki]
	return t, ok
}

// Forget drops a client's sighting, so a revoked client does not leave a last
// seen behind for whoever pairs the same name next.
func (r *Registry) Forget(spki string) {
	r.mu.Lock()
	delete(r.seen, spki)
	r.mu.Unlock()
}

// ErrNotPaired is what a client that is not in the paired set is refused with.
var ErrNotPaired = errors.New("this client is not paired with this server")

// TLSConfig is the server side of the pinned mutual TLS
// (adr-2609182357322050).
//
// The paired check is in VerifyConnection and NOT in VerifyPeerCertificate, and
// this is not a style choice. Go does not call VerifyPeerCertificate on a
// resumed connection at all: it restores the peer certificates from the session
// ticket and checks only NotAfter, and under RequireAnyClientCert not even the
// chain check runs. Measured against the standard library, a revoked client was
// served on every resumed request while VerifyPeerCertificate ran once in four.
// SessionTicketsDisabled is set as well, so nothing here rests on one hook.
//
// Neither hook is enough on its own either, and that is why the request path
// checks again: both are per CONNECTION, and an HTTP/1.1 keep-alive or an
// HTTP/2 stream outlives them for as long as the client keeps the socket open —
// which a chat client does, because it streams. "Refused on its next request"
// is made true by the per-request lookup and by nothing else.
//
// No chain is built and none is wanted: the client's certificate was minted by
// this server from a key it recorded, and what is checked is that key. A
// certificate's common name and serial are whatever the presenter chose, since
// RequireAnyClientCert verifies nothing, so they are never what is looked up.
//
// Said the other way round, because it is the consequence and not the rule: a
// client holding a paired key is admitted under a certificate IT signed, with
// any name in it. The minted leaf is a convenience, not a credential — the
// credential is the private key, proved by the handshake — so a key that ever
// leaves the device it was made on is the whole of an identity, and revoking
// it on the panel is the only thing that answers that. docs/pairing-explained.md
// says so in the operator's own terms.
func (r *Registry) TLSConfig(id *Identity) *tls.Config {
	return &tls.Config{
		Certificates:           []tls.Certificate{id.TLSCertificate()},
		ClientAuth:             tls.RequireAnyClientCert,
		MinVersion:             tls.VersionTLS12,
		SessionTicketsDisabled: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return ErrNotPaired
			}
			if _, ok := r.Lookup(FingerprintOf(cs.PeerCertificates[0])); !ok {
				return ErrNotPaired
			}
			return nil
		},
	}
}

// PeerFingerprint is the pin of the certificate a request arrived under, and
// whether it arrived under one at all.
func PeerFingerprint(peer []*x509.Certificate) (string, bool) {
	if len(peer) == 0 {
		return "", false
	}
	return FingerprintOf(peer[0]), true
}

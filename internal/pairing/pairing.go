// Package pairing holds the server's own TLS identity and the client
// certificates it mints (adr-2609182357322050).
//
// Three things live here and nothing else does: the server's private key,
// which persists across restarts because it is what every paired client pinned;
// the self-signed leaf, which is derived from that key at every start and may
// be reissued freely; and the SPKI fingerprint, which is the one identifier
// both sides of a pairing agree on.
//
// The fingerprint is RFC 7469's — base64(sha256(DER SubjectPublicKeyInfo)) —
// and it is computed in exactly one place, because the client computes the same
// value on the other side of the wire from an X9.63 point with the fixed P-256
// header prepended. Two spellings of this hash would make pairing appear to
// succeed and every later connection fail.
//
// Standard library only. No certificate authority is involved and none is asked
// for: the client pins the key it saw and the server checks the key it
// recorded, and neither builds a chain.
package pairing

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
)

// maxKeyBytes caps what LoadIdentity will read. A P-256 key in PEM is about
// 240 bytes; anything approaching this is broken or hostile.
const maxKeyBytes = 64 << 10

// leafValidity is how long a minted certificate lasts, and it is deliberately
// long.
//
// The leaf carries the key; the PAIRED SET carries the authorization. Alice
// revokes a client by taking it off that list, which takes effect on its next
// request, so a short leaf buys no revocation that the list does not already
// give — and costs one this design cannot pay: the client cannot mint its own
// certificate, so an expiry is a silent de-pairing with no renewal path. Go
// checks NotAfter on a resumed connection, so an expired leaf would start
// failing handshakes with nothing to explain it.
const leafValidity = 10 * 365 * 24 * time.Hour

// Fingerprint is the RFC 7469 pin of a DER SubjectPublicKeyInfo:
// base64(sha256(spki)). It is the only place this hash is computed.
func Fingerprint(spki []byte) string {
	sum := sha256.Sum256(spki)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// FingerprintOf is Fingerprint for a parsed certificate — the form the TLS
// callbacks and the request path want, so neither reaches for the raw field.
func FingerprintOf(cert *x509.Certificate) string {
	return Fingerprint(cert.RawSubjectPublicKeyInfo)
}

// Identity is the server's key and the leaf derived from it.
type Identity struct {
	key         *ecdsa.PrivateKey
	leaf        *x509.Certificate
	leafDER     []byte
	fingerprint string
}

// LoadIdentity reads the server's key from path, generating and saving one when
// the file is absent, and derives a leaf for the given SANs.
//
// The key is read through the same hardened path config.json is read through:
// a symlink is refused rather than followed, a FIFO cannot wedge the start, and
// a file this account does not own — or one any other account could have read —
// is refused rather than adopted. In shared-cache mode the data root is
// group-writable at mode 3775, where the sticky bit stops a co-tenant deleting
// a file but not creating one, so a key that is merely "present" is not a key.
//
// The leaf is derived and never stored. A renamed Mac or a new address reissues
// it and no client re-pairs, because what a client pinned is the KEY's
// fingerprint. Deleting the key file is the one thing that re-pairs everybody.
func LoadIdentity(path string, sans []string) (*Identity, error) {
	key, err := loadOrCreateKey(path)
	if err != nil {
		return nil, err
	}
	leafDER, err := selfSign(key, sans)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, err
	}
	return &Identity{
		key:         key,
		leaf:        leaf,
		leafDER:     leafDER,
		fingerprint: FingerprintOf(leaf),
	}, nil
}

// Fingerprint is what a client pins and what the panel shows.
func (i *Identity) Fingerprint() string { return i.fingerprint }

// Leaf is the certificate this server presents.
func (i *Identity) Leaf() *x509.Certificate { return i.leaf }

// TLSCertificate is the identity in the shape crypto/tls wants.
func (i *Identity) TLSCertificate() tls.Certificate {
	return tls.Certificate{Certificate: [][]byte{i.leafDER}, PrivateKey: i.key, Leaf: i.leaf}
}

// MintLeaf signs a client certificate for a public key the client sent, under
// the name the client chose.
//
// There is no certificate signing request, and that is not an omission: no
// public Apple API mints a certificate, so the client sends its raw public key
// and proof of possession is the mutual-TLS handshake itself — a client that
// does not hold the private key cannot complete it.
func (i *Identity) MintLeaf(pub crypto.PublicKey, name string) ([]byte, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(leafValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return x509.CreateCertificate(rand.Reader, tmpl, i.leaf, pub, i.key)
}

// PublicKeyFromSPKI parses the DER SubjectPublicKeyInfo a client sends, and
// refuses anything that is not a P-256 public key.
//
// The curve is checked rather than assumed. The Secure Enclave produces P-256
// and nothing else, Apple's TLS stack will not use an Ed25519 certificate, and
// a key of another kind reaching x509.CreateCertificate would mint a
// certificate no client could ever present — a pairing that appears to work and
// never connects.
func PublicKeyFromSPKI(der []byte) (crypto.PublicKey, error) {
	if len(der) == 0 {
		return nil, errors.New("the public key is empty")
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("the public key is not a SubjectPublicKeyInfo: %w", err)
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok || ec.Curve != elliptic.P256() {
		return nil, errors.New("the public key is not a P-256 key")
	}
	return ec, nil
}

func loadOrCreateKey(path string) (*ecdsa.PrivateKey, error) {
	b, info, err := config.ReadRegularInfo(path, maxKeyBytes)
	switch {
	case err == nil:
		if err := config.PrivateToThisAccount(info); err != nil {
			return nil, fmt.Errorf("refusing to use %s: %s — delete it and a new key will be made, which pairs every client again", filepath.Base(path), err)
		}
		block, _ := pem.Decode(b)
		if block == nil || block.Type != "EC PRIVATE KEY" {
			return nil, fmt.Errorf("%s is not a private key this server wrote", filepath.Base(path))
		}
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
		return key, nil
	case errors.Is(err, os.ErrNotExist):
		return createKey(path)
	default:
		return nil, err
	}
}

func createKey(path string) (*ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := config.WriteSecretFile(path, pemBytes); err != nil {
		return nil, err
	}
	return key, nil
}

// selfSign derives the served leaf from the key. Loopback and this Mac's own
// name are always in it, so the panel and a client on this Mac can reach the
// TLS port whatever the bind turned out to be.
func selfSign(key *ecdsa.PrivateKey, sans []string) ([]byte, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "Dessau"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(leafValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	for _, s := range sans {
		if s == "" {
			continue
		}
		if ip := net.ParseIP(s); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
			continue
		}
		tmpl.DNSNames = append(tmpl.DNSNames, s)
	}
	tmpl.DNSNames = append(tmpl.DNSNames, "localhost")
	return x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
}

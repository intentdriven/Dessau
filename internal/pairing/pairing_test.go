package pairing_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/Dessau/internal/pairing"
)

// The server's KEY is what a client pins, so it has to outlive a restart. The
// leaf is derived from it at every start and may change freely — new addresses,
// a renamed Mac, a new validity — without a single client re-pairing
// (adr-2609182357322050, decision 3). Regenerating the key is the one thing
// that re-pairs everybody.
func TestTheServerKeyPersistsAcrossRestartsAndTheLeafIsReissued(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server-key.pem")

	first, err := pairing.LoadIdentity(path, []string{"alices-mac.local"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := pairing.LoadIdentity(path, []string{"alices-mac.local", "192.0.2.5"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() != second.Fingerprint() {
		t.Errorf("the fingerprint moved across a restart: %q then %q — every paired client would have to pair again",
			first.Fingerprint(), second.Fingerprint())
	}
	if string(first.Leaf().Raw) == string(second.Leaf().Raw) {
		t.Error("the leaf was not reissued although the addresses changed; it is derived, not stored")
	}
	if got := second.Leaf().DNSNames; len(got) == 0 || got[0] != "alices-mac.local" {
		t.Errorf("the reissued leaf carries DNS names %v, want the ones it was asked for", got)
	}
}

// A private key another account could have read is not a private key. The
// shared install leaves the root group-writable at mode 3775, so the key is
// held to the same rule config.json is held to.
func TestAKeyFileAnotherAccountCouldReadIsRefused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server-key.pem")
	if _, err := pairing.LoadIdentity(path, nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("the key file's mode is %#o, want 0600", perm)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := pairing.LoadIdentity(path, nil); err == nil {
		t.Error("a world-readable key file was accepted; a key another account could have read is not a key")
	}
}

// A symlink planted where the key goes is refused rather than followed.
func TestASymlinkWhereTheKeyGoesIsRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere.pem")
	if err := os.WriteFile(target, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "server-key.pem")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := pairing.LoadIdentity(path, nil); err == nil {
		t.Error("a symlink was followed to read the server's private key")
	}
}

// The fingerprint is RFC 7469's: base64(sha256(DER SubjectPublicKeyInfo)). The
// client computes the same value from the X9.63 point with the P-256 header
// prepended, so the two sides have to agree byte for byte or pairing appears to
// succeed and every later connection fails.
func TestTheFingerprintIsOfTheSubjectPublicKeyInfo(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	fromSPKI := pairing.Fingerprint(spki)
	if len(fromSPKI) != 44 {
		t.Fatalf("the fingerprint is %d characters, want 44 of base64", len(fromSPKI))
	}
	// The same key, reached through a certificate, has to hash the same.
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := id.MintLeaf(&key.PublicKey, "Bob's iPad")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(leaf)
	if err != nil {
		t.Fatal(err)
	}
	if got := pairing.Fingerprint(parsed.RawSubjectPublicKeyInfo); got != fromSPKI {
		t.Errorf("the same key hashed to %q through a certificate and %q directly", got, fromSPKI)
	}
	if parsed.Subject.CommonName != "Bob's iPad" {
		t.Errorf("the leaf's common name is %q, want the name the client chose", parsed.Subject.CommonName)
	}
}

// The public key arrives as raw DER over the pairing call. Anything that is not
// a P-256 public key is refused before it reaches x509.
func TestOnlyAP256PublicKeyIsAccepted(t *testing.T) {
	id, err := pairing.LoadIdentity(filepath.Join(t.TempDir(), "k.pem"), nil)
	if err != nil {
		t.Fatal(err)
	}
	rsa, err := x509.MarshalPKIXPublicKey(mustRSA(t))
	if err != nil {
		t.Fatal(err)
	}
	for what, der := range map[string][]byte{
		"empty":      nil,
		"not DER":    []byte("hello"),
		"an RSA key": rsa,
	} {
		if _, err := pairing.PublicKeyFromSPKI(der); err == nil {
			t.Errorf("%s was accepted as a client's public key", what)
		}
	}
	good, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&good.PublicKey)
	pub, err := pairing.PublicKeyFromSPKI(der)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := id.MintLeaf(pub, "Bob"); err != nil {
		t.Fatal(err)
	}
}

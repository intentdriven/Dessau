package pairing_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

// mustRSA is a key of the wrong kind, made only to be refused.
func mustRSA(t *testing.T) *rsa.PublicKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &k.PublicKey
}

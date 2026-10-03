//go:build !darwin

package gateway

import "net/netip"

// lookupPeerUID has no answer off macOS, so every connection reads as not
// this account's: a narrowed panel refuses rather than guesses.
func lookupPeerUID(local, remote netip.AddrPort) (uid uint32, found bool, err error) {
	return 0, false, errPeerLookupUnsupported
}

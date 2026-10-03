//go:build !darwin || !cgo

package gateway

import "net/netip"

// peerLookupSupported says whether lookupPeerUID can answer in this build.
const peerLookupSupported = false

// lookupPeerUID has no answer off macOS, or in a macOS build without cgo,
// which cannot call libproc, so every connection reads as not this account's:
// a narrowed panel refuses rather than guesses.
func lookupPeerUID(local, remote netip.AddrPort) (uid uint32, found bool, err error) {
	return 0, false, errPeerLookupUnsupported
}

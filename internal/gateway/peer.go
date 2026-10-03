package gateway

import (
	"errors"
	"net/netip"
	"os"
)

// errPeerLookupUnsupported is what the peer lookup answers where it cannot
// ask the kernel which process holds a connection's other end: everywhere
// but macOS, which is the only place Dessau runs.
var errPeerLookupUnsupported = errors.New("this system cannot say which account a connection comes from")

// peerUIDLookup is the kernel lookup behind peerIsThisAccount, a variable so
// a test can stand in for the answers a single account cannot produce: a
// socket found but held by another account.
var peerUIDLookup = lookupPeerUID

// peerIsThisAccount reports whether the process at the other end of a TCP
// connection to this server — the connection's local and remote addresses as
// the server sees them — runs as the account Dessau serves from
// (itd-2610031004535845, adr-2610031016411351).
//
// The kernel is asked which process holds the socket whose own address pair
// is the reverse of this one, among the processes this account may inspect.
// Another account's processes cannot be inspected, so a connection from one
// is not found; a connection that is not found, and any lookup that fails, is
// "not this account". It never answers yes on anything but the kernel's word.
func peerIsThisAccount(local, remote netip.AddrPort) (bool, error) {
	uid, found, err := peerUIDLookup(local, remote)
	if err != nil || !found {
		return false, err
	}
	return uid == uint32(os.Geteuid()), nil
}

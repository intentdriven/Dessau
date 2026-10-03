package gateway

import (
	"errors"
	"net"
	"net/netip"
	goruntime "runtime"
	"testing"
	"time"
)

// loopbackPair is an accepted connection and the address pair the server
// sees for it.
func loopbackPair(t *testing.T) (local, remote netip.AddrPort) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			accepted <- c
		}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	server := <-accepted
	t.Cleanup(func() { server.Close() })
	return netip.MustParseAddrPort(server.LocalAddr().String()), netip.MustParseAddrPort(server.RemoteAddr().String())
}

// A loopback connection from this account is attributed to it, and a pair
// no socket holds — what a connection from a process this account cannot
// inspect looks like — is not (spc-2610031016319710 step 1). Off macOS the
// lookup says it cannot answer, and the answer is no.
func TestAConnectionFromThisAccountIsAttributedToIt(t *testing.T) {
	local, remote := loopbackPair(t)
	ours, err := peerIsThisAccount(local, remote)
	if goruntime.GOOS != "darwin" {
		if ours || !errors.Is(err, errPeerLookupUnsupported) {
			t.Errorf("off macOS: ours = %v, err = %v; want no, unsupported", ours, err)
		}
		return
	}
	if err != nil || !ours {
		t.Fatalf("a connection from this process: ours = %v, err = %v", ours, err)
	}
	stranger := netip.AddrPortFrom(remote.Addr(), remote.Port()^0x5a5a)
	if ours, err := peerIsThisAccount(local, stranger); ours || err != nil {
		t.Errorf("a pair no inspectable socket holds: ours = %v, err = %v; want no", ours, err)
	}
	// The cost of one lookup, which every connection to a narrowed panel
	// pays once.
	start := time.Now()
	const n = 20
	for range n {
		peerIsThisAccount(local, remote)
	}
	per := time.Since(start) / n
	t.Logf("one peer lookup takes %v", per)
	if per > time.Second {
		t.Errorf("one lookup takes %v, which a connection cannot wait for", per)
	}
}

// BenchmarkPeerLookup measures the lookup a connection pays once.
func BenchmarkPeerLookup(b *testing.B) {
	if goruntime.GOOS != "darwin" {
		b.Skip("the lookup answers only on macOS")
	}
	t := &testing.T{}
	local, remote := loopbackPair(t)
	b.ResetTimer()
	for range b.N {
		peerIsThisAccount(local, remote)
	}
}

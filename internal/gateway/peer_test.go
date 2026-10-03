package gateway

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"syscall"
	"testing"
	"time"
)

// loopbackPair is an accepted connection, the address pair the server sees
// for it, and the client's end, which a test may close.
func loopbackPair(t testing.TB) (local, remote netip.AddrPort, client net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	type accepted struct {
		c   net.Conn
		err error
	}
	got := make(chan accepted, 1)
	go func() {
		c, err := ln.Accept()
		got <- accepted{c, err}
	}()
	client, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	var a accepted
	select {
	case a = <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("the listener accepted nothing")
	}
	if a.err != nil {
		t.Fatal(a.err)
	}
	t.Cleanup(func() { a.c.Close() })
	return netip.MustParseAddrPort(a.c.LocalAddr().String()), netip.MustParseAddrPort(a.c.RemoteAddr().String()), client
}

// A Mac build answers the lookup: one without cgo would get the fallback,
// which refuses every connection to a panel narrowed to this account, the
// operator's own included. The release builds on Macs with a C toolchain, and
// this keeps one that lost it from shipping quietly (adversarial review of
// #153).
func TestAMacBuildCanAskTheKernel(t *testing.T) {
	if goruntime.GOOS == "darwin" && !peerLookupSupported {
		t.Error("this macOS build has no libproc lookup: it was built without cgo")
	}
}

// The decision on the kernel's answer, for every answer: only a socket found
// and held by this account is this account's. A socket held by another
// account — which one account's test cannot open on a Mac without root —
// is not, and neither is one not found or a lookup that failed
// (spc-2610031016319710 step 1).
func TestOnlyASocketThisAccountHoldsIsThisAccounts(t *testing.T) {
	me := uint32(os.Geteuid())
	failed := errors.New("the kernel would not say")
	for name, c := range map[string]struct {
		uid   uint32
		found bool
		err   error
		want  bool
	}{
		"held by this account":    {me, true, nil, true},
		"held by another account": {me + 1, true, nil, false},
		"held by root":            {0, true, nil, me == 0},
		"not found":               {me, false, nil, false},
		"lookup failed":           {me, true, failed, false},
	} {
		t.Run(name, func(t *testing.T) {
			// peerUIDLookup is package state: no test that swaps it may run
			// in parallel with one that reads it.
			old := peerUIDLookup
			t.Cleanup(func() { peerUIDLookup = old })
			peerUIDLookup = func(netip.AddrPort, netip.AddrPort) (uint32, bool, error) { return c.uid, c.found, c.err }
			ours, err := peerIsThisAccount(netip.MustParseAddrPort("127.0.0.1:1"), netip.MustParseAddrPort("127.0.0.1:2"))
			if ours != c.want {
				t.Errorf("ours = %v, want %v", ours, c.want)
			}
			if c.err != nil && !errors.Is(err, c.err) {
				t.Errorf("err = %v, want the lookup's own", err)
			}
		})
	}
}

// A loopback connection from this account is attributed to it, and a pair
// no socket holds — what a connection from a process this account cannot
// inspect looks like — is not. Without libproc the lookup says it cannot answer,
// and the answer is no.
func TestAConnectionFromThisAccountIsAttributedToIt(t *testing.T) {
	local, remote, client := loopbackPair(t)
	ours, err := peerIsThisAccount(local, remote)
	if !peerLookupSupported {
		if ours || !errors.Is(err, errPeerLookupUnsupported) {
			t.Errorf("without libproc: ours = %v, err = %v; want no, unsupported", ours, err)
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

	// The lookup must find the CLIENT's socket, never the server's own. Both
	// ends are this process's here, so a lookup that reversed the pair would
	// find the server's accepted socket and still answer yes — and in use that
	// would attribute every connection to the serving account, since Dessau
	// always holds the server end. With the client's end closed and the
	// server's still open, the answer must be no (adversarial review of #153).
	client.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		ours, err := peerIsThisAccount(local, remote)
		if !ours && err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("with only the server's end open: ours = %v, err = %v; want no — the lookup matched the server's own socket", ours, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A connection from another account's process is not this account's. It
// needs root to start a process as another account, so it runs only as root
// on a Mac and skips with that reason otherwise.
func TestAConnectionFromAnotherAccountIsNotAttributedToThisOne(t *testing.T) {
	if !peerLookupSupported {
		t.Skip("the lookup answers only in a macOS build with cgo")
	}
	if os.Geteuid() != 0 {
		t.Skip("starting a process as another account needs root; run this test as root on a Mac to exercise it")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	// The test binary sits in go test's own work directory, which only root
	// may enter, so the other account runs a copy from a directory it can.
	dir, err := os.MkdirTemp("/tmp", "dessau-peer-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(dir, "peer-helper")
	if err := os.WriteFile(helper, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(helper, "-test.run=^TestPeerHelperProcess$")
	cmd.Env = append(os.Environ(), "DESSAU_PEER_HELPER_ADDR="+ln.Addr().String())
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 4294967294, Gid: 4294967294}} // nobody
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Closing its standard input is what lets the helper exit, so it comes
	// before the wait, in one deferred call: two would run the other way round.
	defer func() {
		stdin.Close()
		cmd.Wait()
	}()
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	local := netip.MustParseAddrPort(conn.LocalAddr().String())
	remote := netip.MustParseAddrPort(conn.RemoteAddr().String())
	// Root may inspect every process, so the socket is found; it is held by
	// another account, and the answer is no.
	if ours, err := peerIsThisAccount(local, remote); ours || err != nil {
		t.Errorf("a connection from another account: ours = %v, err = %v; want no", ours, err)
	}
}

// TestPeerHelperProcess is not a test: it is the other account's process in
// the test above, which dials the address it is given and holds the
// connection until its standard input closes.
func TestPeerHelperProcess(t *testing.T) {
	addr := os.Getenv("DESSAU_PEER_HELPER_ADDR")
	if addr == "" {
		t.Skip("a helper for TestAConnectionFromAnotherAccountIsNotAttributedToThisOne")
	}
	c, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer c.Close()
	bufio.NewReader(os.Stdin).ReadString('\n')
	os.Exit(0)
}

// BenchmarkPeerLookup measures the lookup a connection pays once.
func BenchmarkPeerLookup(b *testing.B) {
	if !peerLookupSupported {
		b.Skip("the lookup answers only in a macOS build with cgo")
	}
	local, remote, _ := loopbackPair(b)
	b.ResetTimer()
	for range b.N {
		peerIsThisAccount(local, remote)
	}
}

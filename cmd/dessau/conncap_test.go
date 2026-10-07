package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/bind"
	"github.com/intentdriven/Dessau/internal/config"
)

// The connection caps (iss-2609190254516275). A read bound turns "one
// connection held forever" into "held for the bound, then reconnect"; nothing
// before these capped how many a peer could hold at once, and /pair asks for no
// credential.

// fakeAddr is a remote address a test can choose, so the per-address cap can
// be exercised with addresses no test machine holds.
type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }

// fakeConn records whether anything read from it and when it was closed. The
// embedded net.Conn is nil: a method the wrapper calls that is not defined
// here panics, which is the test saying the wrapper did more than it should.
type fakeConn struct {
	net.Conn
	remote net.Addr

	mu     sync.Mutex
	reads  int
	closes int
	closed chan struct{}
}

func newFakeConn(remote string) *fakeConn {
	return &fakeConn{remote: fakeAddr(remote), closed: make(chan struct{})}
}

func (c *fakeConn) Read([]byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	return 0, io.EOF
}

func (c *fakeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closes++
	if c.closes == 1 {
		close(c.closed)
	}
	return nil
}

func (c *fakeConn) RemoteAddr() net.Addr { return c.remote }
func (c *fakeConn) LocalAddr() net.Addr  { return fakeAddr("198.51.100.10:11535") }

func (c *fakeConn) counts() (reads, closes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads, c.closes
}

// fakeListener hands out the connections a test pushes into it.
type fakeListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func (l *fakeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *fakeListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *fakeListener) Addr() net.Addr { return fakeAddr("198.51.100.10:11535") }

// capHarness drives a capped fake listener: one goroutine accepts, as an
// http.Server's Serve loop does, and each dial says whether the connection was
// handed on or closed.
type capHarness struct {
	t        *testing.T
	fake     *fakeListener
	ln       *connCap
	accepted chan net.Conn
	logs     *syncBuffer
}

func newCapHarness(t *testing.T, perAddr, total int) *capHarness {
	t.Helper()
	h := &capHarness{
		t:        t,
		fake:     &fakeListener{conns: make(chan net.Conn), done: make(chan struct{})},
		accepted: make(chan net.Conn, 1),
		logs:     &syncBuffer{},
	}
	log := slog.New(slog.NewTextHandler(h.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h.ln = newConnCap(h.fake, perAddr, total, log)
	go func() {
		for {
			c, err := h.ln.Accept()
			if err != nil {
				return
			}
			h.accepted <- c
		}
	}()
	t.Cleanup(func() { h.ln.Close() })
	return h
}

// dial offers one connection from remote and reports what became of it: the
// connection the server was handed, or nil and the fake it closed.
func (h *capHarness) dial(remote string) (net.Conn, *fakeConn) {
	h.t.Helper()
	fc := newFakeConn(remote)
	h.fake.conns <- fc
	select {
	case c := <-h.accepted:
		return c, fc
	case <-fc.closed:
		return nil, fc
	case <-time.After(5 * time.Second):
		h.t.Fatalf("a connection from %s was neither handed on nor closed", remote)
		return nil, nil
	}
}

func (h *capHarness) admit(remote string) net.Conn {
	h.t.Helper()
	c, _ := h.dial(remote)
	if c == nil {
		h.t.Fatalf("a connection from %s was refused under its caps", remote)
	}
	return c
}

func (h *capHarness) refuse(remote, why string) {
	h.t.Helper()
	c, fc := h.dial(remote)
	if c != nil {
		h.t.Fatalf("a connection from %s was handed on, want it closed: %s", remote, why)
	}
	if reads, _ := fc.counts(); reads != 0 {
		h.t.Errorf("a connection from %s was read %d times before it was refused; it is closed before any read", remote, reads)
	}
}

// One address at its cap is refused; another address still connects.
func TestAnAddressAtItsCapIsRefusedWhileAnotherStillConnects(t *testing.T) {
	h := newCapHarness(t, 2, 100)
	h.admit("192.0.2.1:50001")
	h.admit("192.0.2.1:50002")
	h.refuse("192.0.2.1:50003", "192.0.2.1 already holds its two connections")
	h.admit("192.0.2.2:50001")
	if got := h.ln.Refused(); got != 1 {
		t.Errorf("the listener counted %d refusals, want 1", got)
	}
	// Counted and logged, and the log line describes the listener and never
	// the peer (docs/logging.md: no client address is ever written).
	logs := h.logs.String()
	if !strings.Contains(logs, "level=DEBUG") {
		t.Errorf("the refusal was not logged at debug: %q", logs)
	}
	if strings.Contains(logs, "192.0.2.1") || strings.Contains(logs, "50003") {
		t.Errorf("the refusal's log line names the peer: %q", logs)
	}
}

// An IPv6 peer is counted by its full address: two addresses in one /64 are
// two peers. An IPv4 peer that reaches a dual-stack socket as a mapped address
// is the same peer as when it arrives as itself.
func TestPeersAreCountedByTheirFullAddress(t *testing.T) {
	h := newCapHarness(t, 2, 100)
	h.admit("[2001:db8::1]:50001")
	h.admit("[2001:db8::1]:50002")
	h.refuse("[2001:db8::1]:50003", "2001:db8::1 already holds its two connections")
	h.admit("[2001:db8::2]:50001")

	h.admit("192.0.2.7:50001")
	h.admit("[::ffff:192.0.2.7]:50002")
	h.refuse("192.0.2.7:50003", "192.0.2.7 holds two connections, one of them as a mapped address")
}

// A listener at its total is refused, whoever is asking.
func TestAListenerAtItsTotalIsRefused(t *testing.T) {
	h := newCapHarness(t, 10, 3)
	h.admit("192.0.2.1:50001")
	h.admit("192.0.2.2:50001")
	h.admit("[2001:db8::1]:50001")
	h.refuse("192.0.2.3:50001", "the listener already holds its three connections")
}

// A closed connection frees its slot, on both caps.
func TestAClosedConnectionFreesItsSlot(t *testing.T) {
	h := newCapHarness(t, 2, 2)
	a := h.admit("192.0.2.1:50001")
	h.admit("192.0.2.1:50002")
	h.refuse("192.0.2.1:50003", "the address and the listener are both full")
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	h.admit("192.0.2.1:50004")
	h.refuse("192.0.2.1:50005", "the slot the close freed has been taken again")
}

// Closing a connection twice frees one slot, not two — net/http can close a
// connection on more than one path, and a TLS connection closes its own.
func TestClosingTwiceFreesOnce(t *testing.T) {
	h := newCapHarness(t, 2, 100)
	a := h.admit("192.0.2.1:50001")
	h.admit("192.0.2.1:50002")
	a.Close()
	a.Close()
	h.admit("192.0.2.1:50003")
	h.refuse("192.0.2.1:50004", "a double close freed two slots for one connection")
}

// This Mac is never locked out of its own panel by the per-address cap. Every
// account on this Mac arrives as the one loopback address, so a per-address
// cap there would be a cap on the whole Mac, shared by the panel and every
// local client. Held here at the production figures, with the panel's load
// at more than the per-address cap — far more than any panel opens.
func TestALoopbackClientIsNotStarvedByThePanelsParallelLoad(t *testing.T) {
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// Closed last, after the server's side: the side that closes first keeps
	// the TIME_WAIT, and kept on these clients' ephemeral ports it collides
	// with the bind tests' free ports.
	var clients []net.Conn
	t.Cleanup(func() { closeConns(clients) })
	release := make(chan struct{})
	var entered sync.WaitGroup
	entered.Add(connsPerAddr)
	srv := listenerServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			entered.Done()
			<-release
		}
		io.WriteString(w, "ok")
	}), requestReadTimeout)
	ln := capConns(raw, testLogger(t))
	go srv.Serve(ln)
	t.Cleanup(func() { close(release); srv.Close() })

	addr := raw.Addr().String()
	for i := 0; i < connsPerAddr; i++ {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, c)
		if _, err := fmt.Fprint(c, "GET /api/events HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			t.Fatal(err)
		}
	}
	waited := make(chan struct{})
	go func() { entered.Wait(); close(waited) }()
	select {
	case <-waited:
	case <-time.After(10 * time.Second):
		t.Fatalf("the panel's %d parallel streams were not all admitted from loopback", connsPerAddr)
	}

	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 10 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("a loopback client was refused while the panel held %d connections: %v", connsPerAddr, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("a loopback client got %d, want 200", resp.StatusCode)
	}
}

// The TLS listeners — the paired clients' port — are capped beneath the TLS
// layer, so a connection over the cap is closed before a handshake byte is
// read. Held through acquireTLSBind at the production total, from loopback,
// which the per-address cap leaves alone and the total does not.
func TestTheTLSListenersAreCappedBeneathTheHandshake(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()

	// Closed last, for the reason the loopback test gives.
	var clients []net.Conn
	t.Cleanup(func() { closeConns(clients) })
	cfg := config.Default()
	cfg.TLSPort = port
	id, reg := testIdentity(t)
	lns := acquireTLSBind(bind.ForHost("127.0.0.1"), cfg, id, reg, testLogger(t))
	if len(lns) != 1 {
		t.Fatalf("the TLS bind took %d listeners, want 1", len(lns))
	}
	ln := lns[0]
	t.Cleanup(func() { ln.Close() })

	var mu sync.Mutex
	var held []net.Conn
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			if _, ok := c.(*tls.Conn); !ok {
				t.Errorf("the TLS listener handed on a %T; net/http needs the *tls.Conn itself", c)
			}
			mu.Lock()
			held = append(held, c)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, c := range held {
			c.Close()
		}
	})

	addr := ln.Addr().String()
	for i := 0; i < connsPerListener; i++ {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		clients = append(clients, c)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		mu.Lock()
		n := len(held)
		mu.Unlock()
		if n == connsPerListener {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d connections were handed on under the cap", n, connsPerListener)
		}
		time.Sleep(10 * time.Millisecond)
	}

	over, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer over.Close()
	over.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = over.Read(make([]byte, 1))
	if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("a connection over the TLS listener's total of %d was held open (read: %v), want it closed",
			connsPerListener, err)
	}
}

// The wrapped connection keeps the half-close net/http uses to flush an answer
// before it closes, so capping a plain listener does not turn a graceful close
// into a reset.
func TestACappedConnectionKeepsItsHalfClose(t *testing.T) {
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln := capConns(raw, testLogger(t))
	defer ln.Close()
	go func() {
		if c, err := net.Dial("tcp", raw.Addr().String()); err == nil {
			defer c.Close()
			io.Copy(io.Discard, c)
		}
	}()
	c, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	cw, ok := c.(interface{ CloseWrite() error })
	if !ok {
		t.Fatalf("a capped TCP connection (%T) has no CloseWrite", c)
	}
	if err := cw.CloseWrite(); err != nil {
		t.Errorf("CloseWrite on a capped TCP connection: %v", err)
	}
}

// The plain listeners are capped in runServer before they serve, in the shape
// the other wiring tests here hold: capConns is a function a test can call
// with any listener it likes, which makes every test above true of a socket
// this process might never cap.
func TestThePlainListenersAreCappedBeforeTheyServe(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	wrap := strings.Index(body, "lns[i] = capConns(ln, log)")
	serve := strings.Index(body, "serveAll(srv, lns,")
	if wrap < 0 {
		t.Fatal("runServer no longer caps the plain listeners")
	}
	if serve < 0 || serve < wrap {
		t.Error("the plain listeners are served before they are capped")
	}
	tlsSrc, err := os.ReadFile("tlsbind.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tlsSrc), "tls.NewListener(capConns(ln, log), tlsCfg)") {
		t.Error("the TLS listeners are no longer capped beneath the TLS layer")
	}
}

// The figures, and the page that states them.
func TestTheConnectionCapsAreTheRecordedFigures(t *testing.T) {
	if connsPerAddr != 32 || connsPerListener != 512 {
		t.Errorf("the caps are %d per address and %d per listener, want the 32 and 512 recorded in "+
			".abcd/work/DECISIONS.md", connsPerAddr, connsPerListener)
	}
	page, err := os.ReadFile("../../docs/bind-address.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{fmt.Sprint(connsPerAddr), fmt.Sprint(connsPerListener)} {
		if !strings.Contains(string(page), want) {
			t.Errorf("docs/bind-address.md does not state the figure %s", want)
		}
	}
}

func closeConns(cs []net.Conn) {
	for _, c := range cs {
		c.Close()
	}
}

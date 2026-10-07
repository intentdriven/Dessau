package main

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/intentdriven/Dessau/internal/gateway"
)

// The bounds every listener this process opens is built with.
//
// headerReadTimeout is how long a peer may take over the request line and the
// headers, idleTimeout is how long an idle keep-alive connection is held, and
// requestReadTimeout is how long the whole request — headers and body together
// — may take to arrive. Without the last one the body was bounded in bytes and
// not in time, and a peer that sent headers and then trickled held a goroutine
// and a connection for as long as it kept the socket open
// (iss-2609190226050845). /pair is the sharp case, because it asks for no
// credential.
//
// Thirty seconds: it is gateway.BodyReadTimeout, the figure internal/gateway
// puts on the completions body per request, for the same reason, and defined
// as it so the two cannot drift apart (iss-2609190254515481). That deadline covers
// the one route that reads a large body and lifts itself before the generation
// starts; this one covers everything else — /pair, the control plane, and any
// route added later that forgets to bound itself. A bound on the listener is
// the one that cannot be forgotten.
//
// There is deliberately NO WriteTimeout. Generation legitimately takes minutes
// and a completion streams for as long as the model answers for, so a bound on
// the response side is a bound on how long a model may think. The request read
// is what requestReadTimeout covers; the response is the WriteTimeout's
// business and this server does not set one.
//
// The read bound does not reach the response either, and that is a property of
// net/http rather than an accident: the moment the body is read to the end,
// net/http starts its background read on the connection and CLEARS the read
// deadline to do it. A handler that has read its request — which the gateway
// does, in one io.ReadAll, and which a bodiless request such as the control
// panel's event stream has done before it starts — then streams for as long as
// it likes. The tests in listeners_test.go hold that, because a deadline still
// armed while a handler worked would put that background read into a timeout,
// and a background read that errors cancels the request context that the
// generation runs under.
//
// The one shape this WOULD bite is a handler that reads part of its body,
// answers for longer than the bound, and then reads the rest: the body has not
// hit EOF, so nothing has cleared the deadline, and the second read fails on a
// clock that started before the answer did. No route does that — every one of
// them takes its body in a single read up front — and a route that wanted to
// would have to lift the deadline itself, the way internal/gateway already
// lifts its own before a generation starts.
const (
	headerReadTimeout  = 15 * time.Second
	idleTimeout        = 120 * time.Second
	requestReadTimeout = gateway.BodyReadTimeout
)

// listenerServer is the http.Server both listeners are built from — the plain
// one and the TLS one — so that a bound put on one is a bound on the other.
// The read bound is a parameter only so a test can use a short one and still
// be the same server in every other respect.
func listenerServer(h http.Handler, read time.Duration) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: headerReadTimeout,
		ReadTimeout:       read,
		IdleTimeout:       idleTimeout,
	}
}

// The connection caps every listener this process opens is wrapped in
// (iss-2609190254516275): one remote address may hold at most connsPerAddr
// open connections, and one listener at most connsPerListener. The read
// bounds above turn "one connection held forever" into "held for the bound,
// then reconnect"; these are what stop a peer reconnecting in a loop from
// holding an unbounded number of sockets and goroutines at once. /pair is the
// sharp case again, because it asks for no credential.
//
// The figures are recorded, with what would show them wrong, in
// .abcd/work/DECISIONS.md (2026-10-07). In short: a model queues at most 64
// requests beyond its batch and refuses the rest itself, and the clients this
// server is built for open a handful of connections each (URLSession and a
// browser hold six per host; the Discord bridge asks in-process and holds
// none), so 32 is several times any one machine's normal parallelism while
// still taking sixteen addresses to fill a listener. 512 is eight models'
// whole queues — more than this Mac holds loaded at once — and costs a few
// tens of megabytes of goroutines and buffers at most, far inside the
// descriptor limit.
//
// Constants rather than settings: the record asks for none, and a ceiling an
// operator can lift is one more thing the panel has to explain.
const (
	connsPerAddr     = 32
	connsPerListener = 512
)

// capConns wraps a listener in the connection caps. It is the one primitive
// both servers' sockets go through: runServer wraps the plain listeners before
// they serve, and acquireTLSBind wraps each raw socket BENEATH the TLS layer,
// so a connection over a cap is closed before a handshake byte is read and
// net/http is still handed the *tls.Conn it looks for.
func capConns(ln net.Listener, log *slog.Logger) net.Listener {
	return newConnCap(ln, connsPerAddr, connsPerListener, log)
}

// connCap is a listener whose Accept closes a connection over either cap the
// moment it is accepted, before anything reads from it, and goes on to the
// next one. It never returns the refusal as an error: an error from Accept is
// net/http's signal to back off or to stop serving, and a peer at its cap is
// neither.
//
// The per-address cap is keyed by the peer's full address — an IPv6 peer is
// not collapsed to its /64. That leaves a peer that can mint addresses (an
// IPv6 host on the same link can, freely) able to sidestep the per-address
// cap, which is what the listener's total is for: it is the ceiling that holds
// whatever the addresses are. Collapsing to a /64 would instead put every
// client behind one router's prefix into a single bucket.
//
// Loopback peers are not held to the per-address cap. Every account on this
// Mac arrives as the one loopback address, so a per-address cap there is a cap
// on the whole Mac, shared between the operator's own panel and every local
// client — one busy local client would lock the operator out of the panel. The
// pool says the same of its own per-source cap: an address is not a client.
//
// Nor do they share the remote peers' total: loopback has a budget of its own,
// of the same size. Bound to ::, the wildcard listener is dual-stack and the
// panel's localhost reaches it as ::1, so a total shared with the network
// would let a machine on the LAN that fills it lock the operator out of the
// panel. Each side filling its own budget leaves the other's alone.
type connCap struct {
	net.Listener
	perAddr, total int
	log            *slog.Logger

	mu       sync.Mutex
	open     int // remote connections, held to total
	loopback int // loopback connections, held to a total of their own
	byAddr   map[string]int

	refused atomic.Uint64
	lastLog atomic.Int64 // unix nanoseconds of the last refusal logged
}

func newConnCap(ln net.Listener, perAddr, total int, log *slog.Logger) *connCap {
	return &connCap{Listener: ln, perAddr: perAddr, total: total, log: log, byAddr: map[string]int{}}
}

// Refused is how many connections this listener has closed over a cap.
func (l *connCap) Refused() uint64 { return l.refused.Load() }

func (l *connCap) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		key, exempt := peerKey(c.RemoteAddr())
		if over := l.admit(key, exempt); over != "" {
			// Counted before the close, so a refusal the peer has seen is
			// one the count already holds.
			l.logRefusal(over)
			c.Close()
			continue
		}
		return &cappedConn{Conn: c, release: func() { l.release(key, exempt) }}, nil
	}
}

// admit takes a slot for a connection from key, or names the cap it is over.
func (l *connCap) admit(key string, exempt bool) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if exempt {
		if l.loopback >= l.total {
			return "loopback"
		}
		l.loopback++
		return ""
	}
	if l.open >= l.total {
		return "listener"
	}
	if l.byAddr[key] >= l.perAddr {
		return "per-address"
	}
	l.open++
	l.byAddr[key]++
	return ""
}

func (l *connCap) release(key string, exempt bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if exempt {
		l.loopback--
		return
	}
	l.open--
	if l.byAddr[key]--; l.byAddr[key] <= 0 {
		delete(l.byAddr, key)
	}
}

// logRefusal counts a refusal and logs it at debug, at most once a second: a
// peer reconnecting in a loop causes one per accept, and a debug level left
// on must not let it write to this Mac's disk as fast as it can connect. The
// line carries the running count, so the refusals in between are not lost. It
// names this listener and never the peer — no client address is written to
// the log (docs/logging.md).
func (l *connCap) logRefusal(over string) {
	n := l.refused.Add(1)
	now := time.Now().UnixNano()
	last := l.lastLog.Load()
	if last != 0 && now-last < int64(time.Second) {
		return
	}
	if !l.lastLog.CompareAndSwap(last, now) {
		return
	}
	l.log.Debug("refused a connection over the "+over+" connection cap",
		"listener", l.Addr().String(), "per_address_cap", l.perAddr,
		"listener_cap", l.total, "refused_total", n)
}

// peerKey is the per-address cap's key for a remote address, and whether that
// address is loopback. The key is the full address without its port. An IPv4
// peer that a dual-stack socket presents as IPv4-mapped IPv6 is unmapped,
// which is the same address spelled the other way rather than a collapse. An
// address that does not parse is keyed by its whole string: counted, and
// never exempt.
func peerKey(a net.Addr) (string, bool) {
	if a == nil {
		return "", false
	}
	ap, err := netip.ParseAddrPort(a.String())
	if err != nil {
		return a.String(), false
	}
	ip := ap.Addr().Unmap()
	return ip.String(), ip.IsLoopback()
}

// cappedConn gives its slot back when it is closed, exactly once however many
// times it is closed: net/http can close a connection on more than one path,
// and a TLS connection closes the one beneath it.
type cappedConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *cappedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

// CloseWrite passes the half-close through. net/http half-closes a plain
// connection to flush an answer before closing it, and finds the method by
// asking the connection it was handed, so a wrapper without it would turn
// that graceful close into a possible reset.
func (c *cappedConn) CloseWrite() error {
	if cw, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return errors.ErrUnsupported
}

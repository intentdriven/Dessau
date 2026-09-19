package main

import (
	"net/http"
	"time"
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
// Thirty seconds, which is the figure internal/gateway already puts on the
// completions body per request, and for the same reason. That deadline covers
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
	requestReadTimeout = 30 * time.Second
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

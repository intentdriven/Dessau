package runtime

import (
	"context"
	_ "embed"
	"net"
	"net/http"
)

// serveScript is the launcher every model server is started through: the
// pinned mlx-lm's own server, run on a Unix socket rather than a TCP port
// (serve.py says how, and what it refuses). It is Dessau's own code, carried
// in the binary and handed to the interpreter with -c, never read from a file
// anything else could have written.
//
//go:embed serve.py
var serveScript string

// childBaseURL is what every request to a model server is written against.
// Its host is a label, never resolved and never dialled: the transport
// childTransport builds connects to the server's socket whatever the URL
// says. The label is under .invalid, which no resolver answers (RFC 6761),
// so a request that ever reached some other transport would fail rather than
// carry a prompt to whatever a name lookup returned.
const childBaseURL = "http://model-server.invalid"

// childTransport is the one way anything in Dessau reaches a model server:
// over the Unix socket the server listens on, in a directory only this
// account can open (iss-2610030846581757). The pool builds one per server and
// hands it out with every Upstream, so the readiness probe, the gateway, the
// self-test and the tool-call probe all reach the server through it and
// nothing else does; internal/archtest holds this file to being the only one
// that dials a Unix socket.
//
// There is no proxy: a proxy named in the environment would otherwise be
// handed every prompt. There is no response timeout either — a generation
// legitimately runs for minutes — so each caller bounds its own request with
// its context, as it always has.
func childTransport(socket string) *http.Transport {
	var d net.Dialer
	return &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return d.DialContext(ctx, "unix", socket)
		},
		MaxIdleConnsPerHost: 32,
	}
}

package runtime

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
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
	return &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialChild(ctx, socket, os.Geteuid())
		},
		MaxIdleConnsPerHost: 32,
	}
}

// dialChild connects to the model server's socket and keeps the connection
// only when the process listening there runs as owner.
//
// The socket's directory is checked when the pool makes it and before each
// launch, but a running server's path is dialled for as long as it runs, and
// a name can be taken over in that time: a temporary directory another
// account can write, a directory that went and was made again by somebody
// else. A listener there would be theirs, and the prompt sent to it handed to
// them. The kernel says who is listening (LOCAL_PEERCRED: the credentials the
// listener had when it called listen), so that is asked on every connection.
func dialChild(ctx context.Context, socket string, owner int) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return nil, err
	}
	if err := peerIs(conn, owner); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// peerIs refuses a Unix connection whose other end does not run as uid.
func peerIs(conn net.Conn, uid int) error {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return errors.New("model server connection is not a Unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return err
	}
	var peer uint32
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		peer, credErr = socketPeerUID(int(fd))
	}); err != nil {
		return err
	}
	if credErr != nil {
		return fmt.Errorf("model server's credentials: %w", credErr)
	}
	if int(peer) != uid {
		return fmt.Errorf("the model server's socket is held by another account (uid %d)", peer)
	}
	return nil
}

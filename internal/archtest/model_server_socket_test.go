package archtest_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Nothing in Dessau reaches a model server over TCP (iss-2610030846581757).
//
// A model server answers on a Unix socket in a directory only the serving
// account can open, so only that account's processes can reach it; a loopback
// port was reachable by every account on the Mac, and the server behind it
// loads whatever directory a request names. The packages that launch a model
// server or hold one's address — the runtime, the gateway, the self-test, the
// tool-call probe and the app that wires them — must therefore never name a
// loopback address with a port, never pass the server a host or port flag,
// and never ask the kernel for a free TCP port.
//
// And the socket is dialled in one place: internal/runtime/childconn.go builds
// the transport every caller is handed with an Upstream, so a second dialler
// is a second door this test is here to notice.
//
// What is not checked, so a green run is not over-read: a dial spelled some
// other way (a net.TCPAddr built by hand, an address read from a file), and
// packages outside the five named here. instance and lifecycle name
// 127.0.0.1 for Dessau's own port, which is the gateway's and not a model
// server's.
func TestNothingReachesAModelServerOverTCP(t *testing.T) {
	root := repoRootDir(t)
	pkgs := []string{"runtime", "gateway", "selftest", "toolprobe", "app"}
	forbidden := []string{`127.0.0.1:%d`, `"--port"`, `"--host"`, `Listen("tcp"`, `freePort`}
	const canonical = "internal/runtime/childconn.go"
	sawCanonical := false

	walkRepoFiles(t, filepath.Join(root, "internal"), walkOptions{}, func(path string, d fs.DirEntry) error {
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		src := readRepoFile(t, root, rel)

		// internal/mlxtest is the fake model server tests stand up on the
		// socket the pool names: it listens there, and ships in no binary.
		if strings.Contains(src, `"unix"`) && !strings.HasPrefix(rel, "internal/mlxtest/") {
			if rel == canonical {
				sawCanonical = true
			} else {
				t.Errorf("%s dials a Unix socket; the model server's socket is dialled in %s alone", rel, canonical)
			}
		}

		inScope := false
		for _, p := range pkgs {
			if strings.HasPrefix(rel, "internal/"+p+"/") {
				inScope = true
			}
		}
		if !inScope {
			return nil
		}
		for _, f := range forbidden {
			if strings.Contains(src, f) {
				t.Errorf("%s names %s — a model server is reached over its private socket, never over TCP", rel, f)
			}
		}
		return nil
	})
	if !sawCanonical {
		t.Errorf("%s does not dial the model server's socket — the one place it is meant to be dialled", canonical)
	}
}

// Every request a caller sends a model server goes through the transport the
// pool handed it with the Upstream, which is the one that dials the socket.
// A caller that sent it through a transport or client of its own would find
// nothing listening — or, were a model server ever reachable some other way,
// would reach it around the socket.
func TestEveryUpstreamRequestGoesThroughTheUpstreamsTransport(t *testing.T) {
	root := repoRootDir(t)
	for _, c := range []struct{ file, spelling string }{
		{"internal/gateway/gateway.go", "up.Transport.RoundTrip("},
		{"internal/gateway/ask.go", "up.Transport.RoundTrip("},
		{"internal/selftest/request.go", "UpstreamClient(r.opts.Client, up.Transport)"},
		{"internal/toolprobe/probe.go", "selftest.UpstreamClient(p.opts.Client, up.Transport)"},
	} {
		if !strings.Contains(readRepoFile(t, root, c.file), c.spelling) {
			t.Errorf("%s does not send its upstream request through the Upstream's transport (%s)", c.file, c.spelling)
		}
	}
}

package main

import (
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/Gropius/internal/bind"
	"github.com/intentdriven/Gropius/internal/config"
)

// The TLS listeners are the bind plan's listeners, on the TLS port. Loopback
// first, like every other bind, so a Mac that narrowed to itself still has a
// port its own chat client can pair on (adr-2609091123526871, rule 1).
func TestTheTLSBindIsThePlansAddressesOnItsOwnPort(t *testing.T) {
	cases := []struct {
		what string
		plan bind.Plan
		want []string
	}{
		{"the wildcard", bind.ForHost("0.0.0.0"), []string{"127.0.0.1:11536", "0.0.0.0:11536"}},
		{"this Mac only", bind.ForHost("127.0.0.1"), []string{"127.0.0.1:11536"}},
		{"a specific address", bind.ForHost("192.0.2.5"), []string{"127.0.0.1:11536", "192.0.2.5:11536"}},
		{"a bind that narrowed", bind.ForHost("0.0.0.0").WithoutExtra("gone"), []string{"127.0.0.1:11536"}},
	}
	for _, c := range cases {
		got := c.plan.Addrs(config.Default().EffectiveTLSPort())
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: the TLS bind is %v, want %v — the TLS listener must narrow with the plain one", c.what, got, c.want)
		}
	}
}

// A TLS port this Mac cannot take is a port this server does not answer on,
// and never a reason to refuse to start: the plain port is the singleton's
// contention point, and the port-ownership challenge contacts it alone.
func TestATLSPortAlreadyHeldDoesNotStopTheServer(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	port := held.Addr().(*net.TCPAddr).Port

	cfg := config.Default()
	cfg.TLSPort = port
	id, reg := testIdentity(t)
	lns := acquireTLSBind(bind.ForHost("127.0.0.1"), cfg, id, reg, testLogger(t))
	for _, ln := range lns {
		defer ln.Close()
	}
	if len(lns) != 0 {
		t.Errorf("the TLS bind took %d listeners on a held port, want none", len(lns))
	}
}

// No TLS port means no TLS listener, which is what a port that collides with
// the plain one is narrowed to at load.
func TestNoTLSPortMeansNoTLSListener(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"host":"0.0.0.0","port":11535,"tls_port":11535,"decode_concurrency":4}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, notices, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.EffectiveTLSPort(); got != 0 {
		t.Fatalf("the TLS port in force is %d, want 0 (%v)", got, notices.Repaired)
	}
	id, reg := testIdentity(t)
	if lns := acquireTLSBind(bind.ForHost("0.0.0.0"), loaded, id, reg, testLogger(t)); len(lns) != 0 {
		for _, ln := range lns {
			ln.Close()
		}
		t.Error("a server with no TLS port took a TLS listener")
	}
}

// The certificate is minted for what the bind acquired plus this Mac's own
// name. It decides nothing about trust — the client pins the key — but a name
// missing from it is one no client can use without a warning it must not be
// taught to ignore.
func TestTheCertificateCoversWhatTheBindAcquired(t *testing.T) {
	sans := tlsSANs(bind.ForHost("192.0.2.5"), "alices-mac")
	joined := strings.Join(sans, " ")
	for _, want := range []string{"alices-mac", "alices-mac.local", "192.0.2.5"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the certificate's names %v do not cover %q", sans, want)
		}
	}
	if got := strings.Join(tlsSANs(bind.ForHost("0.0.0.0"), "alices-mac"), " "); strings.Contains(got, "0.0.0.0") {
		t.Errorf("the wildcard reached the certificate's names: %q", got)
	}
}

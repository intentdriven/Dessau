package discovery

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/brutella/dnssd"

	"github.com/intentdriven/Dessau/internal/config"
)

// The instance name is this Mac's own Computer Name, exactly as System Settings
// shows it: no product prefix, no parentheses (adr-2609200729102059,
// decision 4).
//
// Two things make that the right name rather than a matter of taste. The
// service TYPE already says what the service is, so a prefix repeats it in
// every browser list; and the Computer Name is what Tailscale calls the same
// Mac, so one machine keeps one identity on the LAN and on the tailnet. A
// decorated name is perfectly legal in DNS-SD — an instance name is an
// arbitrary UTF-8 label — which is exactly why nothing but a test holds this.
func TestTheInstanceNameIsTheComputerNameWithNoDecoration(t *testing.T) {
	for _, name := range []string{"Alice's Mac", "Bob-MBP", "Carol’s MacBook Pro", "dessau"} {
		if got := serviceName(name); got != name {
			t.Errorf("serviceName(%q) = %q, want the Computer Name unchanged", name, got)
		}
	}
	for _, decoration := range []string{"Dessau", "dessau-", "(", ")"} {
		if got := serviceName("AlicesMac"); strings.Contains(got, decoration) {
			t.Errorf("serviceName(%q) = %q, which carries %q; the service type already says what "+
				"the service is, and the name has to match this Mac's name everywhere else",
				"AlicesMac", got, decoration)
		}
	}
	if got := serviceName(""); got != "dessau" {
		t.Errorf("serviceName(%q) = %q, want the %q fallback for a Mac that answers nothing", "", got, "dessau")
	}
}

// capturingAnnouncer records the dnssd.Config it is handed and serves nothing,
// so what Start would actually put on the wire can be read back without a
// multicast socket.
type capturingAnnouncer struct {
	cfg dnssd.Config
}

func (c *capturingAnnouncer) Register(cfg dnssd.Config) (registration, error) {
	c.cfg = cfg
	return idleRegistration{}, nil
}

type idleRegistration struct{}

func (idleRegistration) Respond(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// And the advertiser puts that name on the wire. The address records keep their
// own derived label, which must NEVER be the machine's own: claiming it makes
// macOS rename the machine, which is the regression
// TestServiceHostNeverClaimsTheMachineHostname exists for. The two names are
// different things and this holds them apart.
func TestStartAdvertisesUnderTheComputerName(t *testing.T) {
	computer := config.ComputerName()
	if computer == "" {
		t.Skip("no ComputerName on this machine")
	}

	rec := &capturingAnnouncer{}
	a := &Advertiser{Port: 11535, Log: slog.New(slog.DiscardHandler), announce: rec}
	if err := a.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer a.Stop()

	if want := serviceName(computer); rec.cfg.Name != want {
		t.Errorf("advertised instance name %q, want %q — this Mac's Computer Name", rec.cfg.Name, want)
	}
	if !strings.HasPrefix(rec.cfg.Host, "dessau-") {
		t.Errorf("advertised host label %q, want the derived dessau- label; claiming the machine's "+
			"own name makes macOS rename the machine", rec.cfg.Host)
	}
	if rec.cfg.Type != ServiceType {
		t.Errorf("advertised service type %q, want %q", rec.cfg.Type, ServiceType)
	}
}

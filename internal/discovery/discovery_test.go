package discovery

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/intentdriven/Dessau/internal/config"
)

// Regression test for a bug that renamed the user's Mac.
//
// brutella/dnssd is a standalone responder: it publishes A/AAAA records claiming
// whatever Config.Host is set to. macOS's mDNSResponder already owns
// <LocalHostName>.local. When Dessau claimed that same name, macOS detected a
// collision and renamed the machine (AlicesMac -> AlicesMac-2) — a persistent
// change to the user's system settings.
//
// The name we publish under must therefore never equal the machine's own.
func TestServiceHostNeverClaimsTheMachineHostname(t *testing.T) {
	local := config.LocalHostName()
	if local == "" {
		t.Skip("no LocalHostName on this machine")
	}

	got := serviceHost(local)

	if strings.EqualFold(got, local) {
		t.Fatalf("serviceHost(%q) = %q — publishing address records for the machine's own "+
			"hostname makes macOS rename the machine to avoid the collision", local, got)
	}
	if !strings.HasPrefix(got, "dessau-") {
		t.Errorf("serviceHost(%q) = %q, want a dessau- prefixed name that nothing else can own", local, got)
	}
}

// The advertised auth/model hints must reflect the live callbacks, so a runtime
// change (an API key set in the control panel, a model finishing download) is
// published rather than frozen at the value from Start.
func TestTxtRecordReflectsLiveCallbacks(t *testing.T) {
	keyed := false
	models := 0
	a := &Advertiser{
		AuthRequired: func() bool { return keyed },
		Models:       func() int { return models },
	}

	rec := a.txtRecord()
	if rec["auth"] != "none" || rec["models"] != "0" {
		t.Fatalf("initial record = %v, want auth=none models=0", rec)
	}

	// Simulate the user setting an API key and downloading two models.
	keyed = true
	models = 2
	rec = a.txtRecord()
	if rec["auth"] != "bearer" {
		t.Errorf("auth = %q after a key was set, want bearer", rec["auth"])
	}
	if rec["models"] != "2" {
		t.Errorf("models = %q, want 2", rec["models"])
	}
}

func TestSameText(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	if !sameText(base, map[string]string{"a": "1", "b": "2"}) {
		t.Error("identical maps reported as different")
	}
	if sameText(base, map[string]string{"a": "1", "b": "3"}) {
		t.Error("differing value reported as same")
	}
	if sameText(base, map[string]string{"a": "1"}) {
		t.Error("differing length reported as same")
	}
}

// The host label and the instance name are each a single DNS label, capped at
// 63 octets. dnssd does no length validation, and a message whose labels fail
// to pack is silently dropped by its transport — the service advertises
// nothing while Start and the responder both report success. macOS permits
// LocalHostNames up to the full 63 characters, so the published names must be
// clamped to arrive legal.
func TestServiceNamesStayWithinOneDNSLabel(t *testing.T) {
	long := strings.Repeat("x", 63)
	if got := serviceHost(long); len(got) > 63 {
		t.Errorf("serviceHost(63 chars) = %q (%d octets), over the 63-octet DNS label limit", got, len(got))
	}
	if got := serviceName(long); len(got) > 63 {
		t.Errorf("serviceName(63 chars) = %q (%d octets), over the 63-octet DNS label limit", got, len(got))
	}
	// The instance name is arbitrary UTF-8, so the cut must land between
	// runes: a label of two-byte runes is clamped to whole runes and stays
	// valid, and a cut that lands after a word leaves no trailing space.
	wide := strings.Repeat("\u00e9", 40)
	if got := serviceName(wide); !utf8.ValidString(got) || len(got) > maxDNSLabel-labelHeadroom {
		t.Errorf("serviceName(40 two-byte runes) = %q (%d octets, valid=%v); the clamp must cut between runes and stay within %d octets", got, len(got), utf8.ValidString(got), maxDNSLabel-labelHeadroom)
	}
	spaced := strings.Repeat("a", 58) + " Pro"
	if got := serviceName(spaced); strings.HasSuffix(got, " ") {
		t.Errorf("serviceName(%q) = %q, ends in a space", spaced, got)
	}
	trailing := strings.Repeat("a", 50) + "--bbbb"
	if got := serviceHost(trailing); strings.HasSuffix(got, "-") {
		t.Errorf("serviceHost(%q) = %q ends with a hyphen, which is not legal in a DNS host label", trailing, got)
	}
}

func TestServiceHostIsALegalDNSLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"AlicesMac", "dessau-alicesmac"},
		{"Alice's iMac", "dessau-alice-s-imac"},
		{"Mac-Pro-2", "dessau-mac-pro-2"},
		{"", "dessau-host"},
	}
	for _, tt := range tests {
		got := serviceHost(tt.in)
		if got != tt.want {
			t.Errorf("serviceHost(%q) = %q, want %q", tt.in, got, tt.want)
		}
		for _, r := range got {
			legal := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
			if !legal {
				t.Errorf("serviceHost(%q) = %q contains %q, which is not legal in a DNS label", tt.in, got, r)
			}
		}
	}
}

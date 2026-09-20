package config

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The endpoint Dessau hands to other machines must use the Bonjour name
// (LocalHostName), not the BSD hostname. On this machine they differ: the BSD
// hostname is "Mac" while Bonjour publishes "AlicesMac". "Mac.local" resolves to
// loopback locally — so the bug is invisible when testing on the same Mac — and
// fails to resolve from every other machine on the network.
func TestLocalHostNameMatchesBonjourNotBSDHostname(t *testing.T) {
	// The same absolute path the code under test uses: an oracle resolved
	// through PATH could answer from a different binary than the one Dessau
	// asks, and this test would then be comparing two machines' answers.
	out, err := exec.Command("/usr/sbin/scutil", "--get", "LocalHostName").Output()
	if err != nil {
		t.Skip("scutil unavailable")
	}
	want := strings.TrimSpace(string(out))
	if want == "" {
		t.Skip("no LocalHostName set on this machine")
	}

	got := LocalHostName()
	if got != want {
		t.Errorf("LocalHostName() = %q, want %q (the name Bonjour publishes)", got, want)
	}

	// Guard the specific mistake: falling back to os.Hostname() when the two
	// disagree would produce a URL that other machines cannot resolve.
	if bsd, err := os.Hostname(); err == nil {
		bsd = strings.TrimSuffix(bsd, ".local")
		if bsd != want && got == bsd {
			t.Errorf("LocalHostName() returned the BSD hostname %q instead of the Bonjour name %q; "+
				"the advertised endpoint would not resolve from other machines", bsd, want)
		}
	}
}

func TestLocalHostNameIsNotEmpty(t *testing.T) {
	if LocalHostName() == "" {
		t.Error("LocalHostName() returned empty; the Connect tab would show a broken URL")
	}
}

// The Bonjour instance name is the Computer Name, which is a DIFFERENT setting
// from the LocalHostName and is routinely spelled differently: System Settings
// shows "Alice's Mac" while the LocalHostName it derives is "Alices-Mac". The
// instance name people read has to be the former (adr-2609200729102059), so it
// is read from its own key rather than inferred from the host label.
func TestComputerNameMatchesSystemSettings(t *testing.T) {
	// The same absolute path the code under test uses, for the same reason as
	// the LocalHostName oracle above.
	out, err := exec.Command("/usr/sbin/scutil", "--get", "ComputerName").Output()
	if err != nil {
		t.Skip("scutil unavailable")
	}
	want := strings.TrimSpace(string(out))
	if want == "" {
		t.Skip("no ComputerName set on this machine")
	}

	if got := ComputerName(); got != want {
		t.Errorf("ComputerName() = %q, want %q (the name System Settings shows)", got, want)
	}
}

// A Mac that answers nothing still has to be advertised under something, so the
// chain runs Computer Name, then LocalHostName, and the advertiser supplies the
// last resort. What this holds is that the fallback is never an empty string
// while a name of some kind is available.
func TestComputerNameFallsBackToTheHostName(t *testing.T) {
	if ComputerName() == "" && LocalHostName() != "" {
		t.Error("ComputerName() returned empty although this Mac has a LocalHostName; " +
			"the advertisement would carry no instance name at all")
	}

	// The chain itself, driven through the seam: an empty answer and a failed
	// call both fall back to the host name, and a real answer is trimmed and
	// returned as is.
	real := scutilGet
	defer func() { scutilGet = real }()
	host := LocalHostName()
	for name, stub := range map[string]func(string) ([]byte, error){
		"empty":  func(string) ([]byte, error) { return []byte("\n"), nil },
		"failed": func(string) ([]byte, error) { return nil, errors.New("configd is not answering") },
	} {
		scutilGet = stub
		if got := resolveComputerName(); got != host {
			t.Errorf("%s scutil answer: resolveComputerName() = %q, want the LocalHostName %q", name, got, host)
		}
	}
	scutilGet = func(string) ([]byte, error) { return []byte("Alice's Mac\n"), nil }
	if got := resolveComputerName(); got != "Alice's Mac" {
		t.Errorf("resolveComputerName() = %q, want the Computer Name exactly as scutil spells it", got)
	}
}

package config

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// LocalHostName returns the name this Mac answers to on the network, i.e. the
// "<name>.local" that Bonjour publishes.
//
// This is NOT os.Hostname(). The two genuinely differ: os.Hostname() returns the
// DNS/BSD hostname (often something short like "Mac"), while Bonjour uses the
// *LocalHostName* from System Settings (e.g. "AlicesMac"). Advertising the wrong
// one produces a URL that quietly resolves to loopback on this machine — so it
// looks fine while testing — and fails to resolve from every other machine on
// the network, which is exactly the case that matters.
//
// The result is memoized: resolving it forks a `scutil` subprocess, and this is
// called from the state snapshot that the SSE stream re-renders on every registry
// change — up to ~10x/sec during a download, per open dashboard. That is 10+
// fork+execs a second competing with token generation on the same Mac, all for a
// value that does not change during a process's lifetime.
var (
	hostnameOnce sync.Once
	hostnameVal  string
)

func LocalHostName() string {
	hostnameOnce.Do(func() { hostnameVal = resolveLocalHostName() })
	return hostnameVal
}

// ComputerName returns the name System Settings shows for this Mac — "Alice's
// Mac", spaces, apostrophes and all.
//
// It is a DIFFERENT setting from LocalHostName, and the two are routinely
// spelled differently: macOS derives "Alices-Mac" from "Alice's Mac" for the
// DNS label and leaves the readable one alone. Anything a person reads wants
// this one; anything that has to be a DNS label wants LocalHostName. The
// Bonjour instance name is the first kind (adr-2609200729102059): a DNS-SD
// instance name is an arbitrary UTF-8 label, not a host label, and naming the
// service after the Mac the way everything else on the network does is what
// makes one machine one identity.
//
// Memoized behind its own sync.Once, and for the same reason LocalHostName is:
// it forks a subprocess, and the advertiser re-derives its configuration on
// every refresh tick.
var (
	computerOnce sync.Once
	computerVal  string
)

func ComputerName() string {
	computerOnce.Do(func() { computerVal = resolveComputerName() })
	return computerVal
}

func resolveComputerName() string {
	// scutil, by absolute path, exactly as resolveLocalHostName does and for
	// the reasons spelled out there.
	if out, err := scutilGet("ComputerName"); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	// A Mac that answers nothing here still has a host label, and a slightly
	// wrong readable name beats no advertisement at all. The caller supplies
	// the last resort when even this is empty.
	return LocalHostName()
}

func resolveLocalHostName() string {
	// scutil is the authoritative source; it is what System Settings edits and
	// what mDNSResponder publishes.
	//
	// The absolute path, not the name, as internal/capability does for sysctl.
	// This runs with no user gesture at all — memoized behind the sync.Once
	// above, reached from the endpoint list the menu bar builds and the control
	// panel's snapshot re-renders — so it runs once early in every account on
	// this Mac that launches Dessau. Dessau is built for a Mac shared by
	// several accounts, where a group-writable directory ahead of /usr/sbin on
	// this account's PATH is another account's way into this process; and even
	// with nobody hostile, a bare name is no proof of which tool answered.
	if out, err := scutilGet("LocalHostName"); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	// Fall back to the BSD hostname. Better a possibly-wrong name than none.
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(h, ".local")
}

// scutilGet asks scutil for one key, bounded in time. Both readers run once,
// memoized, but the advertiser calls them under its own lock at start, so a
// wedged configd must not hold that lock for ever: a few seconds is longer
// than scutil ever takes and short enough that Stop is not blocked behind it.
func scutilGet(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "/usr/sbin/scutil", "--get", key).Output()
}

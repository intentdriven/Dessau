package archtest_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/discovery"
)

// The chat client is a second surface onto the same server, and two of its
// values are not its own to choose: the address it opens on, and the mDNS
// service type it browses for. Both are the server's, and a client holding a
// stale copy of either fails in a way the user cannot diagnose -- a first run
// that reaches nothing, or a network search that finds nothing while the
// server is advertising a metre away. Swift is compiled by client/build.sh,
// not by `go test`, so these are read out of the client's own files here.

// serverURLDefault matches the AppStorage declaration that seeds the client's
// stored server address on first launch. The placeholder in SettingsView is a
// separate string and is deliberately not matched: it illustrates the shape of
// an address on another Mac and may name a persona's host.
var serverURLDefault = regexp.MustCompile(
	`@AppStorage\("serverURL"\)\s+var\s+serverURL\s*:\s*String\s*=\s*"([^"]*)"`)

// clientServiceType matches the client's own declaration of the service type it
// browses for, so that a comment mentioning the type cannot stand in for it. It
// is read out of the shared discovery file, which both clients compile.
var clientServiceType = regexp.MustCompile(
	`(?m)^\s*let\s+dessauServiceType\s*=\s*"([^"]*)"`)

// TestChatClientOpensOnTheServersOwnDefaultAddress holds the client's first-run
// address to the endpoint a freshly installed server listens on.
//
// The documented order installs the server and then the client on the same
// Mac, so loopback on the server's default port is the one address that is
// right before the user has told the client anything. The client no longer
// waits on a server to let the person type — the Mac's own model answers out
// of the box — but the stored address is what a launch reconnects to when a
// server answered last, and what the model picker asks for its models: a
// default that reaches nothing turns the first pick of a server into a failure
// the user cannot place.
func TestChatClientOpensOnTheServersOwnDefaultAddress(t *testing.T) {
	root := repoRootDir(t)
	source := readRepoFile(t, root, filepath.Join("client", "DessauChat", "DessauChat.swift"))

	m := serverURLDefault.FindStringSubmatch(source)
	if m == nil {
		t.Fatal(`client/DessauChat/DessauChat.swift declares no @AppStorage("serverURL") default; ` +
			"the address the client opens on is unchecked")
	}
	want := fmt.Sprintf("http://localhost:%d", config.Default().Port)
	if m[1] != want {
		t.Errorf("the chat client's first-run server address is %q; a default server listens on %q",
			m[1], want)
	}
}

// TestChatClientBrowsesForTheAdvertisedServiceType holds the client's Bonjour
// browse to the service type the server publishes.
//
// Two declarations have to agree with discovery.ServiceType, and each fails
// silently on its own: a browse for another type simply never returns a
// result, and macOS Local Network Privacy hands an undeclared service type an
// empty result set rather than an error. Either way the client reports "no
// servers found" while the server is advertising.
func TestChatClientBrowsesForTheAdvertisedServiceType(t *testing.T) {
	root := repoRootDir(t)

	t.Run("client source", func(t *testing.T) {
		source := readRepoFile(t, root, filepath.Join("client", "DessauChat", "Discovery.swift"))
		// The declaration, not a mention: a comment naming the type would
		// otherwise satisfy this while the browse asked for something else.
		m := clientServiceType.FindStringSubmatch(source)
		if m == nil {
			t.Fatal("client/DessauChat/Discovery.swift declares no dessauServiceType; " +
				"the type the client browses for is unchecked")
		}
		if m[1] != discovery.ServiceType {
			t.Errorf("the chat client browses for %q; the server advertises on %q",
				m[1], discovery.ServiceType)
		}
	})

	t.Run("bundle declarations", func(t *testing.T) {
		// Every client bundle, not the Mac's alone: the iPad's plist browses
		// the same network with the same code, and a plist that omits the type
		// browses to a silent empty list on its own system.
		plists, err := filepath.Glob(filepath.Join(root, "client", "Info*.plist"))
		if err != nil || len(plists) == 0 {
			t.Fatal("client/ holds no Info plist; no client bundle declares what it may browse for")
		}
		for _, path := range plists {
			declared := plistStringArray(t, path, "NSBonjourServices")
			// The systems accept the type with or without the trailing dot the
			// DNS-SD wire format uses; both are the same declaration.
			if !slices.Contains(declared, discovery.ServiceType) &&
				!slices.Contains(declared, discovery.ServiceType+".") {
				t.Errorf("client/%s declares NSBonjourServices %v, which does not include %q; "+
					"the system returns an empty browse rather than an error for an undeclared type",
					filepath.Base(path), declared, discovery.ServiceType)
			}
		}
	})
}

package archtest_test

import (
	"strings"
	"testing"
)

// The client's half of adr-2609182357322050, pinned at the source.
//
// The client has no test target, so what is assertable here is the SHAPE — that
// the calls the design turns on are present and the ones it forbids are
// absent — and never the behaviour. Everything about how it renders, and
// everything about a real device, is an owed hand check and is recorded as one.

// Both TLS challenges are answered, and the server trust one is answered by
// comparing keys rather than by asking the system.
//
// App Transport Security cannot LOOSEN trust, so no declarative pinning can
// accept a self-signed leaf (the SOTA note of 2026-09-19, item 6). The delegate
// is the only sanctioned route and the SPKI comparison is the gate.
func TestTheClientAnswersBothTLSChallenges(t *testing.T) {
	src := clientSources(t, repoRootDir(t))
	pairing, ok := src["Pairing.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/Pairing.swift is gone, and with it the client's half of the pairing")
	}
	for _, want := range []string{
		"NSURLAuthenticationMethodServerTrust",
		"NSURLAuthenticationMethodClientCertificate",
		"URLCredential(identity:",
		"SecTrustCopyCertificateChain",
		"URLSessionDelegate",
	} {
		if !strings.Contains(pairing, want) {
			t.Errorf("the pairing source names no %s, so that challenge is answered by the system's default handling", want)
		}
	}
}

// No declarative pinning, in either bundle. It cannot do this job, and one
// written beside the delegate would read as the thing that was doing it.
func TestTheClientDoesNotTryToPinThroughAppTransportSecurity(t *testing.T) {
	root := repoRootDir(t)
	for _, plist := range []string{"client/Info.plist", "client/Info-iPad.plist"} {
		body := readRepoFile(t, root, plist)
		// The KEY, not a mention of it: client/Info.plist's own comment names
		// NSAllowsArbitraryLoads to say it is deliberately not set, and a scan
		// that read a sentence as a declaration would refuse the sentence.
		for _, forbidden := range []string{"NSPinnedDomains", "NSPinnedCAIdentities", "NSPinnedLeafIdentities"} {
			if strings.Contains(body, "<key>"+forbidden+"</key>") {
				t.Errorf("%s declares %s; App Transport Security cannot loosen trust, so a declarative pin "+
					"cannot accept this server's certificate and would only look as though it did", plist, forbidden)
			}
		}
		if strings.Contains(body, "<key>NSAllowsArbitraryLoads</key>") {
			t.Errorf("%s allows arbitrary loads; the pairing needs no such thing and it would turn off "+
				"the transport rules for every other request this client makes", plist)
		}
	}
}

// The Secure Enclave is attempted and never assumed. Measured on 2026-09-19:
// under the ad-hoc signature client/build.sh gives this app, the Enclave key
// cannot be stored (errSecMissingEntitlement, -34018), so the fallback is a
// permanent Keychain key. A build signed with a provisioning profile takes the
// Enclave path without any other change — which is only true while both
// attempts are written.
func TestTheClientAttemptsTheEnclaveAndFallsBack(t *testing.T) {
	pairing := clientSources(t, repoRootDir(t))["Pairing.swift"]
	for _, want := range []string{
		"kSecAttrTokenIDSecureEnclave",
		"kSecAttrIsPermanent",
		"SecKeyCreateRandomKey",
	} {
		if !strings.Contains(pairing, want) {
			t.Errorf("the pairing source names no %s", want)
		}
	}
	if strings.Count(pairing, "SecKeyCreateRandomKey") < 2 {
		t.Error("there is one call to SecKeyCreateRandomKey, so there is one attempt: the Enclave is being " +
			"assumed rather than tried, and a device that refuses it has no key at all")
	}
	// A presence flag inside a TLS handshake is a prompt with nowhere to draw
	// it, on a queue that cannot wait for one.
	for _, forbidden := range []string{".userPresence", ".biometryAny", ".biometryCurrentSet"} {
		if strings.Contains(pairing, forbidden) {
			t.Errorf("the pairing key is guarded by %s, which puts a prompt inside a TLS handshake running "+
				"on URLSession's own queue", forbidden)
		}
	}
}

// What the client pins is the certificate a handshake presented, and never a
// value that arrived in a Bonjour record or a pairing answer: both are channels
// anything on the network can answer on, and a pin taken from one pins whatever
// the loudest answer said.
func TestTheClientPinsWhatTheHandshakePresented(t *testing.T) {
	src := clientSources(t, repoRootDir(t))
	app := src["GropiusChat.swift"]
	if !strings.Contains(app, "pinning.lastPresentedSPKI") {
		t.Error("pairing does not record what the handshake presented, so the pin comes from somewhere it " +
			"can be told rather than from somewhere it can be seen")
	}
	if strings.Contains(app, "pinnedSPKI = answer.fingerprint") {
		t.Error("the client pins the fingerprint the pairing answer carried — that answer came over the " +
			"plain port, where anything on the network can answer")
	}
	discovery := src["Discovery.swift"]
	if strings.Contains(discovery, "pinnedSPKI") {
		t.Error("the Bonjour browser reaches the pin; mDNS is unauthenticated multicast and is not a place " +
			"to learn a key from")
	}
}

// A paired server is reached over TLS or not at all. A fallback to the plain
// port on a TLS failure is an off switch anything on the network can reach for.
func TestAPairedServerIsNeverSpokenToInTheClear(t *testing.T) {
	app := clientSources(t, repoRootDir(t))["GropiusChat.swift"]
	body, ok := swiftFunctionBody(app, "func request(_ path: String) -> URLRequest? {")
	if !ok {
		t.Fatal("the client's request builder is gone")
	}
	if !strings.Contains(body, "httpsBase") {
		t.Error("the request builder does not use the paired server's TLS base")
	}
	if !strings.Contains(body, `r.setValue("Bearer`) {
		t.Error("the request builder no longer attaches the API key at all, which is the unpaired path")
	}
	// The paired branch must return before the bearer token is reached.
	paired := strings.Index(body, "httpsBase")
	bearer := strings.Index(body, `r.setValue("Bearer`)
	if paired > bearer {
		t.Error("the API key is attached before the paired branch is taken, so a paired client sends it")
	}
}

// URLSession.shared takes no delegate, so a paired request sent through it is
// refused at the handshake with nothing to explain it.
func TestTheClientUsesItsOwnSession(t *testing.T) {
	for name, src := range clientSources(t, repoRootDir(t)) {
		for _, line := range strings.Split(src, "\n") {
			if !strings.Contains(line, "URLSession.shared") {
				continue
			}
			if strings.Contains(strings.TrimSpace(line), "//") {
				continue // a comment saying why it is not used
			}
			t.Errorf("%s uses URLSession.shared: %q — a shared session carries no delegate, so neither "+
				"TLS challenge is ever seen", name, strings.TrimSpace(line))
		}
	}
}

// The pairing record is in the Keychain. A preferences plist is rewritable by
// anything running as the user, and a pin anything can rewrite is not a pin —
// which is the reason the API key was moved out of UserDefaults already.
func TestThePinIsNotInThePreferences(t *testing.T) {
	src := clientSources(t, repoRootDir(t))
	pairing := src["Pairing.swift"]
	if !strings.Contains(pairing, "kSecClassGenericPassword") {
		t.Error("the pairing record is not stored in the Keychain")
	}
	for _, name := range []string{"Pairing.swift", "GropiusChat.swift"} {
		for _, line := range strings.Split(src[name], "\n") {
			if strings.Contains(line, "@AppStorage") && strings.Contains(strings.ToLower(line), "pair") {
				t.Errorf("%s keeps a pairing value in the preferences: %q", name, strings.TrimSpace(line))
			}
		}
	}
}

// The lock beside a server's name is the pairing and nothing else. The picker
// already draws a lock for "this server wants an API key", which is a different
// claim about a different thing, so the two must not be the same glyph.
func TestTheLockForAPairingIsNotTheLockForAnAPIKey(t *testing.T) {
	picker := clientSources(t, repoRootDir(t))["Picker.swift"]
	if !strings.Contains(picker, "model.pairedHere") {
		t.Error("the picker does not draw anything for a paired server")
	}
	if !strings.Contains(picker, "lock.shield.fill") {
		t.Error("the paired server has no lock beside its name")
	}
	if strings.Contains(picker, `server.authRequired ? "lock.shield.fill"`) {
		t.Error("the API-key lock and the pairing lock are the same glyph, so the panel claims one thing " +
			"and means another")
	}
}

// swiftFunctionBody is the body of a Swift declaration, found by matching
// braces from the opening one. Good enough for a source scan and deliberately
// not a parser: what it is asked is whether two things appear, and in which
// order.
func swiftFunctionBody(src, signature string) (string, bool) {
	start := strings.Index(src, signature)
	if start < 0 {
		return "", false
	}
	depth := 0
	for i := start + len(signature) - 1; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1], true
			}
		}
	}
	return "", false
}

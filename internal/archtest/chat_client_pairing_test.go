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
		t.Fatal("client/DessauChat/Pairing.swift is gone, and with it the client's half of the pairing")
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
	app := src["DessauChat.swift"]
	// The reading has to be of THIS handshake. The delegate's last-presented
	// fingerprint is shared state, so a probe that never completes a handshake
	// would otherwise leave the previous server's key standing and be pinned
	// for this one (iss-2609190110244227).
	if !strings.Contains(app, "pinning.forgetLastPresented()") {
		t.Error("pairing does not clear the last handshake before the one it learns its pin from, so a " +
			"probe that never connects pins whatever the previous connection presented")
	}
	if !strings.Contains(app, "guard reached,") {
		t.Error("pairing does not require its probe to have reached the server, so a failed probe is " +
			"indistinguishable from one that succeeded")
	}
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
	app := clientSources(t, repoRootDir(t))["DessauChat.swift"]
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
	for _, name := range []string{"Pairing.swift", "DessauChat.swift"} {
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

// Learning a pin means having none for the length of one handshake. The pin in
// force has to be put back on every way out of pairing that does not set a new
// one, or a failed pairing leaves an already-paired client accepting any
// certificate until the app is relaunched (iss-2609190100212365).
func TestAFailedPairingRestoresThePinItCleared(t *testing.T) {
	app := clientSources(t, repoRootDir(t))["DessauChat.swift"]
	body, ok := swiftFunctionBody(app, "func pair(as name: String) async throws {")
	if !ok {
		t.Fatal("the client's pairing flow is gone")
	}
	if !strings.Contains(body, "pinning.pinnedSPKI = nil") {
		t.Skip("pairing no longer clears the pin, so there is nothing to restore")
	}
	cleared := strings.Count(body, "pinning.pinnedSPKI = nil")
	restored := strings.Count(body, "pinning.pinnedSPKI = previous")
	if restored < cleared {
		t.Errorf("pairing clears the pin %d time(s) and restores it %d time(s); a path out that does "+
			"neither leaves this client accepting any certificate", cleared, restored)
	}
	// And the certificate is not left in the keychain by a pairing that did
	// not finish (iss-2609190100217781).
	if strings.Count(body, "PairingStore.removeLeaf") < 2 {
		t.Error("a pairing that cannot complete the handshake leaves its certificate in the keychain")
	}
}

// Forgetting a pairing removes everything it made. The certificate was added
// with no tag of its own, so a delete keyed on the tag misses it and every
// pairing leaves one more behind (iss-2609190100217781).
func TestForgettingAPairingRemovesTheCertificateToo(t *testing.T) {
	pairing := clientSources(t, repoRootDir(t))["Pairing.swift"]
	body, ok := swiftFunctionBody(pairing, "static func forget() {")
	if !ok {
		t.Fatal("PairingStore.forget is gone")
	}
	if !strings.Contains(body, "removeLeaf") {
		t.Error("forgetting a pairing does not remove the certificate it stored")
	}
	for _, want := range []string{"kSecClassIdentity", "kSecClassKey"} {
		if !strings.Contains(body, want) {
			t.Errorf("forgetting a pairing does not remove the %s", want)
		}
	}
	remove, ok := swiftFunctionBody(pairing, "static func removeLeaf(_ der: Data) {")
	if !ok {
		t.Fatal("PairingStore.removeLeaf is gone")
	}
	if !strings.Contains(remove, "kSecValueRef") {
		t.Error("the certificate is removed by something other than its bytes; a label this app chose does " +
			"not survive the add on the macOS file keychain, where the common name replaces it")
	}
}

// The fallback out of the Secure Enclave is the ad-hoc-signing case and nothing
// else (iss-2609190200098392). The spike of 2026-09-19 measured exactly one
// condition — `errSecMissingEntitlement`, -34018, the Keychain add refused for
// want of an entitlement an ad-hoc signature cannot carry — and the decision
// line falls back on that condition. A fallback taken on ANY nil return reads
// the same in the source and is not the same thing: a policy change, a revoked
// profile or a future OSStatus would hand this device a non-Enclave key with
// nothing said, in the one place the feature sells itself as hardware-backed.
//
// So: the CFError is captured and read, the fallback names the status it is
// for, anything else is thrown to the person pairing, and which kind of key was
// made is logged rather than inferred.
func TestTheEnclaveFallbackIsTakenOnlyForTheMissingEntitlement(t *testing.T) {
	pairing := clientSources(t, repoRootDir(t))["Pairing.swift"]
	body, ok := swiftFunctionBody(pairing, "static func makeKey() throws -> SecKey {")
	if !ok {
		t.Fatal("PairingStore.makeKey does not throw, so a key it could not make has no reason attached " +
			"and the caller cannot tell a refused Enclave from a refused Keychain")
	}
	if !strings.Contains(body, "errSecMissingEntitlement") && !strings.Contains(body, "-34018") {
		t.Error("the fallback out of the Secure Enclave names no status, so it is taken on any failure at " +
			"all — wider than the decision of 2026-09-19, which falls back on errSecMissingEntitlement")
	}
	if strings.Contains(body, "SecKeyCreateRandomKey(enclave as CFDictionary, nil)") {
		t.Error("the Enclave attempt discards its CFError, so the reason it failed cannot be read and " +
			"every failure looks like the one the fallback is for")
	}
	if !strings.Contains(body, "&error") {
		t.Error("no CFError out-parameter is captured in makeKey, so nothing distinguishes the " +
			"missing-entitlement case from any other refusal")
	}
	if !strings.Contains(body, "CFErrorGetCode") && !strings.Contains(body, "localizedDescription") {
		t.Error("the captured CFError is never read, so it is captured and discarded, which is the defect " +
			"with an extra variable in it")
	}
	if !strings.Contains(body, "throw ") {
		t.Error("makeKey throws nothing, so an Enclave failure that is not the missing entitlement still " +
			"ends in a silently downgraded key")
	}
	// The access control is the second way out of the Enclave block, and it was
	// the one left standing (iss-2609190207534043). A nil return from
	// SecAccessControlCreateWithFlags is an Enclave attempt that failed for a
	// reason which is NOT the missing entitlement — the single condition the
	// spike measured — so it belongs with the other refusals: the CFError is
	// read and thrown. Stepping past it into the software key is the same
	// silent downgrade, taken on a condition nobody has ever observed.
	if strings.Contains(body, "[.privateKeyUsage], nil)") {
		t.Error("SecAccessControlCreateWithFlags discards its CFError, so a nil access control carries no " +
			"reason and the one place the refusal could be read is thrown away")
	}
	if !strings.Contains(body, "guard let access = SecAccessControlCreateWithFlags") {
		t.Error("a nil access control does not stop makeKey, so it falls through to a non-Enclave key: " +
			"the downgrade this fallback was narrowed to prevent, on a condition that is not the " +
			"missing entitlement")
	}
	if strings.Count(body, "Unmanaged<CFError>?") < 2 {
		t.Error("the access control call and the key call do not each capture a CFError, so one of the " +
			"two ways the Enclave can refuse has no reason attached")
	}
	// Which key was made is recorded where it can be read afterwards. An app
	// that cannot say which kind it holds cannot be held to saying so.
	if !strings.Contains(body, "logger.") {
		t.Error("makeKey logs nothing, so which kind of key this device holds is inferred rather than " +
			"reported")
	}
	// And the caller takes the reason rather than flattening it.
	app := clientSources(t, repoRootDir(t))["DessauChat.swift"]
	if !strings.Contains(app, "try PairingStore.makeKey()") {
		t.Error("the pairing flow does not call makeKey as a throwing call, so the reason a key could not " +
			"be made is thrown away at the one place it would be shown")
	}
}

// The limits page carries the limits adr-2609182357322050 accepted, and a
// limit missing from it is a reader who believes pairing covers something it
// does not (iss-2609190200113554).
//
// The compromised-Mac item is the one that was half-written: the page named
// the device holding the client key and not the Mac running the server, which
// keeps the server's own key, the paired set and the shared API key. Held as
// the four limits rather than as a form of words, so the page can be rewritten
// and still be held.
func TestTheLimitsPageNamesEveryLimitPairingHas(t *testing.T) {
	page := readDoc(t, "pairing-explained.md")

	if _, _, ok := strings.Cut(page, "## What it does not solve"); !ok {
		t.Fatal("docs/pairing-explained.md no longer states what pairing does not solve")
	}
	for _, limit := range []struct {
		what    string
		phrases []string
	}{
		{"first-come pairing", []string{"network can pair", "Clients"}},
		{"trust on first use", []string{"first connection", "fingerprint"}},
		{"the plain port is still there", []string{"ordinary port", "API key"}},
		{"a compromised client device", []string{"keychain", "controls the device"}},
		{"a compromised server Mac", []string{"Mac running your server", "Neither end is protected"}},
	} {
		if !containsAll(page, limit.phrases...) {
			t.Errorf("docs/pairing-explained.md does not tell the reader about %s", limit.what)
		}
	}
}

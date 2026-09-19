package config

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Client is one chat client that has paired with this server
// (adr-2609182357322050, decision 6).
//
// What is here is what survives a restart. When the server last heard from a
// client is deliberately NOT here: it changes on every request, and writing it
// would fsync the operator's settings file once per chat completion and race
// every save they make. The gateway holds it in memory and the panel reads it
// from there.
type Client struct {
	// Name is what the client called itself at pairing. It is chosen by
	// whoever paired — under first-come pairing, anyone on the network — so it
	// is bounded and its grammar is checked before it is stored: it reaches a
	// log line and an HTML panel. It identifies nothing; the fingerprint does.
	Name string `json:"name"`
	// SPKI is the RFC 7469 fingerprint of the client's public key, and is also
	// the key this client is filed under. It is a hash of a public value, so it
	// is not a secret and is not redacted — the panel shows the whole of it,
	// because comparing it against what the client shows is the only check
	// there is on a pairing nobody approved.
	SPKI string `json:"spki"`
	// PairedAt is the Unix second the pairing was recorded.
	PairedAt int64 `json:"paired_at"`
}

// MaxClients bounds the paired set, for the reason MaxModels bounds the
// per-model settings: everything saved is written to one file that is read back
// at the next start, and a config.json grown without limit is one the next
// start cannot read. Here the reason is sharper, because the endpoint that
// grows this map is unauthenticated by design: without a ceiling, anything on
// the network could pair until the file passed MaxConfigBytes, at which point
// Load returns the defaults, the server has no API key, a new one is generated
// and saved, and every OpenAI client on the LAN stops working — reached from a
// feature meant to be convenient.
const MaxClients = 64

// MaxClientNameBytes bounds a pairing name. Long enough for "Bob's iPad Pro in
// the kitchen", short enough that sixty-four of them are a few kilobytes.
const MaxClientNameBytes = 128

// fingerprintChars is the length of a base64-encoded SHA-256 hash.
const fingerprintChars = 44

// ClientIDs is the fingerprints of the paired clients, sorted, so that
// everything that lists them lists them in one order.
func (c Config) ClientIDs() []string {
	out := make([]string, 0, len(c.Clients))
	for k := range c.Clients {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// EffectiveTLSPort is the port the TLS listener takes: the one that was set, or
// the port beside the plain one. Zero means no TLS listener at all, which is
// what a tls_port that collides with the plain port is narrowed to.
func (c Config) EffectiveTLSPort() int {
	if c.TLSPort == NoTLSListener {
		return 0
	}
	if c.TLSPort == 0 {
		// The port beside the plain one, unless there is no port beside it.
		if c.Port >= 65535 {
			return 0
		}
		return c.Port + 1
	}
	return c.TLSPort
}

// NoTLSListener is the stored tls_port meaning "no TLS listener at all". It is
// negative so that it cannot be a port, and so that zero keeps meaning "the
// port beside the plain one" — which is what every configuration written
// before this field carries.
const NoTLSListener = -1

// validateClients checks the paired set the way validateModels checks the
// per-model settings: the ceiling, then each row.
func (c Config) validateClients() error {
	if len(c.Clients) > MaxClients {
		return fmt.Errorf("clients names %d paired clients, more than the %d this holds", len(c.Clients), MaxClients)
	}
	for _, id := range c.ClientIDs() {
		cl := c.Clients[id]
		if cl.SPKI != id {
			return fmt.Errorf("the paired client %q is filed under %q, which is not its own fingerprint", cl.SPKI, id)
		}
		if err := validFingerprint(id); err != nil {
			return fmt.Errorf("paired client %q: %w", id, err)
		}
		if err := validClientName(cl.Name); err != nil {
			return fmt.Errorf("paired client %q: %w", id, err)
		}
	}
	return nil
}

func validFingerprint(s string) error {
	if len(s) != fingerprintChars {
		return fmt.Errorf("a key fingerprint is %d characters, this one is %d", fingerprintChars, len(s))
	}
	// Strict, so that only the canonical spelling of a hash is a fingerprint.
	// Sixteen 44-character strings decode to the same 32 bytes under the
	// non-strict decoder, and a hand-edited file could carry any of them: they
	// would fail closed, never matching a key, but a row that can never match
	// is a row nobody can explain (iss-2609190110241408).
	raw, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil || len(raw) != 32 {
		return fmt.Errorf("%q is not a base64 SHA-256 hash", s)
	}
	return nil
}

// validClientName is the grammar a pairing name must pass. It is applied at the
// pairing endpoint, where the name arrives from the network, and again in
// Validate, where a hand-edited file arrives.
func validClientName(name string) error {
	if name == "" {
		return fmt.Errorf("a paired client has no name")
	}
	if len(name) > MaxClientNameBytes {
		return fmt.Errorf("a paired client's name is %d bytes, more than the %d this holds", len(name), MaxClientNameBytes)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("a paired client's name is only spaces")
	}
	for _, r := range name {
		// Control characters and anything unprintable: the name goes into a log
		// line an operator reads and onto a panel an operator clicks.
		if r == unicode.ReplacementChar || !unicode.IsPrint(r) {
			return fmt.Errorf("a paired client's name carries a character that is not printable")
		}
	}
	return nil
}

// ValidClientName is validClientName for the pairing endpoint, which refuses a
// name before anything is written rather than letting Validate refuse the whole
// save afterwards.
func ValidClientName(name string) error { return validClientName(name) }

// ValidFingerprint is validFingerprint for the same caller.
func ValidFingerprint(s string) error { return validFingerprint(s) }

// sanitizeClients drops a paired client the file carries that this build will
// not use, and says which.
//
// Dropped rather than refused, and the reason is the one Load's own comment
// gives with an edge on it: a refusal here returns Default(), which has no API
// key, which makes secureExposedBind generate and save a new one — so one
// malformed row written by hand, or by an older build, would silently change
// the key every client on the network sends.
func (c *Config) sanitizeClients() []string {
	if len(c.Clients) == 0 {
		return nil
	}
	var dropped []string
	kept := make(map[string]Client, len(c.Clients))
	ids := make([]string, 0, len(c.Clients))
	for k := range c.Clients {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	for _, id := range ids {
		cl := c.Clients[id]
		switch {
		case len(kept) >= MaxClients:
			dropped = append(dropped, fmt.Sprintf("the paired client %q (over the %d-client ceiling)", id, MaxClients))
		case validFingerprint(id) != nil:
			dropped = append(dropped, fmt.Sprintf("the paired client %q (not a key fingerprint)", id))
		case cl.SPKI != id:
			dropped = append(dropped, fmt.Sprintf("the paired client %q (filed under a fingerprint that is not its own)", id))
		case validClientName(cl.Name) != nil:
			dropped = append(dropped, fmt.Sprintf("the paired client %q (%s)", id, validClientName(cl.Name)))
		default:
			kept[id] = cl
			continue
		}
	}
	if len(dropped) == 0 {
		return nil
	}
	if len(kept) == 0 {
		c.Clients = nil
	} else {
		c.Clients = kept
	}
	return dropped
}

// sanitizeTLSPort narrows a TLS port this Mac cannot serve, rather than
// refusing the file over it.
//
// A tls_port equal to port cannot be listened on twice, and a value out of
// range cannot be listened on at all. Refusing either would refuse a save over
// a field the operator did not touch — the wedge this repository has built
// three times — so the listener goes and the setting says so.
func (c *Config) sanitizeTLSPort() []string {
	switch {
	case c.TLSPort == NoTLSListener:
		return nil
	case c.TLSPort == 0:
		return nil
	case c.TLSPort == c.Port:
		c.TLSPort = NoTLSListener
		return []string{fmt.Sprintf("tls_port %d is the port this server already answers on, so there is no TLS listener", c.Port)}
	case c.TLSPort < 0 || c.TLSPort > 65535:
		was := c.TLSPort
		c.TLSPort = NoTLSListener
		return []string{fmt.Sprintf("tls_port %d is not a port, so there is no TLS listener", was)}
	}
	return nil
}

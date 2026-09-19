package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pairedClient(name, spki string) Client {
	return Client{Name: name, SPKI: spki, PairedAt: 1_700_000_000}
}

// A fingerprint is 44 base64 characters of a 32-byte hash. Everything else is
// dropped at load with a notice rather than refused: a refusal makes Load
// return Default(), which carries no API key, which makes the exposed-bind rule
// mint and save a new one — every OpenAI client on the network broken over one
// bad row.
func TestAMalformedClientIsDroppedWithANoticeAndNeverRefuses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	good := strings.Repeat("A", 43) + "="
	raw := `{"host":"0.0.0.0","port":11535,"decode_concurrency":4,"api_key":"secret","clients":{` +
		`"` + good + `":{"name":"Bob's iPad","spki":"` + good + `","paired_at":1700000000},` +
		`"short":{"name":"Carol","spki":"short","paired_at":1}}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, notices, err := Load(path)
	if err != nil {
		t.Fatalf("a malformed client refused the whole file: %v", err)
	}
	if cfg.APIKey != "secret" {
		t.Fatalf("the settings fell back to defaults over one bad client; api_key is %q", cfg.APIKey)
	}
	if len(cfg.Clients) != 1 {
		t.Errorf("Load kept %d clients, want the one well-formed row", len(cfg.Clients))
	}
	if _, ok := cfg.Clients[good]; !ok {
		t.Error("the well-formed client was dropped")
	}
	if !strings.Contains(strings.Join(notices.Ignored, " "), "client") {
		t.Errorf("no notice named the dropped client: %v", notices.Ignored)
	}
}

// The name is chosen by whoever paired, which under first-come pairing is
// anyone on the network. It reaches a log line and an HTML panel.
func TestAClientNameIsBoundedAndCarriesNoControlCharacters(t *testing.T) {
	good := strings.Repeat("A", 43) + "="
	for what, name := range map[string]string{
		"a line break":        "Bob\nAdmin",
		"a carriage return":   "Bob\rAdmin",
		"a null":              "Bob\x00",
		"an escape":           "Bob\x1b[2J",
		"longer than the cap": strings.Repeat("b", MaxClientNameBytes+1),
		"empty":               "",
	} {
		c := Default()
		c.Clients = map[string]Client{good: pairedClient(name, good)}
		if err := c.Validate(); err == nil {
			t.Errorf("Validate accepted a client name with %s", what)
		}
	}
	c := Default()
	c.Clients = map[string]Client{good: pairedClient("Bob's iPad", good)}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate refused an ordinary client: %v", err)
	}
}

// An unauthenticated endpoint that grows config.json without limit is one the
// next start cannot read — and a start that cannot read it locks the server
// down to loopback and mints a new API key. The same reasoning as MaxModels.
func TestThePairedSetHasACeiling(t *testing.T) {
	c := Default()
	c.Clients = map[string]Client{}
	for i := range MaxClients + 1 {
		spki := strings.Repeat("A", 40) + string(rune('A'+i/26)) + string(rune('A'+i%26)) + "A="
		c.Clients[spki] = pairedClient("client", spki)
	}
	err := c.Validate()
	if err == nil {
		t.Fatalf("Validate accepted %d paired clients, over the %d ceiling", len(c.Clients), MaxClients)
	}
	if !strings.Contains(err.Error(), "clients") {
		t.Errorf("the refusal does not name the setting: %v", err)
	}
}

// The key of the map is the fingerprint, so a row filed under a different one
// would be found by a lookup that never checked.
func TestAClientIsFiledUnderItsOwnFingerprint(t *testing.T) {
	a := strings.Repeat("A", 43) + "="
	b := strings.Repeat("B", 43) + "="
	c := Default()
	c.Clients = map[string]Client{a: pairedClient("Bob", b)}
	if err := c.Validate(); err == nil {
		t.Error("Validate accepted a client filed under a fingerprint that is not its own")
	}
}

// Clone's promise is that a posted body cannot reach the live configuration
// before Validate has seen it. A map left shared is exactly that hazard, and
// for this map it is the list of who may connect.
func TestCloneCopiesThePairedSet(t *testing.T) {
	good := strings.Repeat("A", 43) + "="
	c := Default()
	c.Clients = map[string]Client{good: pairedClient("Bob", good)}
	clone := c.Clone()
	clone.Clients[good] = pairedClient("Carol", good)
	if c.Clients[good].Name != "Bob" {
		t.Error("writing to the clone's paired set reached the original — a posted body would reach the live authorization list")
	}
}

// tls_port equal to port cannot be listened on twice, and refusing it would
// refuse a save over a field the operator did not touch. It is narrowed at
// load, with a notice, like every other repairable value.
func TestATLSPortThatCollidesWithThePortIsNarrowedNotRefused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw, err := json.Marshal(map[string]any{
		"host": "0.0.0.0", "port": 11535, "tls_port": 11535, "decode_concurrency": 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, notices, err := Load(path)
	if err != nil {
		t.Fatalf("a colliding tls_port refused the whole file: %v", err)
	}
	if got := cfg.EffectiveTLSPort(); got != 0 {
		t.Errorf("the TLS port in force is %d, want 0 — no TLS listener", got)
	}
	if !strings.Contains(strings.Join(notices.Repaired, " "), "tls_port") {
		t.Errorf("no notice named tls_port: %v", notices.Repaired)
	}
}

// The default is the port beside the plain one, so a fresh install has a TLS
// listener without the operator choosing a number.
func TestTheDefaultTLSPortIsBesideThePlainOne(t *testing.T) {
	if got := Default().EffectiveTLSPort(); got != Default().Port+1 {
		t.Errorf("the default TLS port is %d, want %d", got, Default().Port+1)
	}
	c := Default()
	c.TLSPort = 9999
	if got := c.EffectiveTLSPort(); got != 9999 {
		t.Errorf("a chosen TLS port is %d, want 9999", got)
	}
}

// Only the canonical spelling of a hash is a fingerprint. Sixteen distinct
// 44-character strings decode to the same 32 bytes under the non-strict
// decoder; a row filed under one of them can never match a key and is a row
// nobody can explain (iss-2609190110241408).
func TestOnlyACanonicalFingerprintIsAFingerprint(t *testing.T) {
	canonical := strings.Repeat("A", 43) + "="
	if err := ValidFingerprint(canonical); err != nil {
		t.Fatalf("the canonical spelling was refused: %v", err)
	}
	// "B=" and "A=" differ only in the bits the padding discards.
	for _, spelling := range []string{
		strings.Repeat("A", 43) + "=", // canonical
		strings.Repeat("A", 42) + "B=",
		strings.Repeat("A", 42) + "C=",
	} {
		err := ValidFingerprint(spelling)
		if spelling == canonical {
			continue
		}
		if err == nil {
			t.Errorf("%q was accepted as a fingerprint; it decodes to the same bytes as another spelling "+
				"and could never match a key", spelling)
		}
	}
}

// A server on the last port has no port beside it, so there is no TLS listener
// rather than one on a number that is not a port (iss-2609190110241408).
func TestThereIsNoPortAfterTheLastOne(t *testing.T) {
	c := Default()
	c.Port = 65535
	if got := c.EffectiveTLSPort(); got != 0 {
		t.Errorf("a server on port 65535 offers paired clients port %d, which is not a port", got)
	}
}

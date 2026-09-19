package discovery

import (
	"strconv"
	"strings"
	"testing"
)

// A client looking for a server it can pair with has to be told two things
// that are not in the advertisement today: that there is a TLS port at all,
// and which key the server holds.
//
// The fingerprint is for DISPLAY and for noticing a key that changed. It is not
// what a client pins: mDNS is unauthenticated multicast, so anything on the
// link can answer with a record of its own, and a pin taken from one pins
// whatever the loudest answer said.
func TestTheAdvertisementCarriesTheFingerprintAndTheTLSPort(t *testing.T) {
	a := &Advertiser{
		Port:        11535,
		Models:      func() int { return 2 },
		Fingerprint: func() string { return strings.Repeat("A", 43) + "=" },
		TLSPort:     func() int { return 11536 },
	}
	txt := a.txtRecord()
	if got := txt["spki"]; got != strings.Repeat("A", 43)+"=" {
		t.Errorf("spki = %q, want the server's fingerprint", got)
	}
	if got := txt["tlsport"]; got != "11536" {
		t.Errorf("tlsport = %q, want 11536", got)
	}
	// RFC 6763 targets 200 bytes for a TXT record; this has to stay well under.
	size := 0
	for k, v := range txt {
		size += len(k) + len(v) + 2
	}
	if size > 200 {
		t.Errorf("the TXT record is %d bytes, over the %d RFC 6763 targets", size, 200)
	}
}

// A server with no TLS listener advertises no fingerprint and no port, rather
// than a zero a client would try to connect to.
func TestAServerWithNoTLSListenerAdvertisesNeither(t *testing.T) {
	a := &Advertiser{Port: 11535, Fingerprint: func() string { return "" }, TLSPort: func() int { return 0 }}
	txt := a.txtRecord()
	for _, key := range []string{"spki", "tlsport"} {
		if v, ok := txt[key]; ok {
			t.Errorf("a server with no TLS listener advertised %s=%q", key, v)
		}
	}
}

// The callbacks are live, like AuthRequired, because the fingerprint can change
// while the server runs — the operator deletes the key file and restarts into a
// new one — and a stale advertisement is one a client compares against and
// refuses.
func TestTheFingerprintIsReadLive(t *testing.T) {
	fingerprint := strings.Repeat("A", 43) + "="
	a := &Advertiser{Port: 11535, Fingerprint: func() string { return fingerprint }, TLSPort: func() int { return 11536 }}
	if a.txtRecord()["spki"] != fingerprint {
		t.Fatal("the first reading is wrong")
	}
	fingerprint = strings.Repeat("B", 43) + "="
	if got := a.txtRecord()["spki"]; got != fingerprint {
		t.Errorf("spki = %q after the key changed, want %q — the advertisement is a snapshot", got, fingerprint)
	}
}

// Nothing else in the record moved.
func TestTheExistingAdvertisementIsUnchanged(t *testing.T) {
	a := &Advertiser{Port: 11535, Models: func() int { return 3 }, AuthRequired: func() bool { return true }}
	txt := a.txtRecord()
	want := map[string]string{
		"txtvers": "1", "api": "openai", "path": "/v1", "auth": "bearer", "models": strconv.Itoa(3),
	}
	for k, v := range want {
		if txt[k] != v {
			t.Errorf("%s = %q, want %q", k, txt[k], v)
		}
	}
	if len(txt) != len(want) {
		t.Errorf("the record carries %d keys, want %d: %v", len(txt), len(want), txt)
	}
}

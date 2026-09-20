package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
)

// adr-2609182357322050's obligation: the plain port's behaviour is BYTE FOR
// BYTE unchanged for an unpaired client, whether or not anyone has paired.
//
// The fixtures are the requests the gateway already answers — the ones every
// OpenAI client on the network makes — replayed against a server with no client
// paired and against the same server with two paired, and compared on status,
// on every header the gateway sets, and on the body.
func TestThePlainPortDoesNotMoveWhenAClientPairs(t *testing.T) {
	spki := strings.Repeat("A", 43) + "="
	other := strings.Repeat("B", 43) + "="
	withNone := config.Default()
	withNone.APIKey = "secret"
	withSome := withNone.Clone()
	withSome.Clients = map[string]config.Client{
		spki:  {Name: "Bob's iPad", SPKI: spki, PairedAt: time.Now().Unix()},
		other: {Name: "Bob's Mac", SPKI: other, PairedAt: time.Now().Unix()},
	}

	seen := map[int]bool{}
	for _, f := range gatewayRequests {
		off := replayAgainst(t, withNone, f)
		on := replayAgainst(t, withSome, f)
		seen[off.code] = true
		if off.code != on.code {
			t.Errorf("%s: status %d with nobody paired, %d with two paired", f.name, off.code, on.code)
		}
		if off.headers != on.headers {
			t.Errorf("%s: headers moved\n nobody paired: %s\n two paired:    %s", f.name, off.headers, on.headers)
		}
		if !bytes.Equal(off.body, on.body) {
			t.Errorf("%s: the body moved\n nobody paired: %q\n two paired:    %q", f.name, off.body, on.body)
		}
	}

	// A replay that only ever saw one status would pass whatever the gateway
	// did to the others, so the fixtures are held to reaching an answered
	// request and a refused one before the comparison above counts for
	// anything.
	if !seen[http.StatusOK] {
		t.Error("no fixture was answered; the replay holds nothing about the path a real client takes")
	}
	refused := false
	for code := range seen {
		if code >= 400 {
			refused = true
		}
	}
	if !refused {
		t.Error("no fixture was refused; the replay holds nothing about what the API key still keeps out")
	}
}

type replayed struct {
	code    int
	headers string
	body    []byte
}

// replayAgainst sends one fixture at a gateway built on cfg and records
// everything a client would see.
//
// Only Date is dropped, and only because it is a clock reading rather than a
// behaviour. Content-Length is NOT dropped: it is the length of the answer, so
// dropping it would let the body change size under a comparison that claimed to
// hold it (iss-2609190200112161).
func replayAgainst(t *testing.T, cfg config.Config, f gatewayRequest) replayed {
	t.Helper()
	srv, _, _ := newTestGateway(t, cfg)
	var reader io.Reader
	if f.body != "" {
		reader = strings.NewReader(f.body)
	}
	req, err := http.NewRequest(f.method, srv.URL+f.path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if f.key != "" {
		req.Header.Set("Authorization", "Bearer "+f.key)
	}
	if f.host != "" {
		req.Host = f.host
	}
	if f.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for k := range resp.Header {
		if k == "Date" {
			continue
		}
		names = append(names, k+": "+strings.Join(resp.Header.Values(k), ","))
	}
	sort.Strings(names)
	return replayed{code: resp.StatusCode, headers: strings.Join(names, "; "), body: raw}
}

// A settings save is not a way to write the list of who may connect. It could
// be: the control plane is loopback-only and asks for no credential, so any
// other account on this Mac would otherwise be able to pair itself by posting
// settings, with any pairing time it chose, and to undo Alice's revocations.
func TestASettingsSaveCannotWriteThePairedSet(t *testing.T) {
	spki := strings.Repeat("A", 43) + "="
	intruder := strings.Repeat("C", 43) + "="
	stored := config.Default()
	stored.Clients = map[string]config.Client{
		spki: {Name: "Bob's iPad", SPKI: spki, PairedAt: 1},
	}

	after := applyOver(t, stored, `{"clients":{"`+intruder+`":{"name":"me","spki":"`+intruder+`","paired_at":9}}}`)
	if _, ok := after.Clients[intruder]; ok {
		t.Error("a settings save paired a client")
	}
	if _, ok := after.Clients[spki]; !ok {
		t.Error("a settings save revoked a client")
	}

	// And a save that says nothing about clients leaves them exactly as they
	// were, which is the rule every other setting the form does not own keeps.
	after = applyOver(t, stored, `{"log_level":"detailed"}`)
	if len(after.Clients) != 1 || after.Clients[spki].Name != "Bob's iPad" {
		t.Errorf("an unrelated save changed the paired set: %+v", after.Clients)
	}
}

// The panel is never served the paired set through the settings form, because a
// collection cannot round-trip through the secret placeholder a string uses:
// the form would post redacted fingerprints back and destroy every pairing.
func TestTheSettingsFormIsNotServedThePairedSet(t *testing.T) {
	spki := strings.Repeat("A", 43) + "="
	cfg := config.Default()
	cfg.Clients = map[string]config.Client{spki: {Name: "Bob", SPKI: spki, PairedAt: 1}}
	if got := redactConfig(cfg); got.Clients != nil {
		t.Errorf("the settings form is served %d paired clients; it must be served none", len(got.Clients))
	}
	if len(cfg.Clients) != 1 {
		t.Error("redactConfig emptied the caller's own paired set")
	}
}

// The fingerprint shown on the panel is the whole of it, because comparing it
// with what the client shows is the only check there is on a pairing nobody
// approved. It is a hash of a public key; there is nothing in it to keep.
func TestTheFingerprintIsShownInFull(t *testing.T) {
	spki := strings.Repeat("A", 43) + "="
	cfg := config.Default()
	cfg.Clients = map[string]config.Client{spki: {Name: "Bob", SPKI: spki, PairedAt: 1}}
	rows := pairedClients(cfg, nil)
	if len(rows) != 1 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows[0].SPKI != spki {
		t.Errorf("the pane is given %q, want the whole fingerprint %q", rows[0].SPKI, spki)
	}
	if rows[0].LastSeen != 0 {
		t.Error("a server that has heard from nobody reported a last sighting")
	}
}

// applyOver posts a settings body at a real control plane and returns the
// settings that ended up in force.
func applyOver(t *testing.T, stored config.Config, posted string) config.Config {
	t.Helper()
	srv, a := newTestControlApp(t, stored)
	resp := postJSON(t, srv, "/api/settings", posted)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the save was refused: %d", resp.StatusCode)
	}
	return a.Config()
}

// The TLS listeners are acquired once, at launch, from the configuration as it
// was then. A save that moves the port paired clients use therefore changes
// nothing until the next start, and has to say so — the silence
// iss-2609091751184914 recorded for advertise, in a second field
// (iss-2609190103062546).
func TestASaveThatChangesTheTLSPortAsksForARestart(t *testing.T) {
	stored := config.Default()
	stored.TLSPort = 11540

	srv, _ := newTestControlApp(t, stored)
	body := strings.Replace(uneditedFormBody(t, stored), `"tls_port":11540`, `"tls_port":11541`, 1)
	if !strings.Contains(body, `"tls_port":11541`) {
		t.Fatalf("the form body does not carry tls_port, so this test proves nothing: %s", body)
	}
	resp := postJSON(t, srv, "/api/settings", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var answer struct {
		Restart bool `json:"restart"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		t.Fatal(err)
	}
	if !answer.Restart {
		t.Error("a save that moved the port paired clients use reported restart=false; the listener is " +
			"still on the old port and nothing said so")
	}
}

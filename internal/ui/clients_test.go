package ui

import (
	"strings"
	"testing"
)

// The Clients pane is a pane of its own, beside Connect and Posture. Every row
// carries the four things adr-2609182357322050 names and a way to revoke.
func TestTheClientsPaneIsAPaneOfItsOwn(t *testing.T) {
	markup := readPanelMarkup(t)
	for _, want := range []string{
		`data-tab="clients"`,
		`id="tab-clients"`,
		`id="clients"`,
		`id="serverFingerprint"`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("the panel's markup carries no %s", want)
		}
	}
}

// THE FINDING THIS TEST EXISTS FOR. The name is chosen by whoever paired, and
// under first-come pairing that is anything on the network. The panel it is
// drawn on is loopback-only, asks for no credential, and has the whole settings
// API behind it — so a name that reached markup would be a script running
// same-origin on the whole settings API.
//
// The rule this holds the pane to is stronger than escaping: every field of a
// row is set with textContent, so there is no markup for a name to be part of.
// A row built with innerHTML would pass an escaping test the day somebody
// forgot one call, which is the hole internal/ui/endpoints_test.go's own
// comment records.
func TestNoClientFieldIsEverInterpolatedIntoMarkup(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderClients")
	for _, field := range []string{"c.name", "c.spki", "c.paired_at", "c.last_seen"} {
		if !strings.Contains(body, field) {
			t.Errorf("the Clients pane does not draw %s at all", field)
		}
	}
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, "innerHTML") {
			continue
		}
		// The one legitimate use is emptying the box before redrawing it.
		if strings.Contains(line, "innerHTML = ''") {
			continue
		}
		t.Errorf("the Clients pane builds markup: %q — every field is set with textContent, "+
			"because the name is chosen by whoever paired", strings.TrimSpace(line))
	}
	if !strings.Contains(body, "strong.textContent = c.name") {
		t.Error("the client's name is not set as text")
	}
}

// The fingerprint is drawn in full, because comparing it against what the
// client shows is the only check there is on a pairing nobody approved.
func TestTheWholeFingerprintIsDrawn(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderClients")
	if !strings.Contains(body, "${c.spki}") {
		t.Error("the pane does not draw the whole fingerprint beside the name")
	}
	if strings.Contains(body, "c.spki.slice") || strings.Contains(body, "c.fingerprint") {
		t.Error("the pane draws a shortened fingerprint; Alice needs the whole of it to compare")
	}
}

// A server that has just started has heard from nobody, and says so rather
// than showing a time nobody recorded.
func TestAClientNotHeardFromSaysSo(t *testing.T) {
	if got := evalPanel(t, `clientSeen(0)`, "clientSeen"); !strings.Contains(strings.ToLower(got), "not since") {
		t.Errorf("clientSeen(0) = %q, want a sentence saying nothing has been heard", got)
	}
	if got := evalPanel(t, `clientSeen(1700000000)`, "clientSeen"); strings.Contains(strings.ToLower(got), "not since") {
		t.Errorf("clientSeen of a real sighting = %q", got)
	}
}

// Revoking is its own route and not a settings save: a settings body that
// could write the paired set would make the credential-free loopback endpoint
// a second pairing endpoint.
func TestRevokingPostsItsOwnRouteKeyedOnTheFingerprint(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderClients")
	if !strings.Contains(body, "/api/clients/revoke") {
		t.Error("the Revoke button does not post to /api/clients/revoke")
	}
	if !strings.Contains(body, "fingerprint") {
		t.Error("the Revoke button does not name the fingerprint it revokes — a name is not what identifies a client")
	}
	if strings.Contains(body, "/api/settings") {
		t.Error("revoking goes through the settings save; it must be a route of its own")
	}
}

package ui

import (
	"encoding/json"
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

// The promise the source scan above stands in for: a client name that is
// markup is DRAWN AS TEXT.
//
// The scan holds the rule that produces that outcome — every field set with
// textContent — and this holds the outcome itself, because a row built some
// other way tomorrow would pass the scan the day somebody wrote it with a
// different call. The name is chosen by whoever paired, and under first-come
// pairing that is anything on the network; the panel it lands on is
// loopback-only, asks for no credential, and has the whole settings API behind
// it.
func TestAClientNameThatIsMarkupIsDrawnAsText(t *testing.T) {
	const hostile = `<img src=x onerror=1>`
	pane := renderClientsPane(t, `[{"name": `+quoteJS(hostile)+`,
		"spki": "AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIIIJJJJKKK=",
		"paired_at": 1700000000, "last_seen": 0}]`)

	if strings.Contains(strings.ToLower(pane.Markup), "<img") {
		t.Errorf("a client called %q reached the panel as markup: %s", hostile, pane.Markup)
	}
	if !strings.Contains(pane.Markup, "&lt;img src=x onerror=1&gt;") {
		t.Errorf("the name is not drawn as escaped text: %s", pane.Markup)
	}
	if !strings.Contains(pane.Text, hostile) {
		t.Errorf("the name is not drawn at all; Alice cannot see what paired with her server: %s", pane.Text)
	}
	// And the fingerprint is beside it, because a name is not what tells two
	// clients apart and an impostor's first move is to choose somebody's.
	if !strings.Contains(pane.Text, "AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIIIJJJJKKK=") {
		t.Errorf("the row does not carry the fingerprint: %s", pane.Text)
	}
}

// renderedPane is what one render of the Clients pane produced: the markup a
// browser would have built, and the text a reader would have seen.
type renderedPane struct {
	Markup string `json:"markup"`
	Text   string `json:"text"`
}

// renderClientsPane runs renderClients against a document that records what was
// set as TEXT and what was set as MARKUP, and serialises the two apart.
//
// A node whose textContent was assigned serialises escaped, exactly as a
// browser would render it; a node whose innerHTML was assigned serialises
// verbatim. That is the whole of the difference this test is about, so a pane
// rewritten to build its rows with innerHTML fails here without anyone having
// to remember to update a source scan.
func renderClientsPane(t *testing.T, clientsJSON string) renderedPane {
	t.Helper()
	src := readPanelSource(t)
	var b strings.Builder
	b.WriteString(`
const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;')
  .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
function node(tag) {
  const n = {
    tag, kids: [], _text: null, _raw: null, className: '',
    appendChild(c) { n.kids.push(c); return c; },
    append(...cs) { for (const c of cs) n.kids.push(c); },
    addEventListener() {},
    get textContent() { return n._text; },
    set textContent(v) { n._text = v; n._raw = null; n.kids = []; },
    get innerHTML() { return n._raw; },
    set innerHTML(v) { n._raw = v === '' ? null : v; n._text = null; n.kids = []; },
  };
  return n;
}
function markupOf(n) {
  if (n._raw !== null) return n._raw;
  if (n._text !== null) return esc(n._text);
  return n.kids.map((k) => markupOf(k)).join('');
}
function textOf(n) {
  if (n._raw !== null) return n._raw;
  if (n._text !== null) return n._text;
  return n.kids.map((k) => textOf(k)).join(' ');
}
const doc = {};
function $(id) { if (!doc[id]) doc[id] = node('div'); return doc[id]; }
const document = { createElement: node };
function confirmBtn(label) { const b = node('button'); b.textContent = label; return b; }
function api() { return { catch() {} }; }
function alertErr() {}
`)
	b.WriteString("const state = { server_fingerprint: 'abc', clients: " + clientsJSON + " };\n")
	for _, name := range []string{"clientPaired", "clientSeen", "renderClients"} {
		b.WriteString(extractFunction(t, src, name))
		b.WriteString("\n")
	}
	b.WriteString(`renderClients();
process.stdout.write(JSON.stringify({markup: markupOf(doc['clients']), text: textOf(doc['clients'])}));`)

	var pane renderedPane
	out := evalJS(t, b.String())
	if err := json.Unmarshal([]byte(out), &pane); err != nil {
		t.Fatalf("the panel returned %q, which is not a rendered pane: %v", out, err)
	}
	return pane
}

// quoteJS is a JSON string literal, so a hostile name reaches the snippet as
// the name and not as source.
func quoteJS(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

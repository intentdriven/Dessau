package ui

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/registry"
)

// The card says what the last check found, in words: a newer version, one
// Dessau will not run, a version never recorded, and an update under way; and
// Update appears exactly where there is something Dessau would run to update
// to (itd-2610030857275099 criteria 4, 7, 8). A decision model is never
// marked: the 'awaiting review' mark criterion 9 gave one is withdrawn
// (iss-2610042101436891), so a record still carrying it shows nothing and
// offers no Update.
func TestTheCardSaysWhatACheckFound(t *testing.T) {
	const c = `"commit":"1111111111111111111111111111111111111111"`
	cases := []struct {
		model, text, mark string
		offered           bool
	}{
		{`{"state":"ready",` + c + `}`, ``, ``, false},
		{`{"state":"ready",` + c + `,"update":{"status":"current"}}`, ``, ``, false},
		{`{"state":"ready",` + c + `,"update":{"status":"available","commit":"2222222222222222222222222222222222222222"}}`,
			`A newer version is available (222222222222).`, `newer version`, true},
		{`{"state":"ready",` + c + `,"update":{"status":"runs_own_code","commit":"2222222222222222222222222222222222222222"}}`,
			`ships its own code, which Dessau will not run`, `newer version not run`, false},
		{`{"state":"ready",` + c + `,"update":{"status":"awaiting_review","commit":"2222222222222222222222222222222222222222"}}`,
			``, ``, false},
		{`{"state":"ready",` + c + `,"update":{"status":"cannot_check","commit":"2222222222222222222222222222222222222222"}}`,
			`could not tell whether it ships its own code`, `newer version not checked`, false},
		{`{"state":"ready"}`, `Version unknown`, ``, true},
		{`{"state":"ready",` + c + `,"updating":42.7,"update":{"status":"available","commit":"2222222222222222222222222222222222222222"}}`,
			`Updating to the newer version: 42%`, ``, false},
		{`{"state":"downloading"}`, ``, ``, false},
	}
	for _, tc := range cases {
		v := evalPanelValue(t, "({text: updateText("+tc.model+"), mark: updateMark("+tc.model+"), offered: updateOffered("+tc.model+")})",
			"updateText", "updateMark", "updateOffered")
		text, _ := v["text"].(string)
		if (tc.text == "") != (text == "") || !strings.Contains(text, tc.text) {
			t.Errorf("updateText(%s) = %q, want it to say %q", tc.model, text, tc.text)
		}
		if mark, _ := v["mark"].(string); mark != tc.mark {
			t.Errorf("updateMark(%s) = %q, want %q", tc.model, mark, tc.mark)
		}
		if offered, _ := v["offered"].(bool); offered != tc.offered {
			t.Errorf("updateOffered(%s) = %v, want %v", tc.model, offered, tc.offered)
		}
	}
}

// Every name the card shows that came from HuggingFace — the repository, and
// the version line with its commit — goes into the markup through escapeHtml,
// so a name carrying markup appears as text (criterion 11). The card is built
// with innerHTML; this holds the lines it adds to the same rule as the rest.
func TestTheVersionLinesAreEscaped(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderModels")
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`escapeHtml\(m\.repo_id\)`),
		regexp.MustCompile(`escapeHtml\(update\)`),
		regexp.MustCompile(`escapeHtml\(updateFiles\)`),
		regexp.MustCompile(`escapeHtml\(mark\)`),
		regexp.MustCompile(`postModel\('/api/models/update', m\.repo_id\)`),
		regexp.MustCompile(`m\.updating != null\) \{\s*actions\.append\(btn\('Cancel update', 'ghost', \(\) =>\s*postModel\('/api/models/cancel', m\.repo_id\)`),
	} {
		if !want.MatchString(body) {
			t.Errorf("renderModels no longer matches %s", want)
		}
	}
	// And the one figure set into a style is a number, never the string the
	// snapshot carried.
	if !strings.Contains(body, "Number(m.updating)") {
		t.Error("the update's progress bar is drawn from the snapshot's value unconverted")
	}
}

// A failed update is said on the card, in the class's own words, beside what
// the last check found; an update under way says only that
// (iss-2610031320392078).
func TestTheCardSaysWhyTheLastUpdateFailed(t *testing.T) {
	const c = `"commit":"1111111111111111111111111111111111111111"`
	const avail = `"update":{"status":"available","commit":"2222222222222222222222222222222222222222"}`
	cases := []struct{ model, want, alsoWant string }{
		{`{"state":"ready",` + c + `,"update_failed":"download"}`, `The last update failed: a file did not download or did not match`, ``},
		{`{"state":"ready",` + c + `,"update_failed":"no_space"}`, `no room for the new version`, ``},
		{`{"state":"ready",` + c + `,"update_failed":"refused"}`, `did not pass the checks`, ``},
		{`{"state":"ready",` + c + `,"update_failed":"busy"}`, `still answering requests`, ``},
		{`{"state":"ready",` + c + `,"update_failed":"not_offered"}`, `not one Dessau runs`, ``},
		{`{"state":"ready",` + c + `,"update_failed":"load"}`, `did not load, so Dessau put this version back`, ``},
		{`{"state":"ready",` + c + `,` + avail + `,"update_failed":"download"}`, `The last update failed`, `A newer version is available (222222222222).`},
		{`{"state":"ready","update_failed":"download"}`, `The last update failed`, `Version unknown`},
	}
	for _, tc := range cases {
		v := evalPanelValue(t, "({text: updateText("+tc.model+")})", "updateText")
		text, _ := v["text"].(string)
		if !strings.Contains(text, tc.want) || !strings.Contains(text, tc.alsoWant) {
			t.Errorf("updateText(%s) = %q, want it to say %q and %q", tc.model, text, tc.want, tc.alsoWant)
		}
	}
	under := `{"state":"ready",` + c + `,"updating":3,"update_failed":"download"}`
	v := evalPanelValue(t, "({text: updateText("+under+")})", "updateText")
	if text, _ := v["text"].(string); strings.Contains(text, "failed") {
		t.Errorf("an update under way still shows the last failure: %q", text)
	}
	for _, planted := range []string{`"<b>x</b>"`, `"constructor"`, `"toString"`} {
		m := `{"state":"ready",` + c + `,"update_failed":` + planted + `}`
		v = evalPanelValue(t, "({text: updateText("+m+")})", "updateText")
		if text, _ := v["text"].(string); text != "" {
			t.Errorf("a class the panel does not know, %s, is shown: %q", planted, text)
		}
	}
}

// renderModelCard runs the panel's own renderModels over one model and
// returns the card's markup as a browser would parse it. escapeHtml is the
// panel's own, run against a node that escapes on the way out of innerHTML as
// a browser does once textContent was assigned; everything renderModels draws
// that is not about the model's version is stubbed to nothing, so a renderer
// that gains a helper fails here by name rather than passing unread.
func renderModelCard(t *testing.T, modelJSON string) string {
	t.Helper()
	src := readPanelSource(t)
	var b strings.Builder
	b.WriteString(`
const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;')
  .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
function node(tag) {
  const n = {
    tag, kids: [], _text: '', _raw: null, className: '', title: '', hidden: false,
    append(...cs) { for (const c of cs) n.kids.push(c); },
    appendChild(c) { n.kids.push(c); return c; },
    addEventListener() {},
    querySelector() { if (!n._q) n._q = node('div'); return n._q; },
    get textContent() { return n._text; },
    set textContent(v) { n._text = String(v); n._raw = null; },
    get innerHTML() { return n._raw !== null ? n._raw : esc(n._text); },
    set innerHTML(v) { n._raw = v; n.kids = []; },
  };
  return n;
}
const doc = {};
function $(id) { if (!doc[id]) doc[id] = node('div'); return doc[id]; }
const document = { createElement: node };
const resourcesSummary = () => ({ text: '' });
const waitingLine = () => '';
const residencyLabel = () => '';
const pinLabel = () => '';
const debugArmedFor = () => false;
const transcriptPill = () => '';
const sequencesInForce = () => 1;
const modelInfoLine = () => '';
const measurementText = () => '';
const toolCallText = () => '';
const loadFailureText = () => '';
const btn = (label) => { const b = node('button'); b.textContent = label; return b; };
const confirmBtn = btn;
const postModel = () => ({ catch() {} });
const postDebugLog = postModel;
const alertErr = () => {};
`)
	b.WriteString("const state = { models: [" + modelJSON + "] };\n")
	for _, name := range []string{"escapeHtml", "renderModels", "updateText", "updateMark", "updateOffered", "updateFilesText"} {
		b.WriteString(extractFunction(t, src, name))
		b.WriteString("\n")
	}
	b.WriteString(`renderModels();
const card = doc['modelList'].kids[0];
process.stdout.write(JSON.stringify({markup: card.innerHTML}));`)
	var out struct {
		Markup string `json:"markup"`
	}
	if err := json.Unmarshal([]byte(evalJS(t, b.String())), &out); err != nil {
		t.Fatalf("the panel did not render a card: %v", err)
	}
	return out.Markup
}

// The card names the files a newer version changes, and every name on it
// that came from HuggingFace — the repository and each file — appears as
// text, never as markup, however it is spelled (itd-2610030857275099
// criterion 11; iss-2610042101436222). The registry admits plain names only,
// so this is the panel's own half of the guarantee, rendered rather than
// searched for.
func TestTheCardShowsHuggingFaceNamesAsText(t *testing.T) {
	const evil = `<img src=x onerror=alert(1)>`
	// The model as Go publishes it, so the fields the card reads are the
	// registry's own and not a spelling this test made up.
	model, err := json.Marshal(registry.Model{
		RepoID: "org/" + evil,
		State:  registry.StateReady,
		Commit: "1111111111111111111111111111111111111111",
		Update: &registry.UpdateCheck{
			Status:       registry.UpdateAvailable,
			Commit:       "2222222222222222222222222222222222222222",
			Files:        []string{"config.json", evil + ".safetensors"},
			FilesChanged: 14,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderModelCard(t, string(model))
	if strings.Contains(markup, "<img") {
		t.Fatalf("a name from HuggingFace reached the card as markup:\n%s", markup)
	}
	escaped := `&lt;img src=x onerror=alert(1)&gt;`
	for _, want := range []string{
		"org/" + escaped,
		"config.json",
		escaped + ".safetensors",
		"and 12 more",
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("the card does not show %q as text:\n%s", want, markup)
		}
	}
}

// The line naming the files says which files and how many more there are, and
// is drawn only beside a newer version the card shows.
func TestTheChangedFilesLineNamesThemAndCountsTheRest(t *testing.T) {
	const c = `"state":"ready","commit":"1111111111111111111111111111111111111111"`
	const v = `"commit":"2222222222222222222222222222222222222222"`
	cases := []struct{ model, want string }{
		{`{` + c + `,"update":{"status":"available",` + v + `,"files":["a.json","b.json"],"files_changed":2}}`,
			`Files it changes: a.json, b.json.`},
		{`{` + c + `,"update":{"status":"available",` + v + `,"files":["a.json"],"files_changed":13}}`,
			`Files it changes: a.json and 12 more.`},
		{`{` + c + `,"update":{"status":"runs_own_code",` + v + `,"files":["config.json"],"files_changed":1}}`,
			`Files it changes: config.json.`},
		{`{` + c + `,"update":{"status":"cannot_check",` + v + `,"files":["config.json"],"files_changed":1}}`,
			`Files it changes: config.json.`},
		{`{` + c + `,"update":{"status":"available",` + v + `}}`, ``},
		{`{` + c + `,"update":{"status":"current","files":["a.json"],"files_changed":1}}`, ``},
		{`{` + c + `,"updating":3,"update":{"status":"available",` + v + `,"files":["a.json"],"files_changed":1}}`, ``},
		{`{"state":"ready","update":{"status":"available",` + v + `,"files":["a.json"],"files_changed":1}}`, ``},
	}
	for _, tc := range cases {
		if got := evalPanel(t, "updateFilesText("+tc.model+")", "updateFilesText", "updateMark"); got != tc.want {
			t.Errorf("updateFilesText(%s) = %q, want %q", tc.model, got, tc.want)
		}
	}
}

package ui

import (
	"regexp"
	"strings"
	"testing"
)

// The card says what the last check found, in words: a newer version, one
// Dessau will not run, one waiting for review, a version never recorded, and
// an update under way; and Update appears exactly where there is something
// Dessau would run to update to (itd-2610030857275099 criteria 4, 7, 8, 9).
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
			`will be offered once a Dessau release has reviewed it`, `newer version awaiting review`, false},
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

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

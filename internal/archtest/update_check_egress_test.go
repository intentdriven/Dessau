package archtest_test

import (
	"strings"
	"testing"
)

// The switch that sends a request to HuggingFace about every downloaded model
// cannot be drawn without the sentence that says what it sends and to whom,
// in the same Settings block (spc-2610030929021692, "The setting": "beside the
// switch one sentence states what a check sends and to whom, held by a test
// the way the bridge's egress sentence is"; adr-2610030857208746).
//
// Scoped to the fieldset for the reason TestTheBridgeSwitchCarriesTheEgressSentence
// gives: a sentence three screens away from the switch is not one the person
// turning it on has read.
func TestTheUpdateCheckSwitchCarriesTheEgressSentence(t *testing.T) {
	block := fieldsetAround(t, `id="setUpdateCheck"`)
	for _, phrase := range []string{
		"Off unless you turn it on",
		"sends HuggingFace the name of each model you downloaded",
		"no access token",
		"refuses",
	} {
		if !strings.Contains(block, phrase) {
			t.Errorf("the update check's Settings block does not say %q — the switch may not be drawn "+
				"without the sentence saying what a check sends and to whom", phrase)
		}
	}
	if !strings.Contains(block, `id="setUpdateCheckInterval"`) {
		t.Error("the interval is not in the same block as the switch")
	}
	if !strings.Contains(block, "docs/model-updates.md") {
		t.Error("the update check's block does not link its documentation page")
	}
}

// The docs page says the same thing the block says.
func TestTheModelUpdatesPageSaysWhatACheckSends(t *testing.T) {
	page := readDoc(t, "model-updates.md")
	for _, phrase := range []string{
		"name of each model",
		"no access token",
		"update_check_enabled",
		"update_check_interval_hours",
	} {
		if !strings.Contains(page, phrase) {
			t.Errorf("docs/model-updates.md does not say %q", phrase)
		}
	}
}

// fieldsetAround is the one Settings block that carries control, with its
// whitespace collapsed.
func fieldsetAround(t *testing.T, control string) string {
	t.Helper()
	markup := readPanelFile(t, "index.html")
	at := strings.Index(markup, control)
	if at < 0 {
		t.Fatalf("the Settings pane carries no %s", control)
	}
	start := strings.LastIndex(markup[:at], "<fieldset")
	end := strings.Index(markup[at:], "</fieldset>")
	if start < 0 || end < 0 {
		t.Fatalf("%s is not inside a Settings block", control)
	}
	return strings.Join(strings.Fields(markup[start:at+end]), " ")
}

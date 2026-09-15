package ui

import (
	"regexp"
	"testing"
)

func summary(t *testing.T, state string) map[string]any {
	t.Helper()
	return evalPanelValue(t, "resourcesSummary("+state+")", "bytes", "resourcesSummary")
}

// Three ready models and one resident: 3 downloaded, 1 loaded; the disk
// figure is the sum of the recorded sizes, and free disk is the snapshot's.
func TestTheRollUpCountsDownloadedAndLoaded(t *testing.T) {
	got := summary(t, `{
		"models": [
			{"repo_id":"org/a","state":"ready","bytes":1073741824},
			{"repo_id":"org/b","state":"ready","bytes":2147483648},
			{"repo_id":"org/c","state":"ready","bytes":3221225472},
			{"repo_id":"org/d","state":"downloading","bytes":99}],
		"resident": [{"repo_id":"org/a","state":"loaded"}],
		"machine": {"budget": 107374182400, "resident_bytes": 1288490188, "exiting_bytes": 0, "free_disk": 214748364800}}`)
	if got["downloaded"] != float64(3) || got["loaded"] != float64(1) {
		t.Errorf("downloaded=%v loaded=%v, want 3 and 1", got["downloaded"], got["loaded"])
	}
	if got["disk"] != float64(6*1073741824) {
		t.Errorf("disk = %v, want the sum of the three recorded sizes", got["disk"])
	}
	text, _ := got["text"].(string)
	for _, want := range []string{"3 models downloaded, 1 loaded", "6.0 GB on disk, 200.0 GB free", "memory: 1.2 GB of 100.0 GB budget resident"} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(text) {
			t.Errorf("text %q does not say %q", text, want)
		}
	}
}

// Memory still on its way out is part of the resident figure and named.
func TestTheRollUpNamesTheExitingPart(t *testing.T) {
	got := summary(t, `{"models": [], "resident": [],
		"machine": {"budget": 107374182400, "resident_bytes": 4294967296, "exiting_bytes": 4294967296, "stuck_servers": 1}}`)
	text, _ := got["text"].(string)
	for _, want := range []string{"4.0 GB of 100.0 GB budget resident", "of which 4.0 GB still exiting", "1 server stuck"} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(text) {
			t.Errorf("text %q does not say %q", text, want)
		}
	}
	if got["exiting"] != float64(4294967296) {
		t.Errorf("exiting = %v", got["exiting"])
	}
}

// The card shows both windows when the served one is below the declared,
// the served one resolved the fold-aware way, and no window at all — never
// zero — for a model that declares none.
func TestTheCardShowsBothWindowsFoldAwareAndNeverZero(t *testing.T) {
	label := func(model, config string) string {
		v := evalPanelValue(t, "({text: contextLabel("+model+", "+config+")})", "tokensLabel", "foldRepoID", "servedContext", "contextLabel")
		s, _ := v["text"].(string)
		return s
	}
	if got := label(`{"repo_id":"org/m","context_length":131072}`, `{"models":{"Org/M":{"served_context":65536}}}`); got != "context 128K declared · 64K served" {
		t.Errorf("both windows: %q", got)
	}
	if got := label(`{"repo_id":"org/m","context_length":131072}`, `{}`); got != "max context 128K" {
		t.Errorf("declared only: %q", got)
	}
	if got := label(`{"repo_id":"org/m","context_length":0}`, `{"models":{"org/m":{"served_context":65536}}}`); got != "" {
		t.Errorf("no declared window: %q, want nothing", got)
	}
	if got := label(`{"repo_id":"org/m"}`, `{}`); got != "" {
		t.Errorf("no context at all: %q, want nothing", got)
	}
}

// The roll-up is drawn from the snapshot on every models render, into the
// block at the head of the tab.
func TestTheRollUpIsDrawnAtTheHeadOfTheModelsTab(t *testing.T) {
	src := readPanelSource(t)
	if !regexp.MustCompile(`\$\('resources'\)\.textContent = resourcesSummary\(state\)\.text`).MatchString(src) {
		t.Error("renderModels no longer draws the roll-up")
	}
}

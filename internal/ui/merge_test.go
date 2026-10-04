package ui

import (
	"strings"
	"testing"
)

// The merge box is ticked the way the server decides: folded, so a setting
// stored under another spelling of the model's id still shows as on
// (iss-2609201015464436), as Config.MergeSystemMessages reads it.
func TestTheMergeBoxReadsTheSettingFolded(t *testing.T) {
	got := evalPanelValue(t,
		`{"exact": mergeFor({"org/m":{"merge_system_messages":true}}, "org/m"),
		  "folded": mergeFor({"ORG/M":{"merge_system_messages":true}}, "org/m"),
		  "other": mergeFor({"org/other":{"merge_system_messages":true}}, "org/m"),
		  "none": mergeFor(undefined, "org/m")}`,
		"foldRepoID", "mergeFor")
	want := map[string]any{"exact": true, "folded": true, "other": false, "none": false}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mergeFor %s = %v, want %v", k, got[k], v)
		}
	}
	if src := readPanelSource(t); !strings.Contains(src, "cb.checked = mergeFor(per, m.repo_id)") {
		t.Error("the merge boxes are not ticked through mergeFor")
	}
}

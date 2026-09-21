package config

import (
	"encoding/json"
	"strings"
	"testing"
)

// The helper's own edges, beside the save-path tests in internal/gateway
// that hold the behaviour end to end: a posted null names a model and says
// nothing about it, a posted value that is not an object is refused naming
// the model, and a nil posted map — the body did not name the collection —
// yields nothing for the caller to replace.
func TestMergeModelSettingsEdges(t *testing.T) {
	stored := map[string]ModelSettings{"org/a": {Pinned: true, NoTranscript: true}}

	got, err := MergeModelSettings(stored, map[string]json.RawMessage{"org/a": json.RawMessage(`null`)})
	if err != nil {
		t.Fatal(err)
	}
	if got["org/a"] != stored["org/a"] {
		t.Errorf("a posted null changed the stored entry: %+v", got["org/a"])
	}

	_, err = MergeModelSettings(stored, map[string]json.RawMessage{"org/a": json.RawMessage(`5`)})
	if err == nil || !strings.Contains(err.Error(), "org/a") {
		t.Errorf("a posted number was not refused naming the model: %v", err)
	}
	_, err = MergeModelSettings(stored, map[string]json.RawMessage{"org/a": json.RawMessage(`{"pinned":"yes"}`)})
	if err == nil || !strings.Contains(err.Error(), "org/a") {
		t.Errorf("a posted field of the wrong type was not refused naming the model: %v", err)
	}

	if got, err := MergeModelSettings(stored, nil); err != nil || got != nil {
		t.Errorf("MergeModelSettings(stored, nil) = %+v, %v; want nothing", got, err)
	}
	if got, err := MergeModelSettings(stored, map[string]json.RawMessage{}); err != nil || len(got) != 0 {
		t.Errorf("an empty posted map kept %+v", got)
	}
}

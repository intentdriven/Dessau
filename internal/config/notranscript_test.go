package config

import (
	"encoding/json"
	"testing"
)

// The exception is a fact about a model: set for one model, read for that
// model under any spelling of its id, and off for every other model
// (itd-2609091715089488). It fails closed: where two spellings of one id
// reach the reader and either carries the exception, the model is excepted,
// because the operator asked for nothing to be written and the map holding
// two spellings reached the reader some way the settings path refuses.
func TestNoTranscriptIsReadFoldedAndFailsClosed(t *testing.T) {
	c := Config{Models: map[string]ModelSettings{
		"org/quiet":    {NoTranscript: true},
		"org/recorded": {Pinned: true},
		// Two spellings of one model, only one of them excepted. The settings
		// path refuses such a map and the file path drops the duplicate, so
		// this one was assembled in Go; the reader still answers the same on
		// every run.
		"org/twice": {Pinned: true},
		"ORG/Twice": {NoTranscript: true},
	}}
	cases := []struct {
		name   string
		repoID string
		want   bool
	}{
		{"the exact key", "org/quiet", true},
		{"the id as the registry spells it", "ORG/Quiet", true},
		{"a model with other settings and no exception", "org/recorded", false},
		{"a model with no settings at all", "org/none", false},
		{"a duplicate under the exact key that carries no exception", "org/twice", true},
		{"a duplicate under the spelling that carries it", "ORG/Twice", true},
		{"a duplicate under a third spelling", "Org/TWICE", true},
	}
	for _, tc := range cases {
		if got := c.NoTranscript(tc.repoID); got != tc.want {
			t.Errorf("%s: NoTranscript(%q) = %v, want %v", tc.name, tc.repoID, got, tc.want)
		}
	}
	if (Config{}).NoTranscript("org/quiet") {
		t.Error("a configuration with no per-model settings excepts a model")
	}
}

// The field is stored under no_transcript, off by default, and an entry that
// carries only the exception is a setting rather than an empty object: it is
// kept on the file path and it survives a round trip through config.json.
func TestNoTranscriptIsAPerModelFieldOffByDefault(t *testing.T) {
	if Default().NoTranscript("org/any") {
		t.Error("the default configuration excepts a model from the recording")
	}
	if (ModelSettings{NoTranscript: true}).IsZero() {
		t.Error("an entry carrying only the exception reads as no settings at all; the file path would drop it")
	}
	b, err := json.Marshal(ModelSettings{NoTranscript: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"no_transcript":true}` {
		t.Errorf("the exception is stored as %s, want {\"no_transcript\":true}", b)
	}
	if b, _ := json.Marshal(ModelSettings{Pinned: true}); string(b) != `{"pinned":true}` {
		t.Errorf("a recorded model's entry carries the field anyway: %s", b)
	}
}

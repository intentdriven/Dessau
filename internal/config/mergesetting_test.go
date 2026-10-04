package config

import "testing"

// The merge setting is read folded, as every per-model setting is: set for
// one model, read for that model under any spelling of its id, and off for
// every other model (iss-2609201015464436).
func TestMergeSystemMessagesIsReadFolded(t *testing.T) {
	c := Config{Models: map[string]ModelSettings{
		"org/merged": {MergeSystemMessages: true},
		"org/plain":  {Pinned: true},
		"ORG/Upper":  {MergeSystemMessages: true},
	}}
	cases := []struct {
		name   string
		repoID string
		want   bool
	}{
		{"the exact key", "org/merged", true},
		{"another spelling of a stored key", "ORG/MERGED", true},
		{"a key stored under another spelling", "org/upper", true},
		{"a model with other settings and no merge", "org/plain", false},
		{"a model with no settings at all", "org/none", false},
	}
	for _, tc := range cases {
		if got := c.MergeSystemMessages(tc.repoID); got != tc.want {
			t.Errorf("%s: MergeSystemMessages(%q) = %v, want %v", tc.name, tc.repoID, got, tc.want)
		}
	}
	if (Config{}).MergeSystemMessages("org/merged") {
		t.Error("a config with no per-model settings merges")
	}
}

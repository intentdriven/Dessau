package hub

import "testing"

// A repo is offered as a model this server can run only when the Hub says it is
// an MLX one. The Hub says it two ways — the library tag a conversion sets, and
// the "-mlx" marker the convention puts in the repo name — and both are the
// Hub's own words rather than a guess about the weights.
func TestIsMLXReadsTheHubsOwnWords(t *testing.T) {
	cases := []struct {
		name string
		m    Model
		want bool
	}{
		{"library tag", Model{ID: "mlx-community/Qwen3-8B-4bit", Tags: []string{"mlx", "safetensors", "qwen3"}}, true},
		{"library tag, odd case", Model{ID: "alice/Model-4bit", Tags: []string{"MLX"}}, true},
		{"marker inside the name", Model{ID: "prism-ml/Ternary-Bonsai-2-27B-mlx-2bit", Tags: []string{"safetensors"}}, true},
		{"marker ending the name", Model{ID: "alice/Bonsai-27B-mlx"}, true},
		{"marker opening the name", Model{ID: "alice/mlx-Bonsai-27B"}, true},
		{"no tag and no marker", Model{ID: "alice/Bonsai-27B", Tags: []string{"transformers", "pytorch"}}, false},
		{"mlx only in the owner", Model{ID: "mlx-lovers/Bonsai-27B", Tags: []string{"pytorch"}}, false},
		{"mlx buried in a word", Model{ID: "alice/Amlxa-27B"}, false},
		{"nothing at all", Model{}, false},
	}
	for _, tc := range cases {
		if got := tc.m.IsMLX(); got != tc.want {
			t.Errorf("%s: Model{ID:%q,Tags:%v}.IsMLX() = %v, want %v", tc.name, tc.m.ID, tc.m.Tags, got, tc.want)
		}
	}
}

package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// writeTokenizerConfig puts a tokenizer_config.json beside a model's weights.
func writeTokenizerConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "tokenizer_config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The directory answers whether a model carries a chat template, offline: a
// non-empty chat_template in tokenizer_config.json, as the string most
// repositories write or the array of named templates some do, or a
// chat_template.jinja file beside it. A directory with neither, an empty
// template, or a tokenizer_config.json that will not parse carries none
// (iss-2609202237468921).
func TestInspectModelDirReportsTheChatTemplate(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  bool
	}{
		{"a string template", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": "{% for m in messages %}{{ m.content }}{% endfor %}"}`)
		}, true},
		{"an array of named templates", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": [{"name": "default", "template": "{{ messages }}"}]}`)
		}, true},
		{"a chat_template.jinja file", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"bos_token": "<s>"}`)
			if err := os.WriteFile(filepath.Join(dir, "chat_template.jinja"), []byte("{{ messages }}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"the jinja file alone, with no tokenizer_config.json", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "chat_template.jinja"), []byte("{{ messages }}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"no template anywhere", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"bos_token": "<s>"}`)
		}, false},
		{"no tokenizer_config.json at all", func(t *testing.T, dir string) {}, false},
		{"an empty string template", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": ""}`)
		}, false},
		{"an empty array of templates", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": []}`)
		}, false},
		{"a null template", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": null}`)
		}, false},
		{"a template of the wrong shape", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": 42}`)
		}, false},
		{"a malformed tokenizer_config.json", func(t *testing.T, dir string) {
			writeTokenizerConfig(t, dir, `{"chat_template": "{{ messages }}"`)
		}, false},
		{"an empty chat_template.jinja", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "chat_template.jinja"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			dir := writeModelDir(t, root, "org", "m", 64)
			c.setup(t, dir)
			complete, _, facts := inspectModelDir(dir)
			if !complete {
				t.Fatal("the directory is a complete model; the template must not decide that")
			}
			if facts.ChatTemplate != c.want {
				t.Errorf("ChatTemplate = %v, want %v", facts.ChatTemplate, c.want)
			}
			if got := ReadModelFacts(dir).ChatTemplate; got != c.want {
				t.Errorf("ReadModelFacts().ChatTemplate = %v, want %v — the download path reads the same fact", got, c.want)
			}
		})
	}
}

// tokenizer_config.json is, in the shared cache, a file another account can
// write, so it is read under the same cap every other manifest is and an
// oversized one says "no template" rather than being read whole.
func TestAnOversizedTokenizerConfigCarriesNoTemplate(t *testing.T) {
	root := t.TempDir()
	dir := writeModelDir(t, root, "org", "m", 64)
	body := `{"chat_template": "{{ messages }}", "pad": "` + strings.Repeat("x", maxManifestJSON) + `"}`
	writeTokenizerConfig(t, dir, body)
	if _, _, facts := inspectModelDir(dir); facts.ChatTemplate {
		t.Error("an oversized tokenizer_config.json was believed")
	}
}

// Rescan assigns the template the way it assigns the context length: on a
// model it adopts, and on an entry it already holds, so a model recorded by a
// build that predates the fact gains it at the next start.
func TestRescanAssignsTheChatTemplate(t *testing.T) {
	r, root := newTestRegistry(t)
	models := filepath.Join(root, "models")
	dir := writeModelDir(t, models, "org", "chatty", 64)
	writeTokenizerConfig(t, dir, `{"chat_template": "{{ messages }}"}`)
	writeModelDir(t, models, "org", "base", 64)

	// An existing entry that predates the fact.
	if err := r.Put(Model{RepoID: "org/chatty", Path: dir, State: StateReady}); err != nil {
		t.Fatal(err)
	}
	if err := r.Rescan(models); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/chatty"); !m.ChatTemplate {
		t.Error("the existing entry did not gain ChatTemplate from its directory")
	}
	if m, _ := r.Get("org/base"); m.ChatTemplate {
		t.Error("a model with no template was adopted as carrying one")
	}

	// And re-derived, not merely set once: a template that goes, goes.
	if err := os.Remove(filepath.Join(dir, "tokenizer_config.json")); err != nil {
		t.Fatal(err)
	}
	if err := r.Rescan(models); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/chatty"); m.ChatTemplate {
		t.Error("ChatTemplate survived the template's removal; the fact is the directory's, re-derived at every rescan")
	}
}

// The fact survives a restart, and is absent from the file rather than written
// as false, like every other omitempty fact on the record.
func TestTheChatTemplateRoundTripsThroughTheIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/chatty", Path: dir, State: StateReady, ChatTemplate: true}); err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/base", Path: dir, State: StateReady}); err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := again.Get("org/chatty"); !m.ChatTemplate {
		t.Error("ChatTemplate did not survive a reopen")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), `"chat_template"`); n != 1 {
		t.Errorf("the index writes chat_template %d times for two models, one with a template; want once", n)
	}
}

// SetCategory records what the Hub said about a model the download did not
// hear it for, bounded the way a download's answer is, and broadcasts so the
// panel and the models list see the change.
func TestSetCategoryRecordsBoundsAndBroadcasts(t *testing.T) {
	r, dir := newTestRegistry(t)
	if err := r.Put(Model{RepoID: "org/adopted", Path: dir, State: StateReady}); err != nil {
		t.Fatal(err)
	}
	ch, unsub := r.Subscribe()
	defer unsub()

	long := strings.Repeat("x", MaxTagBytes+1)
	if err := r.SetCategory("org/adopted", "text-generation", []string{"mlx", long, "with\x00nul", "conversational", "mlx"}); err != nil {
		t.Fatal(err)
	}
	m, err := r.Get("org/adopted")
	if err != nil {
		t.Fatal(err)
	}
	if m.PipelineTag != "text-generation" {
		t.Errorf("PipelineTag = %q, want text-generation", m.PipelineTag)
	}
	if got := strings.Join(m.Tags, ","); got != "mlx,conversational" {
		t.Errorf("Tags = %v, want the usable ones, once each", m.Tags)
	}
	if m.HubSilent {
		t.Error("a model the Hub had words for is marked as one it had none for")
	}
	select {
	case snap := <-ch:
		for _, s := range snap {
			if s.RepoID == "org/adopted" && s.PipelineTag != "text-generation" {
				t.Error("the broadcast snapshot does not carry the category")
			}
		}
	default:
		t.Error("SetCategory did not broadcast")
	}

	// A pipeline tag beyond the bound is dropped, not cut down.
	if err := r.SetCategory("org/adopted", long, nil); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/adopted"); m.PipelineTag != "" {
		t.Errorf("PipelineTag = %q, want it dropped", m.PipelineTag)
	}

	if err := r.SetCategory("org/unknown", "text-generation", nil); err == nil {
		t.Error("SetCategory on a model the index does not hold must fail")
	}
}

// A Hub that answered and had no words for a repository is not asked again at
// every start: SetCategory with nothing usable marks the model as one the Hub
// is silent about, and an answer with words clears the mark.
func TestSetCategoryWithNoWordsMarksTheHubSilent(t *testing.T) {
	r, dir := newTestRegistry(t)
	if err := r.Put(Model{RepoID: "org/quiet", Path: dir, State: StateReady}); err != nil {
		t.Fatal(err)
	}
	if err := r.SetCategory("org/quiet", "", nil); err != nil {
		t.Fatal(err)
	}
	m, _ := r.Get("org/quiet")
	if !m.HubSilent || m.PipelineTag != "" || m.Tags != nil {
		t.Errorf("after an empty answer: %+v, want HubSilent and no words", m)
	}
	if err := r.SetCategory("org/quiet", "text-generation", []string{"conversational"}); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/quiet"); m.HubSilent {
		t.Error("an answer with words left the model marked silent")
	}
}

// The one home of the chat verdict. When the Hub's word is present — a
// pipeline tag or any tag — the operator's rule decides, exactly as it did;
// when it is absent, the directory's chat template decides.
func TestCanChatHasOneAnswer(t *testing.T) {
	rule := config.DefaultChatRule()
	cases := []struct {
		name string
		m    Model
		want bool
	}{
		{"hub word present, rule matches", Model{PipelineTag: "text-generation", Tags: []string{"conversational"}}, true},
		{"hub word present, rule matches, no template needed", Model{PipelineTag: "text-generation", Tags: []string{"conversational"}, ChatTemplate: false}, true},
		{"hub word present, rule does not match, template present", Model{PipelineTag: "automatic-speech-recognition", Tags: []string{"conversational"}, ChatTemplate: true}, false},
		{"only tags present, rule does not match, template present", Model{Tags: []string{"mlx"}, ChatTemplate: true}, false},
		{"only a pipeline tag present, rule does not match, template present", Model{PipelineTag: "text-generation", ChatTemplate: true}, false},
		{"hub word absent, template present", Model{ChatTemplate: true}, true},
		{"hub word absent, no template", Model{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.m.CanChat(rule); got != c.want {
				t.Errorf("CanChat = %v, want %v", got, c.want)
			}
		})
	}

	// A rule that tests nothing marks every model with a Hub word as able to
	// chat, as it always has; a model with none is still its template's.
	empty := config.ChatRule{PipelineTags: []string{}, RequiredTags: []string{}}
	if !(Model{PipelineTag: "automatic-speech-recognition"}).CanChat(empty) {
		t.Error("an empty rule must match every model the Hub has a word for")
	}
	if (Model{}).CanChat(empty) {
		t.Error("an empty rule has nothing to say about a model with no Hub word and no template")
	}
}

// HasHubWord is what tells the two halves of CanChat apart, and what the
// completion job walks the registry by.
func TestHasHubWord(t *testing.T) {
	if (Model{}).HasHubWord() {
		t.Error("no pipeline tag and no tags is no word")
	}
	if !(Model{PipelineTag: "text-generation"}).HasHubWord() {
		t.Error("a pipeline tag is a word")
	}
	if !(Model{Tags: []string{"mlx"}}).HasHubWord() {
		t.Error("a tag is a word")
	}
}

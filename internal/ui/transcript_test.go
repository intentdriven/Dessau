package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Some models keep no transcript even while recording is on
// (itd-2609091715089488): the exception is one box per model in Settings,
// posted on the same per-model map as the pin and the merging switch, and
// the model's card says what the models list says — whether a conversation
// with it is recorded — from the same snapshot in one named helper. What is
// held here is the wiring the settings-surface walk in internal/archtest
// cannot reach: that the row exists, that the box posts, and that the card's
// words are computed from the two facts the value is made of.

// The transcript box is one row of the per-model form, and it posts
// explicitly: ticked is no_transcript:true, clear is no_transcript:false.
// The server merges the map field by field now, so a box the form drew and
// left clear has to SAY it is clear — an absent key means "keep what is
// stored", and a clear box that posted nothing would leave an exception in
// force that the operator had just switched off.
func TestSettingsFormPostsTheTranscriptBox(t *testing.T) {
	const qwen = "mlx-community/Qwen3-8B-4bit"
	cases := []struct {
		name string
		expr string
		want map[string]any
	}{
		{
			name: "a ticked box excepts the model",
			expr: `modelSettings({}, {}, [], [], [], [], [], {}, ["` + qwen + `"], ["` + qwen + `"])`,
			want: map[string]any{qwen: map[string]any{"no_transcript": true}},
		},
		{
			name: "a clear box says so rather than saying nothing",
			expr: `modelSettings({"` + qwen + `":{"no_transcript":true}}, {}, [], [], [], [], [], {}, ["` + qwen + `"], [])`,
			want: map[string]any{qwen: map[string]any{"no_transcript": false}},
		},
		{
			name: "the exception sits beside the other switches on one model",
			expr: `modelSettings({}, {}, ["org/a"], ["org/a"], ["org/a"], [], ["org/a"], {"org/a":0}, ["org/a"], ["org/a"])`,
			want: map[string]any{"org/a": map[string]any{
				"merge_system_messages": true,
				"pinned":                false,
				"served_context":        float64(0),
				"no_transcript":         true,
			}},
		},
		{
			name: "a model the form drew no transcript box for keeps its exception",
			expr: `modelSettings({"org/not-downloaded":{"no_transcript":true}}, {}, [], [], [], [], [], {}, ["org/a"], [])`,
			want: map[string]any{
				"org/not-downloaded": map[string]any{"no_transcript": true},
				"org/a":              map[string]any{"no_transcript": false},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evalPanelValue(t, c.expr, "modelSettings", "applyModelSwitch", "applyModelNumber")
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s = %#v, want %#v", c.expr, got, c.want)
			}
		})
	}
}

// Every exception has to be removable from the form that shows it, and an
// exception can be set on a model that has not been downloaded — by hand in
// config.json, which is the point of setting it before the first request the
// model ever serves. So the rows are the pin rows' shape: one per model on
// this Mac, then one per excepted model this Mac does not have, matched
// folded so a spelling typed by hand still finds its box.
func TestSettingsFormDrawsARowForEveryException(t *testing.T) {
	const models = `[{"repo_id":"org/here","bytes":10},{"repo_id":"org/Plain","bytes":10}]`
	got := evalPanelValue(t,
		fmt.Sprintf(`{"rows": transcriptRows(%s, {"ORG/HERE":{"no_transcript":true},"org/gone":{"no_transcript":true},"org/pinned-only":{"pinned":true}})}`, models),
		"foldRepoID", "noTranscriptFor", "transcriptRows")
	want := map[string]any{"rows": []any{
		map[string]any{"id": "org/here", "checked": true, "absent": false},
		map[string]any{"id": "org/Plain", "checked": false, "absent": false},
		map[string]any{"id": "org/gone", "checked": true, "absent": true},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("transcriptRows = %v, want %v", got, want)
	}
}

// The card computes the value the models list publishes — the machine-wide
// switch AND the model not excepted, the exception read folded — from the
// snapshot it already holds, in one helper, and turns it into the two words
// the chat client's picker uses for the same fact. A rule stated in two
// places rather than one fact read twice, so both states are pinned here.
func TestTheCardSaysWhetherAModelIsRecorded(t *testing.T) {
	cases := []struct {
		name     string
		expr     string
		recorded bool
		words    string
	}{
		{"on and not excepted", `transcriptState({"transcript":true,"models":{}}, "org/m")`, true, "recorded"},
		{"on and excepted", `transcriptState({"transcript":true,"models":{"org/m":{"no_transcript":true}}}, "org/m")`, false, "keeps no transcript"},
		{"on and excepted under another spelling", `transcriptState({"transcript":true,"models":{"ORG/M":{"no_transcript":true}}}, "org/m")`, false, "keeps no transcript"},
		{"another model is excepted", `transcriptState({"transcript":true,"models":{"org/other":{"no_transcript":true}}}, "org/m")`, true, "recorded"},
		{"the switch is off", `transcriptState({"transcript":false,"models":{}}, "org/m")`, false, "keeps no transcript"},
		{"the snapshot says nothing", `transcriptState({}, "org/m")`, false, "keeps no transcript"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evalPanelValue(t, c.expr, "foldRepoID", "noTranscriptFor", "transcriptState")
			if got["recorded"] != c.recorded || got["words"] != c.words {
				t.Errorf("%s = %v, want recorded=%v words=%q", c.expr, got, c.recorded, c.words)
			}
		})
	}
}

// The pill the card draws carries the words as its accessible label, so a
// screen reader says "keeps no transcript" rather than the glyph's name, and
// it is drawn from the helper above rather than from a second reading of the
// fields.
func TestTheCardDrawsTheTranscriptPillWithItsWords(t *testing.T) {
	for _, c := range []struct{ expr, words string }{
		{`transcriptPill({"transcript":true,"models":{"org/m":{"no_transcript":true}}}, "org/m")`, "keeps no transcript"},
		{`transcriptPill({"transcript":true,"models":{}}, "org/m")`, "recorded"},
	} {
		got := evalPanel(t, c.expr, "foldRepoID", "noTranscriptFor", "transcriptState", "transcriptPill")
		if !strings.Contains(got, `aria-label="`+c.words+`"`) || !strings.Contains(got, `role="img"`) {
			t.Errorf("%s = %q, want a role=\"img\" pill labelled %q", c.expr, got, c.words)
		}
	}
	body := extractFunction(t, readPanelSource(t), "renderModels")
	if !strings.Contains(body, "transcriptPill(state.config, m.repo_id)") {
		t.Error("renderModels does not draw the transcript pill from transcriptPill(state.config, m.repo_id)")
	}
}

// The form both draws the transcript boxes and posts them: the lines the
// value tests above cannot reach without a DOM.
func TestSettingsFormIsWiredToTheTranscriptSwitches(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []string{
		"renderTranscriptSwitches()",
		"listedTranscriptModels(), checkedTranscriptModels()",
		"#transcriptList input[type=checkbox]",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("the control panel no longer contains %q — the transcript boxes are then asserted by nothing", want)
		}
	}
	markup := readPanelMarkup(t)
	if !strings.Contains(markup, `id="transcriptList"`) {
		t.Error("the Settings pane has no transcriptList to draw the boxes into")
	}
}

// Both panels say, before Alice tries, how the exception and per-model debug
// logging meet (itd-2609091715089488, the 2026-09-20 decision): the debug
// paragraph says a model that keeps no transcript refuses arming, and the
// transcript control says switching it on takes the model off debug logging.
// The refusal is enforced at the arming endpoint; these are the sentences
// that make it unsurprising.
func TestBothPanelsSayHowTheExceptionMeetsDebugLogging(t *testing.T) {
	markup := readPanelMarkup(t)
	debug := paragraphAfter(t, markup, `id="debugLogBlurb"`)
	if !strings.Contains(debug, "keeps no transcript refuses this") {
		t.Errorf("the debug-logging paragraph does not say a model that keeps no transcript refuses arming:\n%s", debug)
	}
	transcript := hintUnder(t, markup, "<legend>Transcript</legend>")
	for _, want := range []string{
		"debug logging",
		"cannot be armed",
		"refuse",
	} {
		if !strings.Contains(transcript, want) {
			t.Errorf("the transcript control's paragraph does not say %q:\n%s", want, transcript)
		}
	}
}

// The words beside the transcript box are a promise about what Dessau does
// with a person's conversation, so they are pinned whole, as the statistics
// switch's are.
func TestTheTranscriptControlExplainsItselfInWholeSentences(t *testing.T) {
	const want = "Off for every model unless you switch it on here. Tick a model and nothing it is asked " +
		"and nothing it answers is written to the transcript, whether or not the transcript is on, " +
		"from the next request it serves; every other model goes on as it was. The exception is a " +
		"fact about the model that answers a request: a request a recorded model answers is written " +
		"down whole, including earlier turns from an excepted model that the client carried back. " +
		"Every client is told, in the models list, whether a model is recorded, and Dessau Chat " +
		"shows it beside the model before you pick it. A model that keeps no transcript is not " +
		"offered over the Discord bridge, because Discord would keep what Dessau does not, and " +
		"debug logging cannot be armed on it: the arming is refused with the reason, and ticking " +
		"this box takes a model off the debug-logging list. The transcript page says all of this."
	if got := hintUnder(t, markup(t), "<legend>Transcript</legend>"); got != want {
		t.Errorf("the paragraph beside the transcript boxes reads:\n  %s\nwant:\n  %s", got, want)
	}
	if !strings.Contains(markup(t), `href="https://github.com/intentdriven/Dessau/blob/main/docs/transcript.md"`) {
		t.Error("the transcript control no longer links to the page that says what the exception does")
	}
}

func markup(t *testing.T) string {
	t.Helper()
	return readPanelMarkup(t)
}

// paragraphAfter is the text of the element carrying the marker, up to its
// closing paragraph tag, tags removed and whitespace collapsed.
func paragraphAfter(t *testing.T, markup, marker string) string {
	t.Helper()
	i := strings.Index(markup, marker)
	if i < 0 {
		t.Fatalf("the panel's markup carries no %s", marker)
	}
	rest := markup[i:]
	open := strings.Index(rest, ">")
	end := strings.Index(rest, "</p>")
	if open < 0 || end < open {
		t.Fatalf("the paragraph at %s is never closed", marker)
	}
	return strings.Join(strings.Fields(stripTags(rest[open+1:end])), " ")
}

// hintUnder is the first hint paragraph after a legend, as a reader sees
// it: the words between the tags, whitespace collapsed.
func hintUnder(t *testing.T, markup, legend string) string {
	t.Helper()
	start := strings.Index(markup, legend)
	if start < 0 {
		t.Fatalf("the settings form has no %s group", legend)
	}
	rest := markup[start:]
	open := strings.Index(rest, `<p class="hint">`)
	end := strings.Index(rest, "</p>")
	if open < 0 || end < open {
		t.Fatalf("the %s group has no hint paragraph", legend)
	}
	return strings.Join(strings.Fields(stripTags(rest[open+len(`<p class="hint">`):end])), " ")
}

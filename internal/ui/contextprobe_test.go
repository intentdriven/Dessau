package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/Gropius/internal/config"
)

// The probe's two settings are one setting each on three surfaces, and the
// pane is held to the file's keys: filled from them, posted under them.
func TestThePaneIsWiredToTheProbeSettings(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`context_probe:\s*\$\('setContextProbe'\)\.checked`),
		regexp.MustCompile(`\$\('setContextProbe'\)\.checked\s*=\s*!!c\.context_probe`),
		regexp.MustCompile(`idle_threshold_sec:\s*parseInt\(\$\('setIdleThreshold'\)\.value, 10\) \|\| 0`),
		regexp.MustCompile(`\$\('setIdleThreshold'\)\.value\s*=\s*c\.idle_threshold_sec`),
		regexp.MustCompile(`postModel\('/api/models/measure'`),
		regexp.MustCompile(`postModel\('/api/models/adopt'`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that behavior is then asserted by nothing", want)
		}
	}
}

// The card's line says what the result is and what bounded it, in words a
// reader can act on: a limit is a limit, a floor says so, a stale figure says
// why, a run in progress says where it is, and a held run says what held it.
func TestTheResultSaysWhatBoundedIt(t *testing.T) {
	line := func(expr string) string {
		v := evalPanelValue(t, "({text: "+expr+"})", "measurementText", "boundText", "staleText")
		s, _ := v["text"].(string)
		return s
	}
	cases := []struct {
		name, expr, want string
	}{
		{"a limit", `measurementText({repo_id:"org/m", measured:{window:91000, bound:"model"}}, {}, [])`,
			"Measured 91,000 tokens — the model refused above this: its own limit here"},
		{"a floor at the deadline", `measurementText({repo_id:"org/m", measured:{window:91000, bound:"prefill_deadline"}}, {}, [])`,
			"Measured 91,000 tokens — a floor: the prefill deadline stopped the probe first"},
		{"a floor at the served window", `measurementText({repo_id:"org/m", measured:{window:16384, bound:"served_window"}}, {}, [])`,
			"Measured 16,384 tokens — a floor: the served window stopped the probe first"},
		{"stale", `measurementText({repo_id:"org/m", measured:{window:91000, bound:"model", stale:"runtime"}}, {}, [])`,
			"Measured 91,000 tokens, now stale: the runtime changed; measure again"},
		{"in progress", `measurementText({repo_id:"org/m"}, {job:"context-probe", model:"Org/M", step:"bisecting at 65536 tokens"}, [])`,
			"Measuring: bisecting at 65536 tokens"},
		{"held back", `measurementText({repo_id:"org/m"}, {held_by:"in_flight"}, ["org/m"])`,
			"Measurement waiting: a request is in flight"},
		{"held for want of room", `measurementText({repo_id:"org/m"}, {held_by:"no_room"}, ["org/m"])`,
			"Measurement waiting: the model does not fit the memory budget"},
		{"queued", `measurementText({repo_id:"org/m"}, {}, ["org/m"])`,
			"Measurement queued for the next idle minute"},
		{"incomplete", `measurementText({repo_id:"org/m", probe_incomplete:true}, {}, [])`,
			"Measurement incomplete: the last probe was interrupted; press Measure now to run it again"},
		{"nothing", `measurementText({repo_id:"org/m"}, {}, [])`, ""},
	}
	for _, tt := range cases {
		if got := line(tt.expr); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}

// The idle threshold is the same drift one fieldset down from the grace pair:
// a blank field is the default, and the form said which figure that was twice
// over — placeholder="300" on the field and "Blank is five minutes" in the
// prose beside it, both of them config.DefaultIdleThresholdSec written out
// again where nothing binds them to the server (iss-2609190146152463). Both
// are rendered from the snapshot, so a Mac serving a different threshold says
// so and a panel that has not been told the figure claims nothing.
func TestTheIdleThresholdDefaultIsFilledFromTheServer(t *testing.T) {
	page, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if got := attr(fieldTag(t, string(page), "setIdleThreshold"), "placeholder"); got != "" {
		t.Errorf("setIdleThreshold carries placeholder=%q in the markup — a figure nothing binds "+
			"to the server, which is the copy this field is bound to the snapshot to be rid of", got)
	}
	if strings.Contains(string(page), "Blank is five minutes") {
		t.Error("the prose beside the idle threshold still spells the default out in words; " +
			"it is written from the snapshot's figure")
	}

	doc := evalPanelDOM(t, fmt.Sprintf("renderDefaults({\"idle_threshold_sec\":%d});",
		config.DefaultIdleThresholdSec), "renderDefaults", "blankIsSentence", "intervalWords")
	if got, _ := doc["setIdleThreshold"]["placeholder"].(string); got != strconv.Itoa(config.DefaultIdleThresholdSec) {
		t.Errorf("setIdleThreshold's placeholder is %q, want the server's default %d",
			got, config.DefaultIdleThresholdSec)
	}
	said, _ := doc["idleThresholdDefault"]["textContent"].(string)
	if !strings.Contains(said, "Blank is 5 minutes") {
		t.Errorf("the prose beside the field reads %q, want it to say the default is 5 minutes", said)
	}

	// A threshold that is not a whole number of minutes is said in seconds
	// rather than rounded into a figure the server does not hold.
	odd := evalPanelDOM(t, `renderDefaults({"idle_threshold_sec":90});`, "renderDefaults", "blankIsSentence", "intervalWords")
	if said, _ := odd["idleThresholdDefault"]["textContent"].(string); !strings.Contains(said, "Blank is 90 seconds") {
		t.Errorf("a 90-second threshold reads %q, want it said in seconds", said)
	}

	// Told nothing, it claims nothing: no placeholder and no sentence beats a
	// figure the panel chose for itself.
	blank := evalPanelDOM(t, "renderDefaults(undefined);", "renderDefaults", "blankIsSentence", "intervalWords")
	if got, _ := blank["setIdleThreshold"]["placeholder"].(string); got != "" {
		t.Errorf("setIdleThreshold's placeholder is %q for a snapshot carrying no defaults, want it blank", got)
	}
	if got, _ := blank["idleThresholdDefault"]["textContent"].(string); got != "" {
		t.Errorf("the prose beside the field reads %q for a snapshot carrying no defaults, want nothing", got)
	}
}

// PR 108 bound the settings field and the sentence beside it to the server's
// figure, and left three copies of it in status prose: the Self-test hint in
// the markup, and two status lines in the panel — the posture page's self-test
// line and the Self-test view's own hint. A Mac serving a different threshold
// told the operator five minutes anyway (iss-2609190201317726).
//
// All three now read the threshold in force — the setting where one is set and
// the default otherwise, which is config.EffectiveIdleThresholdSec on the Go
// side — and a panel that has not been told it names no figure at all.
func TestTheIdleThresholdInStatusProseIsFilledFromTheServer(t *testing.T) {
	page, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	literal := regexp.MustCompile(`\b` + strconv.Itoa(config.DefaultIdleThresholdSec) + `\b`)
	for name, body := range map[string]string{"index.html": string(page), "app.js": readPanelSource(t)} {
		if strings.Contains(body, "five minutes") {
			t.Errorf("%s still spells the idle threshold out as \"five minutes\"; the figure is the "+
				"server's, off the snapshot, and a Mac serving another one must say so", name)
		}
		if loc := literal.FindStringIndex(body); loc != nil {
			t.Errorf("%s writes config.DefaultIdleThresholdSec out as a literal at byte %d; "+
				"the panel is told the figure and never holds a copy of it", name, loc[0])
		}
	}

	// The words the prose is written in: minutes for a whole number of them,
	// seconds otherwise, and nothing at all for a figure nobody sent.
	words := func(expr string) string {
		v := evalPanelValue(t, "({text: "+expr+"})", "intervalWords")
		s, _ := v["text"].(string)
		return s
	}
	for expr, want := range map[string]string{
		"intervalWords(300)": "5 minutes",
		"intervalWords(60)":  "1 minute",
		"intervalWords(90)":  "90 seconds",
		"intervalWords(0)":   "",
	} {
		if got := words(expr); got != want {
			t.Errorf("%s = %q, want %q", expr, got, want)
		}
	}

	// The threshold in force: the setting where one is set, the default where
	// it is not, and a phrase naming no figure where the panel knows neither.
	said := func(snapshot string) string {
		v := evalPanelValue(t, "({text: idleThresholdWords("+snapshot+")})", "intervalWords", "idleThresholdWords")
		s, _ := v["text"].(string)
		return s
	}
	if got := said(`{"config":{},"defaults":{"idle_threshold_sec":300}}`); got != "5 minutes" {
		t.Errorf("with only a default served the prose says %q, want the default in words", got)
	}
	if got := said(`{"config":{"idle_threshold_sec":600},"defaults":{"idle_threshold_sec":300}}`); got != "10 minutes" {
		t.Errorf("with a threshold set the prose says %q, want the setting and not the default", got)
	}
	if got := said(`{}`); got == "" || strings.ContainsAny(got, "0123456789") {
		t.Errorf("told no threshold the prose says %q, want a phrase carrying no figure", got)
	}

	// The Self-test hint in the markup: a span the panel fills, not a figure
	// written into the page.
	doc := evalPanelDOM(t, `renderIdleProse({"config":{"idle_threshold_sec":600},"defaults":{"idle_threshold_sec":300}});`,
		"intervalWords", "idleThresholdWords", "renderIdleProse")
	if got, _ := doc["selfTestIdleFigure"]["textContent"].(string); got != "10 minutes" {
		t.Errorf("the Self-test hint's figure reads %q, want the threshold in force", got)
	}
	if !strings.Contains(string(page), `id="selfTestIdleFigure"`) {
		t.Error("the Self-test hint carries no span for the threshold, so nothing can fill it from the snapshot")
	}

	// The posture page's self-test line.
	line := evalPanelValue(t, `(postureLines({"config":{"self_test":true,"idle_threshold_sec":600},`+
		`"defaults":{"idle_threshold_sec":300},"bind":{},"endpoints":[]}).find((l) => l.id === "selftest"))`,
		append(postureFunctions, "intervalWords")...)
	if text, _ := line["text"].(string); !strings.Contains(text, "10 minutes") {
		t.Errorf("the posture page's self-test line reads %q, want the threshold in force in it", text)
	}

	// The Self-test view's own hint, on and off.
	hint := func(expr string) string {
		v := evalPanelValue(t, "({text: "+expr+"})", "selfTestHint")
		s, _ := v["text"].(string)
		return s
	}
	if got := hint(`selfTestHint(true, true, "10 minutes")`); !strings.Contains(got, "10 minutes") {
		t.Errorf("the Self-test hint for a measured Mac reads %q, want the threshold in force in it", got)
	}
	if got := hint(`selfTestHint(true, false, "10 minutes")`); !strings.Contains(got, "10 minutes") {
		t.Errorf("the Self-test hint before the first run reads %q, want the threshold in force in it", got)
	}
	if got := hint(`selfTestHint(false, false, "10 minutes")`); strings.Contains(got, "10 minutes") {
		t.Errorf("the Self-test hint with the test off reads %q, and a test that is off runs at no interval", got)
	}
}

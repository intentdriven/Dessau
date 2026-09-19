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
		config.DefaultIdleThresholdSec), "renderDefaults", "blankIsSentence")
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
	odd := evalPanelDOM(t, `renderDefaults({"idle_threshold_sec":90});`, "renderDefaults", "blankIsSentence")
	if said, _ := odd["idleThresholdDefault"]["textContent"].(string); !strings.Contains(said, "Blank is 90 seconds") {
		t.Errorf("a 90-second threshold reads %q, want it said in seconds", said)
	}

	// Told nothing, it claims nothing: no placeholder and no sentence beats a
	// figure the panel chose for itself.
	blank := evalPanelDOM(t, "renderDefaults(undefined);", "renderDefaults", "blankIsSentence")
	if got, _ := blank["setIdleThreshold"]["placeholder"].(string); got != "" {
		t.Errorf("setIdleThreshold's placeholder is %q for a snapshot carrying no defaults, want it blank", got)
	}
	if got, _ := blank["idleThresholdDefault"]["textContent"].(string); got != "" {
		t.Errorf("the prose beside the field reads %q for a snapshot carrying no defaults, want nothing", got)
	}
}

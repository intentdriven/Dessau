package ui

import (
	"regexp"
	"testing"
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

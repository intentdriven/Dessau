package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// The self-test switch is one setting on three surfaces — config.json, the
// Go loop, and this pane — and the pane is held to the other two here: it
// fills the box from the key the file carries and posts it back under the
// same key, so what the operator ticks is what the loop reads.
func TestThePaneIsWiredToTheSelfTestSwitch(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`self_test:\s*\$\('setSelfTest'\)\.checked`),
		regexp.MustCompile(`\$\('setSelfTest'\)\.checked\s*=\s*!!c\.self_test`),
		regexp.MustCompile(`\$\('setSelfTest'\)\.addEventListener\('change'`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that behavior is then asserted by nothing", want)
		}
	}
}

// The self-test's own idle threshold is one setting on three surfaces too
// (iss-2610041956214381): config.json's self_test_idle_threshold_sec, the
// loop's SelfTestQuiet, and a field in the Self-test box — filled from the
// key, posted back under it, a blank field posting zero for the default, and
// an edit marking the form touched so the next snapshot does not overwrite it.
func TestThePaneIsWiredToTheSelfTestThreshold(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`self_test_idle_threshold_sec:\s*parseInt\(\$\('setSelfTestIdleThreshold'\)\.value, 10\) \|\| 0`),
		regexp.MustCompile(`\$\('setSelfTestIdleThreshold'\)\.value\s*=\s*c\.self_test_idle_threshold_sec \|\| ''`),
		regexp.MustCompile(`\$\('setSelfTestIdleThreshold'\)\.addEventListener\('input', \(\) => \{ settingsTouched = true; \}\)`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that behavior is then asserted by nothing", want)
		}
	}
}

// The field sits in the Self-test box, not beside the context probe's, and
// spans exactly the server's bounds — the same as the probe's threshold.
func TestTheSelfTestThresholdFieldIsInTheSelfTestBoxWithTheServersBounds(t *testing.T) {
	pane := settingsPane(t, readPanelMarkup(t))
	start := strings.Index(pane, "<legend>Self-test</legend>")
	if start < 0 {
		t.Fatal("the Settings pane has no Self-test box")
	}
	box := pane[start:]
	if end := strings.Index(box, "</fieldset>"); end >= 0 {
		box = box[:end]
	}
	if !strings.Contains(box, `id="setSelfTestIdleThreshold"`) {
		t.Fatal("the Self-test box carries no setSelfTestIdleThreshold field")
	}
	tag := fieldTag(t, box, "setSelfTestIdleThreshold")
	if got := attr(tag, "type"); got != "number" {
		t.Errorf("setSelfTestIdleThreshold is type=%q, want number", got)
	}
	if got, want := attr(tag, "min"), strconv.Itoa(config.MinIdleThresholdSec); got != want {
		t.Errorf("setSelfTestIdleThreshold has min=%q, want the server's %s", got, want)
	}
	if got, want := attr(tag, "max"), strconv.Itoa(config.MaxIdleThresholdSec); got != want {
		t.Errorf("setSelfTestIdleThreshold has max=%q, want the server's %s", got, want)
	}
	if !strings.Contains(box, `id="selfTestIdleThresholdDefault"`) {
		t.Error("the Self-test box carries no span saying what a blank threshold is")
	}
}

// A blank field is the self-test's default, and the field and the sentence
// beside it say which figure that is from the snapshot, never from a copy.
func TestTheSelfTestThresholdDefaultIsFilledFromTheServer(t *testing.T) {
	doc := evalPanelDOM(t, fmt.Sprintf(`renderDefaults({"idle_threshold_sec":%d,"self_test_idle_threshold_sec":%d});`,
		config.DefaultIdleThresholdSec, config.DefaultSelfTestIdleThresholdSec),
		"renderDefaults", "blankIsSentence", "intervalWords")
	if got, _ := doc["setSelfTestIdleThreshold"]["placeholder"].(string); got != strconv.Itoa(config.DefaultSelfTestIdleThresholdSec) {
		t.Errorf("setSelfTestIdleThreshold's placeholder is %q, want the server's default %d",
			got, config.DefaultSelfTestIdleThresholdSec)
	}
	if said, _ := doc["selfTestIdleThresholdDefault"]["textContent"].(string); !strings.Contains(said, "Blank is 4 hours") {
		t.Errorf("the prose beside the self-test's field reads %q, want it to name the server's default", said)
	}
	if got, _ := doc["setIdleThreshold"]["placeholder"].(string); got != strconv.Itoa(config.DefaultIdleThresholdSec) {
		t.Errorf("setIdleThreshold's placeholder is %q, want the probe's own default %d", got, config.DefaultIdleThresholdSec)
	}

	blank := evalPanelDOM(t, `renderDefaults({});`, "renderDefaults", "blankIsSentence", "intervalWords")
	if got, _ := blank["setSelfTestIdleThreshold"]["placeholder"].(string); got != "" {
		t.Errorf("setSelfTestIdleThreshold's placeholder is %q for a snapshot carrying no defaults, want it blank", got)
	}
	if got, _ := blank["selfTestIdleThresholdDefault"]["textContent"].(string); got != "" {
		t.Errorf("the prose beside the self-test's field reads %q for a snapshot carrying no defaults, want nothing", got)
	}
}

// The results reach the tab on the tab's own cadence, from the endpoint that
// serves them, and every figure in a row is drawn from the run it names.
func TestTheStatisticsTabShowsTheSelfTestResults(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`renderSelfTest\(await api\('/api/selftest'\)\)`),
		regexp.MustCompile(`\bselfTestRow\(`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that behavior is then asserted by nothing", want)
		}
	}
	run := `{"kind":"run","model":"org/a","at":1788696000,"outcome":"ok","cold_load":true,"load_ms":12345,` +
		`"tests":[{"name":"pp512","parallel":1,"prompt_tokens_per_sec":812.5},` +
		`{"name":"tg128","parallel":1,"first_token_ms":210,"tokens_per_sec":41.3},` +
		`{"name":"tg128x4","parallel":4,"tokens_per_sec":120.8}]}`
	got := evalPanelValue(t, "selfTestRow("+run+")", "selfTestRow")
	for field, want := range map[string]string{
		"model": "org/a", "outcome": "ok", "load": "12.3 s",
		"promptRead": "812.5 tok/s", "firstToken": "210 ms", "generation": "41.3 tok/s", "underLoad": "120.8 tok/s ×4",
	} {
		if got[field] != want {
			t.Errorf("%s = %v, want %q", field, got[field], want)
		}
	}
	yielded := evalPanelValue(t, `selfTestRow({"model":"org/b","at":0,"outcome":"yielded","cold_load":false,"load_ms":3,"tests":[]})`, "selfTestRow")
	if yielded["outcome"] != "yielded to a request" || yielded["load"] != "already loaded" || yielded["promptRead"] != "" {
		t.Errorf("yielded row = %v", yielded)
	}
}

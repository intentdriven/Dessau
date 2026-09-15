package ui

import (
	"regexp"
	"testing"
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

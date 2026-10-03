package ui

import (
	"regexp"
	"testing"
)

// The switch for the model API's unload is one setting on three surfaces:
// filled from the file's key, posted under it, and an absent key reads as on
// (itd-2610031024247803 criterion 7).
func TestThePaneIsWiredToTheAPIUnloadSwitch(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`api_unload_off:\s*!\$\('setAPIUnload'\)\.checked`),
		regexp.MustCompile(`\$\('setAPIUnload'\)\.checked\s*=\s*!c\.api_unload_off`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that behavior is then asserted by nothing", want)
		}
	}
}

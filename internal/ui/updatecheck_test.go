package ui

import (
	"regexp"
	"testing"
)

// The update check's two settings are one setting each on three surfaces,
// and the pane is held to the file's keys: filled from them, posted under them
// (itd-2610030857275099 criterion 10).
func TestThePaneIsWiredToTheUpdateCheckSettings(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`update_check_enabled:\s*\$\('setUpdateCheck'\)\.checked`),
		regexp.MustCompile(`\$\('setUpdateCheck'\)\.checked\s*=\s*!!c\.update_check_enabled`),
		regexp.MustCompile(`update_check_interval_hours:\s*parseInt\(\$\('setUpdateCheckInterval'\)\.value, 10\) \|\| 0`),
		regexp.MustCompile(`\$\('setUpdateCheckInterval'\)\.value\s*=\s*c\.update_check_interval_hours`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that behavior is then asserted by nothing", want)
		}
	}
}

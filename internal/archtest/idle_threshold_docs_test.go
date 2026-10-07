package archtest_test

import (
	"fmt"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// The context probe's page states the idle threshold's range from the
// server's own bounds, so raising the cap without the page — or the page
// without the cap — is a failure rather than a figure an operator types and
// sees refused (iss-2610041948272032).
func TestTheContextProbePageStatesTheIdleThresholdsRange(t *testing.T) {
	page := readDoc(t, "context-probe.md")
	for _, want := range []string{
		"`idle_threshold_sec`",
		fmt.Sprintf("%d minute", config.MinIdleThresholdSec/60),
		fmt.Sprintf("%d hours", config.MaxIdleThresholdSec/3600),
	} {
		if !containsAll(page, want) {
			t.Errorf("docs/context-probe.md does not say %q, so the idle threshold's range is not "+
				"the one the server enforces", want)
		}
	}
}

// The self-test's pages name its own threshold, its default and its range
// from the server's constants, and no page still says the self-test and the
// context probe share one (iss-2610041956214381).
func TestTheSelfTestPagesStateItsOwnThreshold(t *testing.T) {
	def := fmt.Sprintf("%d hours", config.DefaultSelfTestIdleThresholdSec/3600)
	for page, wants := range map[string][]string{
		"self-test.md": {
			"`self_test_idle_threshold_sec`", "Settings → Self-test", def,
			fmt.Sprintf("%d hours", config.MaxIdleThresholdSec/3600),
		},
		"self-test-reference.md": {"`self_test_idle_threshold_sec`", def},
	} {
		body := readDoc(t, page)
		for _, want := range wants {
			if !containsAll(body, want) {
				t.Errorf("docs/%s does not say %q", page, want)
			}
		}
	}
	for _, page := range []string{"self-test.md", "self-test-reference.md", "context-probe.md"} {
		body := readDoc(t, page)
		for _, stale := range []string{"shared with the context probe", "shares the idle threshold", "share it"} {
			if containsAll(body, stale) {
				t.Errorf("docs/%s still says %q, and the self-test has a threshold of its own", page, stale)
			}
		}
	}
}

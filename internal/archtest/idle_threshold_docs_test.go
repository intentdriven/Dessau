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

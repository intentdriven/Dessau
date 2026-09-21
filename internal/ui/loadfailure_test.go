package ui

import (
	"strings"
	"testing"
)

// The card says when a model's last load failed: the pool's own reason,
// that nothing idle retries it, and the way out — Load or Measure now, the
// two hand retries that lift it (iss-2609211334570516). A pure function a
// test holds, like the lines beside it.
func TestTheCardSaysWhyAModelDidNotLoadAndHowToRetryIt(t *testing.T) {
	line := func(expr string) string {
		return evalPanel(t, expr, "loadFailureText")
	}
	got := line(`loadFailureText({repo_id:"org/m", load_failure:{reason:"could not load: ValueError: Model type glm_ocr not supported.", at:1, runtime:"0.31.3"}})`)
	want := "Did not load the last time it was tried (could not load: ValueError: Model type glm_ocr not supported) — not tried again on its own until the runtime, the memory budget or the served window changes; press Load or Measure now to try it again"
	if got != want {
		t.Errorf("the line reads %q, want %q", got, want)
	}
	if got := line(`loadFailureText({repo_id:"org/m"})`); got != "" {
		t.Errorf("a model with no failure has a line: %q", got)
	}
}

// The renderer builds the card from that line and marks the model with a
// pill, so a failure is visible before the reason is read.
func TestTheCardIsBuiltFromTheLoadFailureLine(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderModels")
	for _, fragment := range []string{
		"const failed = m.state === 'ready' ? loadFailureText(m) : '';",
		`${failed ? ` + "`" + `<div class="info loadfailure">${escapeHtml(failed)}</div>` + "`" + ` : ''}`,
		`if (m.state === 'ready' && m.load_failure) pill += '<span class="pill failed">did not load</span>';`,
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("renderModels no longer contains %s — the card's load-failure line is then asserted by nothing", fragment)
		}
	}
}

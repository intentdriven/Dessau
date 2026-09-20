package ui

import (
	"strings"
	"testing"
)

// The card says whether the model calls tools, beside the measured context,
// in one of three states: yes, no, or not measured — which covers both a
// model not yet asked and one whose verdict was taken under another runtime.
// A pure function a test holds, like the measurement line beside it.
func TestTheCardSaysWhetherTheModelCallsTools(t *testing.T) {
	line := func(expr string) string {
		return evalPanel(t, expr, "toolCallText")
	}
	cases := []struct {
		name, expr, want string
	}{
		{"can", `toolCallText({repo_id:"org/m", tool_calling:{can:true, at:1, runtime:"0.31.3"}})`, "Tool calls: yes"},
		{"cannot", `toolCallText({repo_id:"org/m", tool_calling:{can:false, at:1, runtime:"0.31.3"}})`, "Tool calls: no"},
		{"not asked", `toolCallText({repo_id:"org/m"})`, "Tool calls: not measured"},
		{"stale", `toolCallText({repo_id:"org/m", tool_calling:{can:true, at:1, runtime:"0.30.0", stale:"runtime"}})`, "Tool calls: not measured"},
	}
	for _, tt := range cases {
		if got := line(tt.expr); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}

// toolCallText is only worth testing while the card is built from it: the
// renderer computes the line for a ready model and places it in the card's
// markup beside the measurement line.
func TestTheCardIsBuiltFromTheToolCallLine(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderModels")
	for _, fragment := range []string{
		"const tools = m.state === 'ready' ? toolCallText(m) : '';",
		`${tools ? `+"`"+`<div class="info toolcalls">${escapeHtml(tools)}</div>`+"`"+` : ''}`,
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("renderModels no longer contains %s — the card's tool-call line is then asserted by nothing", fragment)
		}
	}
}

package ui

import (
	"strings"
	"testing"
)

// The card's residency pill says what the memory is doing, not only that
// it is spoken for: a model still loading says so, and one an idle job is
// holding names the job — so the person who finds every client refused can
// see that the server's own work holds the memory, and press Unload
// (iss-2609211334576018). A pure function a test holds.
func TestTheResidencyPillNamesTheJobHoldingTheModel(t *testing.T) {
	label := func(expr string) string {
		return evalPanel(t, expr, "residencyLabel")
	}
	cases := []struct {
		name, expr, want string
	}{
		{"loaded", `residencyLabel({repo_id:"org/m", state:"loaded"}, {})`, "loaded"},
		{"loading", `residencyLabel({repo_id:"org/m", state:"loading"}, {})`, "loading"},
		{"loading for the probe", `residencyLabel({repo_id:"org/m", state:"loading"}, {job:"context-probe", model:"Org/M"})`, "loading for the context probe"},
		{"held by the probe", `residencyLabel({repo_id:"org/m", state:"loaded"}, {job:"context-probe", model:"org/m"})`, "held by the context probe"},
		{"held by the self-test", `residencyLabel({repo_id:"org/m", state:"loaded"}, {job:"self-test", model:"org/m"})`, "held by the self-test"},
		{"another model's job", `residencyLabel({repo_id:"org/m", state:"loaded"}, {job:"self-test", model:"org/other"})`, "loaded"},
		{"not resident", `residencyLabel(undefined, {job:"self-test", model:"org/m"})`, ""},
	}
	for _, tt := range cases {
		if got := label(tt.expr); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}

// The renderer draws the pill from that label.
func TestTheCardIsBuiltFromTheResidencyLabel(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderModels")
	fragment := "`<span class=\"pill loaded\">${escapeHtml(residencyLabel(resident.get(m.repo_id), state.idle_jobs))}</span>`"
	if !strings.Contains(body, fragment) {
		t.Errorf("renderModels no longer contains %s — the card's residency pill is then asserted by nothing", fragment)
	}
}

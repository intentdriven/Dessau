package ui

import (
	"strings"
	"testing"
)

// What the bridge card says, in the four states the snapshot can report.
//
// The eleventh acceptance criterion of itd-2609180959397172 is that a dropped
// session resumes by itself and the panel shows when it last connected — so
// the moment belongs in the states where the bridge is NOT connected, which is
// where the card used to drop it (iss-2609190242334438). The moment itself is
// the viewer's own locale and time zone, so the assertion replaces it with a
// marker computed in the same runtime rather than pinning a rendering.
func TestTheBridgeCardShowsWhenItLastConnected(t *testing.T) {
	// The snapshot's own seconds, as the panel is given them.
	const marker = `.replace(new Date(1700000000 * 1000).toLocaleString(), '<moment>')`
	for _, c := range []struct {
		name string
		snap string
		want string
	}{
		{"connected", `{state:'connected', since:1700000000}`, "Connected since <moment>."},
		{"connected, no moment", `{state:'connected'}`, "Connected."},
		// The path the criterion is about: the socket dropped and the bridge
		// is re-opening. The moment is what says the bridge ever worked.
		{"reconnecting", `{state:'connecting', since:1700000000}`, "Last connected at <moment>; reconnecting…"},
		{"connecting for the first time", `{state:'connecting'}`, "Connecting to Discord…"},
		{"stopped", `{state:'stopped', reason:'Discord refused the bot token', since:1700000000}`,
			"Stopped: Discord refused the bot token. Last connected at <moment>."},
		{"stopped having never connected", `{state:'stopped', reason:'no bot token'}`, "Stopped: no bot token."},
		{"off", `{state:'off'}`, "The bridge is off."},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := evalPanel(t, "bridgeStateText("+c.snap+")"+marker, "bridgeStateText")
			if got != c.want {
				t.Errorf("bridgeStateText(%s) = %q, want %q", c.snap, got, c.want)
			}
		})
	}
}

// And the card is drawn from it: a sentence computed and then not shown is a
// card that says nothing.
func TestTheBridgeCardIsDrawnFromThatSentence(t *testing.T) {
	body := extractFunction(t, readPanelSource(t), "renderBridgeState")
	if !strings.Contains(body, "bridgeStateText(state.bridge || {})") {
		t.Error("renderBridgeState no longer renders bridgeStateText — the sentence above is then asserted by nothing")
	}
}

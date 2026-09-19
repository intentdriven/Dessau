package archtest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The switch that sends a conversation off this Mac cannot be drawn without
// the sentence that says so, in the same pane.
//
// adr-2609181004167097 condition 5: "The Settings pane that takes the token
// states, beside the switch, that messages to the bot and the model's answers
// pass through the platform and are kept under its terms... The switch cannot
// be rendered without that sentence in the same pane; held by a test." This is
// that test.
//
// It is scoped to the fieldset, not to the page, and that is the whole force
// of it. A search over the whole of index.html would pass a panel whose
// sentence had drifted three screens away from the control it describes, which
// is the same as not having it: what makes an opt-in informed is that the
// person reading the switch reads the price beside it.
func TestTheBridgeSwitchCarriesTheEgressSentence(t *testing.T) {
	block := bridgeFieldset(t)

	// The claim, in the words the ADR and the intent use. Each phrase is
	// checked on its own so a failure says which half went.
	for _, phrase := range []string{
		"messages to the bot",
		"answers pass through Discord",
		"kept under Discord",
	} {
		if !strings.Contains(strings.ToLower(block), strings.ToLower(phrase)) {
			t.Errorf("the Discord bridge's Settings block does not say %q — the switch may not be "+
				"drawn without the sentence saying what leaves this Mac (adr-2609181004167097 "+
				"condition 5)", phrase)
		}
	}
	// And the other three promises the same block makes: outbound only, off by
	// default, and off closes the connection (conditions 1 and 2).
	for _, phrase := range []string{
		"Off unless you turn it on",
		"connects out",
		"closes the connection",
	} {
		if !strings.Contains(block, phrase) {
			t.Errorf("the Discord bridge's Settings block does not say %q", phrase)
		}
	}
	// The token's box is in the same block as the switch: a credential whose
	// field sits somewhere else is one an operator pastes without having read
	// what it buys.
	if !strings.Contains(block, `id="setDiscordToken"`) {
		t.Error("the token field is not in the same block as the switch")
	}
	// And the state, which is what the panel shows instead of refusing a save
	// over a token Discord will not accept.
	if !strings.Contains(block, `id="discordState"`) {
		t.Error("the bridge's state is not shown beside its switch")
	}
}

// The docs page says the same thing the pane says. The ADR's condition names
// both surfaces, and a page that omits it would be the one an operator reads
// first.
func TestTheBridgeDocsPageSaysWhatLeavesTheMac(t *testing.T) {
	page := readDoc(t, "discord-bridge.md")
	for _, phrase := range []string{
		"pass through Discord",
		"Discord's terms",
		"Public Bot",
		"/model",
		"/reset",
	} {
		if !strings.Contains(page, phrase) {
			t.Errorf("docs/discord-bridge.md does not say %q", phrase)
		}
	}
	// The pane points at the page, so the page has to be where the pane says.
	markup := readPanelFile(t, "index.html")
	if !strings.Contains(markup, "docs/discord-bridge.md") {
		t.Error("the Settings pane does not link the Discord bridge page")
	}
}

// bridgeFieldset is the one Settings block that carries the bridge's switch.
func bridgeFieldset(t *testing.T) string {
	t.Helper()
	markup := readPanelFile(t, "index.html")
	const control = `id="setDiscordBridge"`
	at := strings.Index(markup, control)
	if at < 0 {
		t.Fatalf("the Settings pane carries no %s, so the bridge has no control at all "+
			"(AGENTS.md, \"Three surfaces\")", control)
	}
	start := strings.LastIndex(markup[:at], "<fieldset")
	end := strings.Index(markup[at:], "</fieldset>")
	if start < 0 || end < 0 {
		t.Fatal("the bridge's switch is not inside a Settings block, so nothing holds a sentence beside it")
	}
	// Whitespace collapsed: the block is hand-wrapped HTML, so a sentence the
	// reader sees as one line is several in the source, and a search for it
	// would fail over where the line happened to break.
	return strings.Join(strings.Fields(markup[start:at+end]), " ")
}

// readPanelFile reads one of the panel's static files.
func readPanelFile(t *testing.T, name string) string {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(repoRoot, "internal", "ui", "static", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

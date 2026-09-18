package archtest_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestChatClientComposerHasMessagesForm holds the composer to the form of
// Messages' (iss-2609181045444147): a capsule text field and a round, filled
// send button — both standard controls given standard shapes, never a drawn
// background, so the no-styling rule still holds.
func TestChatClientComposerHasMessagesForm(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	start := regexp.MustCompile(`private var composer: some View \{`).FindStringIndex(src)
	if start == nil {
		t.Fatal("client/GropiusChat/GropiusChat.swift declares no composer view")
	}
	end := regexp.MustCompile(`\n    private func send\(\)`).FindStringIndex(src[start[0]:])
	if end == nil {
		t.Fatal("the composer view is not followed by send()")
	}
	composer := src[start[0] : start[0]+end[0]]
	for _, want := range []string{
		`.textInputBorderShape(.capsule)`,
		`.buttonStyle(.borderedProminent)`,
		`.buttonBorderShape(.circle)`,
	} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(composer) {
			t.Errorf("the composer does not carry %s; the field is a capsule and the send button a filled circle, as Messages' are", want)
		}
	}
}

// TestChatClientThoughtsToggleOnAClickAnywhere holds the Thoughts row's
// behaviour: a click anywhere in the row, and in the expanded thinking text,
// toggles it — not only the disclosure triangle — so the thinking can be
// hidden while it is being read.
func TestChatClientThoughtsToggleOnAClickAnywhere(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	start := regexp.MustCompile(`private var reasoningDisclosure: some View \{`).FindStringIndex(src)
	if start == nil {
		t.Fatal("client/GropiusChat/GropiusChat.swift declares no reasoningDisclosure view")
	}
	end := regexp.MustCompile(`\n\}\n`).FindStringIndex(src[start[0]:])
	if end == nil {
		t.Fatal("the reasoningDisclosure view does not end")
	}
	block := src[start[0] : start[0]+end[0]]
	if strings.Count(block, ".onTapGesture") < 2 {
		t.Error("the Thoughts row does not toggle on a click in its label and in its expanded text; only the disclosure triangle toggles it")
	}
	if !strings.Contains(block, ".contentShape(Rectangle())") {
		t.Error("the Thoughts row's label is not made clickable across its whole width")
	}
}

// TestChatClientDrawsMessagesAsBubbles holds the transcript's bubbles
// (iss-2609181055156852): the person's in the system's accent colour and the
// model's in grey by default, both changeable in Settings. The bubble is the
// no-styling rule's second named exception, kept in its own file so the
// exception is one file wide.
func TestChatClientDrawsMessagesAsBubbles(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)
	bubbles, ok := all["Bubbles.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/Bubbles.swift is missing; the transcript's bubbles have no home")
	}
	for _, want := range []string{`Color.accentColor`, `Color.gray`, `@AppStorage("bubbleColorUser")`, `@AppStorage("bubbleColorModel")`} {
		if !strings.Contains(bubbles, want) {
			t.Errorf("client/GropiusChat/Bubbles.swift does not carry %s", want)
		}
	}
	if !strings.Contains(bubbles, "ColorPicker(") || strings.Count(all["GropiusChat.swift"], "BubbleColorRow(") < 2 {
		t.Error("Settings does not offer a colour picker for each bubble")
	}
	if !strings.Contains(all["GropiusChat.swift"], ".bubble(") {
		t.Error("MessageRow does not draw its messages with the bubble modifier")
	}
}

// TestChatClientAsksForTheKeyWhereTheServerIsPicked holds the picker's key
// prompt (iss-2609181058132356): a server that needs an API key is asked for
// it in a sheet right there, which says the key is kept in the Keychain,
// bound to that server, asked once, and changeable in Settings.
func TestChatClientAsksForTheKeyWhereTheServerIsPicked(t *testing.T) {
	root := repoRootDir(t)
	picker := clientSources(t, root)["Picker.swift"]
	for _, want := range []string{`.sheet(item: $askingKeyFor`, `model.saveAPIKey(`, `Keychain`, `only once`, `Settings`, `SecureField(`} {
		if !strings.Contains(picker, want) {
			t.Errorf("client/GropiusChat/Picker.swift does not carry %q; a server that needs a key is not asked for it where it is picked", want)
		}
	}
	if !strings.Contains(clientSources(t, root)["GropiusChat.swift"], "needsAPIKey = true") {
		t.Error("AppModel.connect does not record that the server asked for a key")
	}
}

// TestChatClientTextSizeScalesTheWholeWindow holds the text-size setting
// (itd-2609181100459593): five steps that are the system's own Dynamic Type
// sizes, applied at the window's root and at Settings' root.
func TestChatClientTextSizeScalesTheWholeWindow(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	if !regexp.MustCompile(`enum TextSize: String, CaseIterable \{\s*case smaller, standard, larger, extraLarge, huge\s*\}?`).MatchString(src) {
		t.Error("client/GropiusChat/GropiusChat.swift declares no TextSize with the five steps")
	}
	if n := strings.Count(src, ".dynamicTypeSize("); n < 2 {
		t.Errorf("the Dynamic Type size is applied at %d root(s); the window and Settings each need it", n)
	}
	if !strings.Contains(src, `@AppStorage("textSize")`) {
		t.Error("the text size is not stored")
	}
	if !strings.Contains(src, `Picker("Text size"`) {
		t.Error("Settings offers no text-size picker")
	}
}

// TestChatClientAppearanceFollowsOneSetting holds the appearance setting
// (itd-2609181102147562): Light, Dark or System, applied as the preferred
// colour scheme at the window's root and at Settings' root; System is the
// absence of a preference.
func TestChatClientAppearanceFollowsOneSetting(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	if !regexp.MustCompile(`enum Appearance: String, CaseIterable \{\s*case system, light, dark`).MatchString(src) {
		t.Error("client/GropiusChat/GropiusChat.swift declares no Appearance with the three choices")
	}
	if n := strings.Count(src, ".preferredColorScheme("); n < 2 {
		t.Errorf("the preferred colour scheme is applied at %d root(s); the window and Settings each need it", n)
	}
	if !strings.Contains(src, `@AppStorage("appearance")`) || !strings.Contains(src, `Picker("Appearance"`) {
		t.Error("the appearance is not stored, or Settings offers no picker for it")
	}
}

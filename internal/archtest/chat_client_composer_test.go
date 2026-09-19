package archtest_test

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestChatClientComposerHasMessagesForm holds the composer to the form of
// Messages' (iss-2609181045444147): a capsule text field and a round, filled
// send button. The button is a standard control given the system's circle;
// the field's capsule is drawn in Composer.swift, the no-styling rule's
// third named exception (iss-2609190004092322).
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
		`.composerFieldCapsule()`,
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
	if !strings.Contains(src, "enum TextSize: String, CaseIterable {") {
		t.Fatal("client/GropiusChat/GropiusChat.swift declares no TextSize enum over all of its cases")
	}
	steps := []string{"smaller", "standard", "larger", "extraLarge", "huge"}
	if got := enumCases(t, src, "TextSize"); !slices.Equal(got, steps) {
		t.Errorf("TextSize declares the cases %v; the setting is the five steps %v, and nothing else", got, steps)
	}
	window, settings := sceneRoots(t, src)
	if !strings.Contains(window, ".dynamicTypeSize(") {
		t.Error("the window's root does not apply the Dynamic Type size; the whole window follows one setting")
	}
	if !strings.Contains(settings, ".dynamicTypeSize(") {
		t.Error("the Settings scene's root does not apply the Dynamic Type size; Settings is its own scene and inherits nothing")
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
	if !strings.Contains(src, "enum Appearance: String, CaseIterable {") {
		t.Fatal("client/GropiusChat/GropiusChat.swift declares no Appearance enum over all of its cases")
	}
	choices := []string{"system", "light", "dark"}
	if got := enumCases(t, src, "Appearance"); !slices.Equal(got, choices) {
		t.Errorf("Appearance declares the cases %v; the choice is %v, and nothing else", got, choices)
	}
	window, settings := sceneRoots(t, src)
	if !strings.Contains(window, ".preferredColorScheme(") {
		t.Error("the window's root does not apply the preferred colour scheme; the whole window follows one setting")
	}
	if !strings.Contains(settings, ".preferredColorScheme(") {
		t.Error("the Settings scene's root does not apply the preferred colour scheme; Settings is its own scene and inherits nothing")
	}
	if !strings.Contains(src, `@AppStorage("appearance")`) || !strings.Contains(src, `Picker("Appearance"`) {
		t.Error("the appearance is not stored, or Settings offers no picker for it")
	}
}

// TestChatClientThoughtsRenderMarkdown holds itd-2609181104497297: the
// Thoughts row draws its text through the same markdown blocks the reply
// uses, kept in the row's state.
func TestChatClientThoughtsRenderMarkdown(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	start := regexp.MustCompile(`private var reasoningDisclosure: some View \{`).FindStringIndex(src)
	if start == nil {
		t.Fatal("no reasoningDisclosure view")
	}
	block := src[start[0]:]
	if end := regexp.MustCompile(`\n    (private )?(var|func) `).FindStringIndex(block[1:]); end != nil {
		block = block[:end[0]+1]
	}
	if !strings.Contains(block, "reasoningBlocks") {
		t.Error("the Thoughts row does not draw its text through markdown blocks")
	}
	if !strings.Contains(src, "MarkdownBlocks.parse(displayReasoning)") {
		t.Error("the reasoning is never parsed as markdown")
	}
}

// TestChatClientSidebarShowsCards holds itd-2609181104490133: each row is a
// card with an icon, the title, the date and a summary of exchanges and
// words, and the list takes the sidebar style so the selection is the
// system's highlight.
func TestChatClientSidebarShowsCards(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	for _, want := range []string{`.listStyle(.sidebar)`, `ConversationCard(conversation:`} {
		if !strings.Contains(src, want) {
			t.Errorf("the sidebar lacks %q", want)
		}
	}
	// The promise is about what the CARD draws, so the scan is the card's
	// own body: a substring found anywhere in the file says nothing about
	// the view the sidebar puts in each row (iss-2609181213198794).
	card := swiftBlock(t, src, "struct ConversationCard: View {")
	for _, want := range []struct{ fragment, promise string }{
		{`Image(systemName: icon)`, "the icon for who answered last"},
		{`Text(title)`, "the conversation's title"},
		{`Text(summary)`, "the summary of exchanges and words"},
	} {
		if !strings.Contains(card, want.fragment) {
			t.Errorf("ConversationCard does not draw %s; its body carries no %q", want.promise, want.fragment)
		}
	}
	// The date is the one the conversation started, drawn as a day, a month
	// and a year. Matching `createdAt` against the whole file would pass on
	// the model's own field while the card drew some other date, or none.
	if !regexp.MustCompile(`Text\(conversation\.createdAt, format: \.dateTime\.day\(\)\.month\(\)\.year\(\)\)`).MatchString(card) {
		t.Error("ConversationCard does not draw conversation.createdAt as a day, month and year; the date a row shows is the date its chat started")
	}
	for _, want := range []string{"exchange", "word"} {
		if !strings.Contains(card, want) {
			t.Errorf("ConversationCard's summary counts no %ss", want)
		}
	}
}

// TestChatClientSidebarIsSearchable holds itd-2609181104498312: a
// searchable field on the sidebar filters conversations by title and
// message text.
func TestChatClientSidebarIsSearchable(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	if !strings.Contains(src, `.searchable(text: $query`) {
		t.Error("the sidebar carries no searchable field")
	}
	// What the filter reads; HOW it matches is
	// TestChatClientSidebarSearchesEveryWord's, next to the check that runs it.
	if !regexp.MustCompile(`title: c\.title`).MatchString(src) ||
		!regexp.MustCompile(`messages: c\.messages\.map\(\\\.text\)`).MatchString(src) {
		t.Error("the search does not filter on both the title and the messages")
	}
}

// sceneRoots cuts the client's App body into its two scene bodies: the
// content the WindowGroup is given and the content the Settings scene is
// given, each read to ITS own closing brace. A setting read "at each scene's
// root" is a promise about those two places (iss-2609181200251755,
// iss-2609181124291823); counting a modifier anywhere in the file, or reading
// the Settings scene as everything that follows it, would let a modifier on
// some inner view further down stand in for a root that lost its own.
func sceneRoots(t *testing.T, src string) (window, settings string) {
	t.Helper()
	scene := swiftBlock(t, src, "var body: some Scene {")
	return swiftBlock(t, scene, "WindowGroup("), swiftBlock(t, scene, "Settings {")
}

// swiftBlock is the brace-delimited body that opens at the first "{" at or
// after marker, matched by the package's own `balanced` scan so the block
// ends at the brace that closes it rather than at the first line that looks
// like one. The marker must name one place: a second occurrence is an
// architectural change this scan should be reread for, not silently pick the
// first of.
func swiftBlock(t *testing.T, src, marker string) string {
	t.Helper()
	switch n := strings.Count(src, marker); {
	case n == 0:
		t.Fatalf("client/GropiusChat/GropiusChat.swift carries no %q", marker)
	case n > 1:
		t.Fatalf("%q appears %d times; this scan reads one block and cannot say which", marker, n)
	}
	at := strings.Index(src, marker)
	open := strings.Index(src[at:], "{")
	if open < 0 {
		t.Fatalf("%q opens no block", marker)
	}
	body, end := balanced(src, at+open, '{', '}')
	if end == len(src) && !strings.HasSuffix(src, "}") {
		t.Fatalf("the block opened by %q is never closed", marker)
	}
	return body
}

// enumCases is the case names a Swift enum declares, read from the whole of
// its body rather than from its first case line: a case added on a following
// line belongs to the set just as much, and a regexp anchored to the first
// line would not see it (iss-2609181124291823). The `case .x:` lines of the
// switches inside the enum's computed properties carry a leading dot and end
// in a colon, so they are not read as declarations.
func enumCases(t *testing.T, src, name string) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(swiftBlock(t, src, "enum "+name+": "), "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "case ")
		if !ok || strings.HasPrefix(rest, ".") || strings.Contains(rest, ":") {
			continue
		}
		for _, one := range strings.Split(rest, ",") {
			if one, _, _ = strings.Cut(one, "="); strings.TrimSpace(one) != "" {
				out = append(out, strings.TrimSpace(one))
			}
		}
	}
	return out
}

// TestChatClientComposerFieldHasMessagesProportions holds
// iss-2609190004092322: the composer's field is a capsule of the proportions
// Messages and WhatsApp give theirs — tall enough to read as a field rather
// than a slot, with its text inset from the capsule's curve instead of
// starting against it. Both numbers are named and both are scaled metrics, so
// the text-size setting grows the field and not only the glyphs inside it.
//
// SwiftUI's own bordered capsule offers no way to inset its text, so the
// capsule is drawn in client/GropiusChat/Composer.swift, which is why that
// file is a named exception to the no-styling rule.
func TestChatClientComposerFieldHasMessagesProportions(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)
	src, ok := all["Composer.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/Composer.swift is missing; the composer's metrics have no home")
	}
	// Messages' field is about 34 to 36 points tall at the standard text
	// size, with roughly 12 points before the first glyph.
	for _, m := range []struct {
		name string
		low  float64
		high float64
	}{
		{"fieldMinHeight", 34, 36},
		{"textInset", 12, 16},
	} {
		found := regexp.MustCompile(`static let ` + m.name + `: CGFloat = ([0-9.]+)`).FindStringSubmatch(src)
		if found == nil {
			t.Errorf("Composer.swift declares no ComposerMetrics.%s; the field's shape is an unnamed number", m.name)
			continue
		}
		got, err := strconv.ParseFloat(found[1], 64)
		if err != nil {
			t.Errorf("ComposerMetrics.%s is not a number: %v", m.name, err)
			continue
		}
		if got < m.low || got > m.high {
			t.Errorf("ComposerMetrics.%s is %v; Messages' composer sits between %v and %v", m.name, got, m.low, m.high)
		}
	}
	// A constant the field reads at a fixed size would leave the field the
	// same height while the text inside it grew.
	if strings.Count(src, "@ScaledMetric(relativeTo: .body)") < 2 {
		t.Error("Composer.swift does not read its metrics as scaled metrics; the text-size setting would grow the text and not the field")
	}
	for _, want := range []string{
		`.frame(minHeight: minHeight)`,
		`.padding(.horizontal, horizontal)`,
		`.background(.quaternary, in: .capsule)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("Composer.swift does not carry %s; the field is not a capsule of a pinned height with its text inset", want)
		}
	}

	// The inset has to reach the text the person actually sees: the field
	// carrying the placeholder is the view the capsule is applied to, not a
	// wrapper beside it.
	chat := all["GropiusChat.swift"]
	field := regexp.MustCompile(`TextField\("Message…", text: \$draft, axis: \.vertical\)\n(\s+\.[^\n]*\n)*\s+\.composerFieldCapsule\(\)`)
	if !field.MatchString(chat) {
		t.Error("the placeholder field does not carry .composerFieldCapsule(); the inset applies to something other than the text the person reads")
	}
	// The send button keeps its circle, sized to the field so it is centred
	// beside one line and stays at the foot of a field that has grown.
	if !strings.Contains(chat, ".composerButtonCircle()") || !strings.Contains(src, "struct ComposerButtonCircle") {
		t.Error("the send button's circle is not matched to the field's height")
	}
}

// TestChatClientComposerShowsTheSystemsFocusRing holds iss-2609190034161350:
// the drawn capsule says when the field has keyboard focus. A plain field
// draws no focus effect, so the capsule that replaced the bordered field has
// to show the indication itself — in the system's own focus colour and at the
// thickness the system strokes, never a ring of the client's own invention,
// and only while the environment allows a focus effect, which is how the
// system's focus-ring preference reaches a view that draws its own.
func TestChatClientComposerShowsTheSystemsFocusRing(t *testing.T) {
	root := repoRootDir(t)
	src, ok := clientSources(t, root)["Composer.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/Composer.swift is missing; the composer's focus ring has no home")
	}
	for _, want := range []struct{ fragment, why string }{
		{"@FocusState", "the capsule holds no focus state, so it cannot know the field has keyboard focus"},
		{".focused($", "the field is never bound to the focus state"},
		{`@Environment(\.isFocusEffectEnabled)`, "the ring ignores the system's focus-effect preference"},
		{"keyboardFocusIndicatorColor", "the ring is not drawn in the system's focus colour"},
		{"#if os(macOS)", "the macOS focus colour is not behind a platform guard, so the iPad build has nowhere to differ"},
		{".strokeBorder(", "nothing strokes a ring around the capsule"},
	} {
		if !strings.Contains(src, want.fragment) {
			t.Errorf("Composer.swift does not carry %s: %s", want.fragment, want.why)
		}
	}
	// The thickness is named, and it is the one the system strokes on a
	// bordered field rather than a hairline or a halo.
	found := regexp.MustCompile(`static let focusRingWidth: CGFloat = ([0-9.]+)`).FindStringSubmatch(src)
	if found == nil {
		t.Fatal("Composer.swift declares no ComposerMetrics.focusRingWidth; the ring's thickness is an unnamed number")
	}
	width, err := strconv.ParseFloat(found[1], 64)
	if err != nil {
		t.Fatalf("ComposerMetrics.focusRingWidth is not a number: %v", err)
	}
	if width < 2 || width > 4 {
		t.Errorf("ComposerMetrics.focusRingWidth is %v; the system strokes a ring between 2 and 4 points on a bordered field", width)
	}
}

package archtest_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The native macOS 27 client (itd-2609151701196720) and the intents that build
// on it promise things about the client's SOURCE: no styling of its own, an
// answerer that touches no network, controls that keep the system's text
// intelligence, a browse that runs only while the picker is shown, a key that
// goes to one host, a build that writes the App Intents metadata or fails, one
// word list for the effects, and one set of parser options for markdown. Each
// is a promise a later edit could quietly undo, so each is read here.

// clientSources returns every Swift file of the client, keyed by its base name.
func clientSources(t *testing.T, root string) map[string]string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "client", "DessauChat", "*.swift"))
	if err != nil || len(matches) == 0 {
		t.Fatal("client/DessauChat/ holds no Swift file")
	}
	out := map[string]string{}
	for _, m := range matches {
		raw, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(m)] = string(raw)
	}
	return out
}

// TestChatClientCarriesNoStylingOfItsOwn holds the trunk's first criterion:
// every control is a standard control drawn by the system. The list names the
// modifiers a styled control reaches for. One file is exempt, by name and
// with its reason: Effects.swift is a TextRenderer, which is client-owned
// drawing by definition and is the text-effects intent's whole mechanism.
func TestChatClientCarriesNoStylingOfItsOwn(t *testing.T) {
	root := repoRootDir(t)
	styling := []string{
		".buttonStyle(.glass", ".background(", ".overlay(", ".shadow(",
		".textFieldStyle(", "cornerRadius", ".clipShape(", "Color(nsColor:",
	}
	exempt := map[string]string{
		"Effects.swift":  "the text-effects renderer draws glyphs itself; that is its purpose",
		"Bubbles.swift":  "the transcript's speech bubbles are a filled shape by definition (iss-2609181055156852), drawn in the system's colours",
		"Composer.swift": "SwiftUI's bordered capsule offers no way to inset a field's text (iss-2609190004092322), so the composer's capsule is drawn in the system's own material, one file wide",
	}
	for name, src := range clientSources(t, root) {
		if _, ok := exempt[name]; ok {
			continue
		}
		for _, s := range styling {
			if strings.Contains(src, s) {
				t.Errorf("client/DessauChat/%s uses %q; the client carries no styling of its own — "+
					"a standard control draws the system's design by itself", name, s)
			}
		}
	}
}

// TestChatClientBuiltInBackendTouchesNoNetwork holds the built-in default's
// headline: the Mac's own model answers with nothing sent anywhere. The type
// that talks to the framework builds no request and imports no networking.
func TestChatClientBuiltInBackendTouchesNoNetwork(t *testing.T) {
	root := repoRootDir(t)
	src, ok := clientSources(t, root)["Backends.swift"]
	if !ok {
		t.Fatal("client/DessauChat/Backends.swift is missing; the answerer seam has no home")
	}
	if strings.Contains(src, "import Network") {
		t.Error("client/DessauChat/Backends.swift imports Network; the built-in answerer must touch no network")
	}
	start := strings.Index(src, "struct BuiltInBackend")
	end := strings.Index(src, "struct ServerBackend")
	if start < 0 || end < 0 || end < start {
		t.Fatal("client/DessauChat/Backends.swift does not declare BuiltInBackend before ServerBackend")
	}
	body := src[start:end]
	for _, s := range []string{"URLSession", "URLRequest", "http"} {
		if strings.Contains(body, s) {
			t.Errorf("BuiltInBackend mentions %q; the Mac's own model answers with nothing sent anywhere", s)
		}
	}
}

// TestChatClientKeepsWritingTools holds the text-intelligence intent: the
// composer is a standard TextField and nothing suppresses Writing Tools.
func TestChatClientKeepsWritingTools(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)
	for name, src := range all {
		for _, s := range []string{"writingToolsBehavior(.disabled)", "NSViewRepresentable", "NSTextView"} {
			if strings.Contains(src, s) {
				t.Errorf("client/DessauChat/%s uses %q; the composer is a standard TextField so the system's Writing Tools reach it", name, s)
			}
		}
	}
	if !regexp.MustCompile(`TextField\("Message…", text: \$draft, axis: \.vertical\)`).MatchString(all["DessauChat.swift"]) {
		t.Error("client/DessauChat/DessauChat.swift has no multi-line TextField composer")
	}
}

// TestChatClientBrowsesOnlyWhileThePickerIsShown holds the offer intent's
// scope condition: the Bonjour browse starts when the picker appears and stops
// when it disappears, so the Local Network prompt comes the first time a
// person looks for a server, not at launch.
func TestChatClientBrowsesOnlyWhileThePickerIsShown(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)
	picker, ok := all["Picker.swift"]
	if !ok {
		t.Fatal("client/DessauChat/Picker.swift is missing")
	}
	onAppear := regexp.MustCompile(`\.onAppear\s*\{[^}]*browser\.start\(\)`)
	onDisappear := regexp.MustCompile(`\.onDisappear\s*\{[^}]*browser\.stop\(\)`)
	if !onAppear.MatchString(picker) {
		t.Error("client/DessauChat/Picker.swift does not start the browse in .onAppear")
	}
	if !onDisappear.MatchString(picker) {
		t.Error("client/DessauChat/Picker.swift does not stop the browse in .onDisappear")
	}
	for name, src := range all {
		if name != "Picker.swift" && strings.Contains(src, "browser.start()") {
			t.Errorf("client/DessauChat/%s starts a browse; only the picker may", name)
		}
	}
}

// TestChatClientSendsTheKeyToOneHostOnly holds the offer intent's promise
// about the API key: it is attached only to the host it was entered for.
func TestChatClientSendsTheKeyToOneHostOnly(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["DessauChat.swift"]
	if !regexp.MustCompile(`if !apiKey\.isEmpty, origin == apiKeyHost \{`).MatchString(src) {
		t.Error("client/DessauChat/DessauChat.swift attaches the bearer token without comparing the request's origin to apiKeyHost")
	}
	// The Settings pane must not re-bind the stored key on its initial load:
	// the change handler ignores a value equal to the stored key.
	if !regexp.MustCompile(`guard new != model\.apiKey else \{ return \}`).MatchString(src) {
		t.Error("client/DessauChat/DessauChat.swift saves the API key on every change of the field, including the initial load, which re-binds it to the current server")
	}
}

// TestChatClientBuildWritesTheAppIntentsMetadata holds the Shortcuts intent's
// loud stage: the build script runs the toolchain's metadata processor and
// fails when nothing was written, rather than shipping a bundle whose actions
// Shortcuts will never list.
func TestChatClientBuildWritesTheAppIntentsMetadata(t *testing.T) {
	root := repoRootDir(t)
	raw := readRepoFile(t, root, filepath.Join("client", "build.sh"))
	for _, want := range []string{
		"appintentsmetadataprocessor",
		"-emit-const-values",
		`Metadata.appintents/extract.actionsdata`,
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("client/build.sh does not carry %q; the App Intents metadata step is missing or silent", want)
		}
	}
	if !regexp.MustCompile(`(?m)^\[ -f "\$RES/Metadata\.appintents/extract\.actionsdata" \] \|\|`).MatchString(raw) {
		t.Error("client/build.sh does not fail when the metadata was not written")
	}
	if strings.Contains(raw, "x86_64") || strings.Contains(raw, "lipo") {
		t.Error("client/build.sh still builds an x86_64 slice; macOS 27 runs on no Intel Mac")
	}
}

// TestChatClientEffectWordsLiveInOnePlace holds the text-effects intent: the
// words are one constant, matched whole and case-insensitively, and the
// Settings page names them from that same constant.
func TestChatClientEffectWordsLiveInOnePlace(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)
	effects, ok := all["Effects.swift"]
	if !ok {
		t.Fatal("client/DessauChat/Effects.swift is missing")
	}
	list := regexp.MustCompile(`static let list: \[\(word: String, kind: EffectKind\)\] = \[([^\]]*)\]`).FindStringSubmatch(effects)
	if list == nil {
		t.Fatal("client/DessauChat/Effects.swift declares no EffectWords.list")
	}
	words := regexp.MustCompile(`\("([^"]+)", \.`).FindAllStringSubmatch(list[1], -1)
	if len(words) < 3 || len(words) > 8 {
		t.Errorf("EffectWords.list has %d words; the list is small and fixed", len(words))
	}
	if !strings.Contains(effects, `options: [.caseInsensitive]`) {
		t.Error("EffectWords.matches is not case-insensitive")
	}
	if !strings.Contains(all["DessauChat.swift"], "EffectWords.settingsSentence") {
		t.Error("the Settings page does not name the words from EffectWords")
	}
	for name, src := range all {
		if name != "Effects.swift" && strings.Contains(src, `"congratulations"`) {
			t.Errorf("client/DessauChat/%s repeats an effect word; the list lives in Effects.swift alone", name)
		}
	}
}

// TestChatClientRendersMarkdownWithTheFullSyntax holds the formatted-replies
// intent's mechanism: the full interpreted syntax, so block intents exist, and
// a partial parse rather than nothing when a reply is cut mid-mark.
func TestChatClientRendersMarkdownWithTheFullSyntax(t *testing.T) {
	root := repoRootDir(t)
	md, ok := clientSources(t, root)["Markdown.swift"]
	if !ok {
		t.Fatal("client/DessauChat/Markdown.swift is missing")
	}
	for _, want := range []string{"interpretedSyntax: .full", "failurePolicy: .returnPartiallyParsedIfPossible"} {
		if !strings.Contains(md, want) {
			t.Errorf("client/DessauChat/Markdown.swift does not parse with %q", want)
		}
	}
}

// TestChatClientMenuActionsCarryShortcuts holds the trunk's second criterion:
// every action in the Chat menu has a keyboard shortcut.
func TestChatClientMenuActionsCarryShortcuts(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["DessauChat.swift"]
	start := strings.Index(src, `CommandMenu("Chat")`)
	end := strings.Index(src, "Settings {")
	if start < 0 || end < 0 {
		t.Fatal("client/DessauChat/DessauChat.swift declares no Chat command menu before its Settings scene")
	}
	menu := src[start:end]
	buttons := strings.Count(menu, "Button(") + strings.Count(menu, "Button {")
	shortcuts := strings.Count(menu, ".keyboardShortcut(")
	if buttons == 0 || buttons != shortcuts {
		t.Errorf("the Chat menu has %d actions and %d shortcuts; every action carries one", buttons, shortcuts)
	}
}

// TestChatClientEffectPlaysOnceWhenTheReplyFinishes holds the text-effects
// intent's once-only trigger (iss-2609181116217704). The model queues a
// message id in the stream's defer — when the reply FINISHES — so a row that
// latches at the reply block's first appearance latches at the first streamed
// token, before any id is there: the finishing reply never animates, and the
// id it left behind fires instead on whatever row scroll recreates next. The
// row therefore watches the model's set for the change that queues the id, and
// a row that finds the id already there consumes it without playing.
func TestChatClientEffectPlaysOnceWhenTheReplyFinishes(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["DessauChat.swift"]

	if !strings.Contains(src, "onChange(of: model.effectsToPlay.contains(message.id))") {
		t.Error("MessageRow does not watch model.effectsToPlay for this message's id; " +
			"an effect latched at the block's first appearance is latched at the first streamed token, " +
			"before the stream's defer has queued the id — so a finishing reply never animates")
	}
	if strings.Contains(src, "@State private var playing: Bool?") {
		t.Error("MessageRow still latches an optional `playing` on first appearance; " +
			"the once-only flag belongs to the change that queues the id, not to the row appearing")
	}
	// The consuming call has to sit in both places: on the change that plays
	// the effect, and on the appearance of a row that found the id already
	// queued — the reply finished off screen and its moment has passed.
	if n := strings.Count(src, "model.effectStarted(message.id)"); n < 2 {
		t.Errorf("model.effectStarted(message.id) is called %d time(s); the id is consumed both when the effect plays "+
			"and when a recreated row finds it stale, so no later row can fire on it", n)
	}
	if !strings.Contains(src, "@State private var played = false") {
		t.Error("MessageRow keeps no record that it has already played; the effect plays exactly once per row")
	}
	if !strings.Contains(src, "effectsToPlay.insert(messageID)") {
		t.Error("AppModel queues no message id for the effect; the trigger has nothing to watch")
	}
}

// TestChatClientOpensASecondWindow holds the trunk intent's multi-window
// criterion (iss-2609181116079882): replacing the .newItem group removes the
// standard New Window item, so the client has to put one back and open a
// second window of its own WindowGroup itself. New Chat keeps Cmd-N.
func TestChatClientOpensASecondWindow(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["DessauChat.swift"]

	if !strings.Contains(src, `@Environment(\.openWindow)`) {
		t.Error("the app reads no openWindow action; nothing in the client can open a second window")
	}
	if !strings.Contains(src, `Button("New Window")`) {
		t.Error(`the File commands carry no "New Window" item; replacing .newItem removed the system's own`)
	}
	if !strings.Contains(src, `.keyboardShortcut("n", modifiers: [.command, .shift])`) {
		t.Error("New Window has no Cmd-Shift-N shortcut")
	}
	if !strings.Contains(src, `.keyboardShortcut("n", modifiers: .command)`) {
		t.Error("New Chat has lost its Cmd-N shortcut")
	}
	// The id openWindow is given has to be the id the WindowGroup declares, or
	// the action opens nothing and says so only at runtime.
	group := regexp.MustCompile(`WindowGroup\("[^"]*", id: ([A-Za-z0-9_]+)\)`).FindStringSubmatch(src)
	if group == nil {
		t.Fatal("the WindowGroup declares no id; openWindow has nothing to name")
	}
	open := regexp.MustCompile(`openWindow\(id: ([A-Za-z0-9_]+)\)`).FindStringSubmatch(src)
	if open == nil {
		t.Fatal("nothing calls openWindow(id:)")
	}
	if group[1] != open[1] {
		t.Errorf("openWindow is given %q while the WindowGroup declares %q; the action would open nothing", open[1], group[1])
	}
}

// TestChatClientEffectKeepsTheReplySelectable holds the falsifier the
// text-effects intent's Grounds names (iss-2609181116218290): while the
// renderer plays, the block is rebuilt as concatenated Texts, and a Text that
// carries no textSelection cannot be selected — so for the effect's duration
// the reply would be unselectable.
func TestChatClientEffectKeepsTheReplySelectable(t *testing.T) {
	root := repoRootDir(t)
	effects := clientSources(t, root)["Effects.swift"]
	renderer := strings.Index(effects, ".textRenderer(")
	if renderer < 0 {
		t.Fatal("client/DessauChat/Effects.swift installs no text renderer")
	}
	if !strings.Contains(effects[renderer:], ".textSelection(.enabled)") {
		t.Error("the block being animated carries no .textSelection(.enabled); " +
			"the reply is unselectable for the effect's duration")
	}
}

// TestChatClientReplyReparseStaysWithinItsBudget holds the streaming reply's
// re-parse budget (iss-2609181116218893): the spec allows at most four parses
// a second, so the debounce waits at least a quarter of a second — for the
// reply and for the thoughts, which are parsed the same way, on the one
// debounce they share (TestChatClientThoughtsShareTheReplyScheduler).
func TestChatClientReplyReparseStaysWithinItsBudget(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["DessauChat.swift"]
	found := regexp.MustCompile(`([0-9.]+) - Date\(\)\.timeIntervalSince\(last[A-Za-z]*Parse\)`).FindAllStringSubmatch(src, -1)
	if len(found) == 0 {
		t.Fatal("no re-parse debounce; a streaming reply is parsed on every chunk that arrives")
	}
	for _, m := range found {
		wait, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("re-parse debounce %q is not a number", m[1])
		}
		if wait < 0.25 {
			t.Errorf("the re-parse debounce is %.2fs, which admits %.0f parses a second; the budget is four", wait, 1/wait)
		}
	}
}

// TestChatClientBubbleTextReadsOnItsBubble holds the bubble colours to being
// readable (iss-2609181200258524): the text colour follows the colour of the
// bubble it sits on, rather than being a fixed white that disappears on a
// light bubble the person chose.
func TestChatClientBubbleTextReadsOnItsBubble(t *testing.T) {
	root := repoRootDir(t)
	bubbles := clientSources(t, root)["Bubbles.swift"]
	if strings.Contains(bubbles, "isUser ? Color.white : Color.primary") {
		t.Error("the person's bubble draws its text in a fixed white whatever colour the bubble is; " +
			"white on a light bubble cannot be read")
	}
	if !strings.Contains(bubbles, "isDark") {
		t.Error("nothing in client/DessauChat/Bubbles.swift asks how dark a bubble is; " +
			"the text colour cannot follow the bubble it sits on")
	}
	// A chosen colour survives; the way back is the Default button.
	if !strings.Contains(bubbles, `Button("Default")`) {
		t.Error("a chosen bubble colour has no way back to the default")
	}
}

// TestChatClientBubbleColoursFollowTheAppearance holds the bubble colours to
// the chosen appearance (iss-2609181124295668). The defaults are semantic
// system colours, which resolve themselves in Light and Dark; a colour the
// person picks is a fixed value, so it is drawn through a path with a face
// per appearance; and no fixed white is left as the person's bubble text.
func TestChatClientBubbleColoursFollowTheAppearance(t *testing.T) {
	root := repoRootDir(t)
	bubbles := clientSources(t, root)["Bubbles.swift"]
	for _, want := range []string{"Color.accentColor", "Color.secondary"} {
		if !strings.Contains(bubbles, want) {
			t.Errorf("the bubble defaults do not name the semantic system colour %s; "+
				"a default that is not the system's cannot follow Light and Dark", want)
		}
	}
	if strings.Contains(bubbles, "Color.gray") {
		t.Error("the model's bubble defaults to Color.gray, which is the same grey in Light and in Dark")
	}
	if strings.Contains(bubbles, "Color.white") {
		t.Error("client/DessauChat/Bubbles.swift still draws a fixed Color.white; " +
			"the bubble's text has to be the system's label colour, read in the appearance the fill reads as")
	}
	for _, want := range []string{"dynamicProvider:", "userInterfaceStyle", `.environment(\.colorScheme`} {
		if !strings.Contains(bubbles, want) {
			t.Errorf("client/DessauChat/Bubbles.swift does not carry %q; a picked colour is drawn "+
				"as its stored value whatever the appearance", want)
		}
	}
}

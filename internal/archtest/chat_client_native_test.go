package archtest_test

import (
	"os"
	"path/filepath"
	"regexp"
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
	matches, err := filepath.Glob(filepath.Join(root, "client", "GropiusChat", "*.swift"))
	if err != nil || len(matches) == 0 {
		t.Fatal("client/GropiusChat/ holds no Swift file")
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
		"Effects.swift": "the text-effects renderer draws glyphs itself; that is its purpose",
	}
	for name, src := range clientSources(t, root) {
		if _, ok := exempt[name]; ok {
			continue
		}
		for _, s := range styling {
			if strings.Contains(src, s) {
				t.Errorf("client/GropiusChat/%s uses %q; the client carries no styling of its own — "+
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
		t.Fatal("client/GropiusChat/Backends.swift is missing; the answerer seam has no home")
	}
	if strings.Contains(src, "import Network") {
		t.Error("client/GropiusChat/Backends.swift imports Network; the built-in answerer must touch no network")
	}
	start := strings.Index(src, "struct BuiltInBackend")
	end := strings.Index(src, "struct ServerBackend")
	if start < 0 || end < 0 || end < start {
		t.Fatal("client/GropiusChat/Backends.swift does not declare BuiltInBackend before ServerBackend")
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
				t.Errorf("client/GropiusChat/%s uses %q; the composer is a standard TextField so the system's Writing Tools reach it", name, s)
			}
		}
	}
	if !regexp.MustCompile(`TextField\("Message…", text: \$draft, axis: \.vertical\)`).MatchString(all["GropiusChat.swift"]) {
		t.Error("client/GropiusChat/GropiusChat.swift has no multi-line TextField composer")
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
		t.Fatal("client/GropiusChat/Picker.swift is missing")
	}
	onAppear := regexp.MustCompile(`\.onAppear\s*\{[^}]*browser\.start\(\)`)
	onDisappear := regexp.MustCompile(`\.onDisappear\s*\{[^}]*browser\.stop\(\)`)
	if !onAppear.MatchString(picker) {
		t.Error("client/GropiusChat/Picker.swift does not start the browse in .onAppear")
	}
	if !onDisappear.MatchString(picker) {
		t.Error("client/GropiusChat/Picker.swift does not stop the browse in .onDisappear")
	}
	for name, src := range all {
		if name != "Picker.swift" && strings.Contains(src, "browser.start()") {
			t.Errorf("client/GropiusChat/%s starts a browse; only the picker may", name)
		}
	}
}

// TestChatClientSendsTheKeyToOneHostOnly holds the offer intent's promise
// about the API key: it is attached only to the host it was entered for.
func TestChatClientSendsTheKeyToOneHostOnly(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	if !regexp.MustCompile(`if !apiKey\.isEmpty, origin == apiKeyHost \{`).MatchString(src) {
		t.Error("client/GropiusChat/GropiusChat.swift attaches the bearer token without comparing the request's origin to apiKeyHost")
	}
	// The Settings pane must not re-bind the stored key on its initial load:
	// the change handler ignores a value equal to the stored key.
	if !regexp.MustCompile(`guard new != model\.apiKey else \{ return \}`).MatchString(src) {
		t.Error("client/GropiusChat/GropiusChat.swift saves the API key on every change of the field, including the initial load, which re-binds it to the current server")
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
		t.Fatal("client/GropiusChat/Effects.swift is missing")
	}
	list := regexp.MustCompile(`static let list: \[\(word: String, kind: EffectKind\)\] = \[([^\]]*)\]`).FindStringSubmatch(effects)
	if list == nil {
		t.Fatal("client/GropiusChat/Effects.swift declares no EffectWords.list")
	}
	words := regexp.MustCompile(`\("([^"]+)", \.`).FindAllStringSubmatch(list[1], -1)
	if len(words) < 3 || len(words) > 8 {
		t.Errorf("EffectWords.list has %d words; the list is small and fixed", len(words))
	}
	if !strings.Contains(effects, `options: [.caseInsensitive]`) {
		t.Error("EffectWords.matches is not case-insensitive")
	}
	if !strings.Contains(all["GropiusChat.swift"], "EffectWords.settingsSentence") {
		t.Error("the Settings page does not name the words from EffectWords")
	}
	for name, src := range all {
		if name != "Effects.swift" && strings.Contains(src, `"congratulations"`) {
			t.Errorf("client/GropiusChat/%s repeats an effect word; the list lives in Effects.swift alone", name)
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
		t.Fatal("client/GropiusChat/Markdown.swift is missing")
	}
	for _, want := range []string{"interpretedSyntax: .full", "failurePolicy: .returnPartiallyParsedIfPossible"} {
		if !strings.Contains(md, want) {
			t.Errorf("client/GropiusChat/Markdown.swift does not parse with %q", want)
		}
	}
}

// TestChatClientMenuActionsCarryShortcuts holds the trunk's second criterion:
// every action in the Chat menu has a keyboard shortcut.
func TestChatClientMenuActionsCarryShortcuts(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]
	start := strings.Index(src, `CommandMenu("Chat")`)
	end := strings.Index(src, "Settings {")
	if start < 0 || end < 0 {
		t.Fatal("client/GropiusChat/GropiusChat.swift declares no Chat command menu before its Settings scene")
	}
	menu := src[start:end]
	buttons := strings.Count(menu, "Button(") + strings.Count(menu, "Button {")
	shortcuts := strings.Count(menu, ".keyboardShortcut(")
	if buttons == 0 || buttons != shortcuts {
		t.Errorf("the Chat menu has %d actions and %d shortcuts; every action carries one", buttons, shortcuts)
	}
}

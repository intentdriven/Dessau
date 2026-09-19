package archtest_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The sidebar's search is the one part of the client a person uses without
// thinking about it and notices the moment it is wrong: a two-word search that
// finds nothing because the words are not adjacent reads as a chat that is
// gone. The client has no test target of its own, so the match lives in one
// file that imports Foundation and nothing of the client's
// (client/GropiusChat/SidebarSearch.swift) and is checked by a Swift main that
// `swiftc` compiles on its own. These two tests are how that check reaches CI:
// one runs it, the other holds the shipped sidebar to using it.

// TestChatClientSidebarSearchWords runs client/tests/sidebar-search.sh, which
// compiles the search with its main and runs it. Without Xcode's Swift
// toolchain there is nothing to compile with, and the test says it skipped
// rather than passing on an empty run.
func TestChatClientSidebarSearchWords(t *testing.T) {
	root := repoRootDir(t)
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("SKIPPED, NOT PASSED: xcrun is not on PATH, so the client's sidebar search was not compiled or run")
	}
	if out, err := exec.Command("xcrun", "--find", "swiftc").CombinedOutput(); err != nil {
		t.Skipf("SKIPPED, NOT PASSED: no Swift toolchain (xcrun --find swiftc: %v: %s), "+
			"so the client's sidebar search was not compiled or run", err, strings.TrimSpace(string(out)))
	}
	script := filepath.Join(root, "client", "tests", "sidebar-search.sh")
	cmd := exec.Command("bash", script)
	out, err := cmd.CombinedOutput()
	t.Logf("client/tests/sidebar-search.sh:\n%s", out)
	if err != nil {
		t.Fatalf("the sidebar's search is broken (%v); see the output above", err)
	}
}

// TestChatClientSidebarSearchesEveryWord holds the shipped sidebar to the
// match the check above exercises (iss-2609181213191220): the query is split
// into words and a conversation is left only when EVERY word is somewhere in
// its title or its messages, which is what itd-2609181104498312 promises by
// "conversations whose title or messages contain the words". Matching the
// whole query as one substring keeps only the conversations that carry the
// words adjacent, in the order typed. Reading the source is how a Go suite can
// hold a Swift client.
func TestChatClientSidebarSearchesEveryWord(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)

	search, ok := all["SidebarSearch.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/SidebarSearch.swift is missing; the sidebar's search has no home of its own")
	}
	// The file has to compile on its own with nothing but Foundation, which
	// is what lets client/tests/sidebar-search.sh check it.
	for _, line := range strings.Split(search, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") && strings.TrimSpace(line) != "import Foundation" {
			t.Errorf("client/GropiusChat/SidebarSearch.swift carries %q; it must compile on its own "+
				"against Foundation alone", strings.TrimSpace(line))
		}
	}
	for _, forbidden := range []string{"SwiftUI", "Conversation", "AppModel"} {
		if strings.Contains(search, forbidden) {
			t.Errorf("client/GropiusChat/SidebarSearch.swift mentions %q; it must compile on its own, "+
				"which is what lets client/tests/sidebar-search.sh check it", forbidden)
		}
	}
	// The two halves of the promise: the query becomes words, and every one
	// of them has to be found.
	if !strings.Contains(search, "isWhitespace") {
		t.Error("client/GropiusChat/SidebarSearch.swift does not split the query on whitespace; " +
			"a multi-word search would be one literal substring again (iss-2609181213191220)")
	}
	if !strings.Contains(search, "allSatisfy") {
		t.Error("client/GropiusChat/SidebarSearch.swift does not require every word to be found; " +
			"a search that narrows on one word only is not the promise")
	}
	// Case and accents are not what a search is about, and the standard
	// comparison is the one the rest of the platform searches with.
	if !strings.Contains(search, "localizedStandardContains") {
		t.Error("client/GropiusChat/SidebarSearch.swift does not match with localizedStandardContains; " +
			"the search would be sensitive to case or to accents")
	}

	// And the sidebar has to be the thing calling it: a pure function nothing
	// uses would pass every check above while the list filtered some other way.
	src, ok := all["GropiusChat.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/GropiusChat.swift is missing")
	}
	sidebar := swiftBlock(t, src, "struct Sidebar: View {")
	for _, want := range []string{"SidebarSearch.words(in: query)", "SidebarSearch.matches("} {
		if !strings.Contains(sidebar, want) {
			t.Errorf("the sidebar's filter does not call %s; it is doing its own matching, "+
				"which client/tests/sidebar-search.sh does not reach", want)
		}
	}
	if !strings.Contains(sidebar, "title: c.title") || !strings.Contains(sidebar, "c.messages.map") {
		t.Error("the sidebar's filter does not hand the search both the title and the message text")
	}
}

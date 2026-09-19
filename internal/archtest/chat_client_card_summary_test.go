package archtest_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A sidebar card says how many exchanges a conversation holds, and an exchange
// is a person's message and the reply that answers it (itd-2609181104490133) —
// arithmetic that is wrong only on the conversations nobody looks at twice.
// The client has no test target of its own, so the count lives in one file
// that imports nothing (client/GropiusChat/CardSummary.swift) and is checked
// by a Swift main that `swiftc` compiles on its own. These two tests are how
// that check reaches CI: one runs it, the other holds the file to being
// checkable at all.

// TestChatClientCardSummaryArithmetic runs client/tests/card-summary.sh, which
// compiles the count with its main and runs it. Without Xcode's Swift
// toolchain there is nothing to compile with, and the test says it skipped
// rather than passing on an empty run.
func TestChatClientCardSummaryArithmetic(t *testing.T) {
	root := repoRootDir(t)
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("SKIPPED, NOT PASSED: xcrun is not on PATH, so the card's exchange count was not compiled or run")
	}
	if out, err := exec.Command("xcrun", "--find", "swiftc").CombinedOutput(); err != nil {
		t.Skipf("SKIPPED, NOT PASSED: no Swift toolchain (xcrun --find swiftc: %v: %s), "+
			"so the card's exchange count was not compiled or run", err, strings.TrimSpace(string(out)))
	}
	script := filepath.Join(root, "client", "tests", "card-summary.sh")
	out, err := exec.Command("bash", script).CombinedOutput()
	t.Logf("client/tests/card-summary.sh:\n%s", out)
	if err != nil {
		t.Fatalf("the sidebar card's exchange count is broken (%v); see the output above", err)
	}
}

// TestChatClientCardSummaryCompilesAlone holds the arithmetic to a file of its
// own that names no framework type and imports nothing — which is what lets
// the check above compile and run it without a device.
func TestChatClientCardSummaryCompilesAlone(t *testing.T) {
	root := repoRootDir(t)
	summary, ok := clientSources(t, root)["CardSummary.swift"]
	if !ok {
		t.Fatal("client/GropiusChat/CardSummary.swift is missing; the card's arithmetic has no home of its own")
	}
	for _, forbidden := range []string{"import ", "SwiftUI", "Conversation", "Message"} {
		if strings.Contains(summary, forbidden) {
			t.Errorf("client/GropiusChat/CardSummary.swift mentions %q; it must compile on its own, "+
				"which is what lets client/tests/card-summary.sh check it", forbidden)
		}
	}
	if !strings.Contains(summary, "func exchangeCount(fromPerson") {
		t.Error("client/GropiusChat/CardSummary.swift declares no exchangeCount(fromPerson:); " +
			"the card would be doing its own counting, which nothing checks")
	}
}

package archtest_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The built-in model answers inside a context window the client has to share
// out between its instructions, the conversation so far, the message just
// typed and the room the reply needs. Getting that arithmetic wrong is not
// visible until a conversation is long, and the client has no test target of
// its own — so the arithmetic lives in one file that imports nothing
// (client/DessauChat/ContextBudget.swift) and is checked by a Swift main that
// `swiftc` compiles on its own. These two tests are how that check reaches CI:
// one runs it, the other holds the shipped backend to using it.

// TestChatClientContextBudgetArithmetic runs client/tests/context-budget.sh,
// which compiles the budget with its main and runs it. Without Xcode's Swift
// toolchain there is nothing to compile with, and the test says it skipped
// rather than passing on an empty run.
func TestChatClientContextBudgetArithmetic(t *testing.T) {
	root := repoRootDir(t)
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("SKIPPED, NOT PASSED: xcrun is not on PATH, so the client's context arithmetic was not compiled or run")
	}
	if out, err := exec.Command("xcrun", "--find", "swiftc").CombinedOutput(); err != nil {
		t.Skipf("SKIPPED, NOT PASSED: no Swift toolchain (xcrun --find swiftc: %v: %s), "+
			"so the client's context arithmetic was not compiled or run", err, strings.TrimSpace(string(out)))
	}
	script := filepath.Join(root, "client", "tests", "context-budget.sh")
	cmd := exec.Command("bash", script)
	out, err := cmd.CombinedOutput()
	t.Logf("client/tests/context-budget.sh:\n%s", out)
	if err != nil {
		t.Fatalf("the built-in model's context arithmetic is broken (%v); see the output above", err)
	}
}

// TestChatClientTrimBudgetsTheNewPrompt holds the shipped backend to the
// arithmetic the check above exercises (iss-2609181116072455): the trim counts
// the new prompt's own tokens alongside the prior turns, and a prompt that
// cannot fit on its own is refused in the transcript rather than sent and
// retried. Reading the source is how a Go suite can hold a Swift client.
func TestChatClientTrimBudgetsTheNewPrompt(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)

	budget, ok := all["ContextBudget.swift"]
	if !ok {
		t.Fatal("client/DessauChat/ContextBudget.swift is missing; the context arithmetic has no home of its own")
	}
	// The file has to compile on its own, which means it names none of the
	// framework's types and imports nothing at all.
	for _, forbidden := range []string{"import ", "Transcript", "SystemLanguageModel", "LanguageModelSession"} {
		if strings.Contains(budget, forbidden) {
			t.Errorf("client/DessauChat/ContextBudget.swift mentions %q; it must compile on its own, "+
				"which is what lets client/tests/context-budget.sh check it", forbidden)
		}
	}
	if !strings.Contains(budget, "window - reserve - instructions - prompt") {
		t.Error("client/DessauChat/ContextBudget.swift does not take the new prompt's tokens out of the window; " +
			"a prompt that overflows on its own would be left to the retry to absorb (iss-2609181116072455)")
	}

	backends, ok := all["Backends.swift"]
	if !ok {
		t.Fatal("client/DessauChat/Backends.swift is missing; the answerer seam has no home")
	}
	if !strings.Contains(backends, "ContextBudget(") {
		t.Error("client/DessauChat/Backends.swift builds no ContextBudget; the trim is doing its own arithmetic, " +
			"which nothing checks")
	}
	if !strings.Contains(backends, "prompt: try await model.tokenCount(") {
		t.Error("client/DessauChat/Backends.swift does not count the new prompt's tokens; " +
			"the budget would be given the prior turns only")
	}
	// Not fitting has to end the turn, not soften it: the person is told, and
	// nothing is sent.
	if !strings.Contains(backends, "budget.fits") {
		t.Error("client/DessauChat/Backends.swift never asks whether the prompt fits; " +
			"a prompt too long for the window would still be sent")
	}
	if !strings.Contains(backends, "promptTooLong") {
		t.Error("client/DessauChat/Backends.swift carries no sentence for a prompt that does not fit; " +
			"the transcript would show the framework's overflow error instead of the client's own words")
	}
	// The retry stays: the token counts are the framework's estimate, and an
	// estimate can be wrong in the direction that matters.
	if !strings.Contains(backends, "case .contextSizeExceeded where attempt == 0:") {
		t.Error("client/DessauChat/Backends.swift no longer retries once on a context overflow; " +
			"budgeting the prompt is an estimate, and the retry is what covers an estimate that was wrong")
	}
}

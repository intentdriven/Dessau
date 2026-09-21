package archtest_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The picker's transcript icon is drawn from the models-list `recording`
// field (itd-2609091715089488): which models keep no transcript, joined
// folded like every other join on a repo id, and the words the icon is
// labelled with. The client has no test target of its own, so that rule
// lives in one file that imports nothing (client/DessauChat/
// TranscriptState.swift) and is checked by a Swift main that `swiftc`
// compiles on its own — the client's unit tier (the 2026-09-20 decision).
// These two tests are how that check reaches CI: one runs it, the other
// holds the file to being checkable at all.

// TestChatClientTranscriptStateRule runs client/tests/transcript-state.sh,
// which compiles the rule with its main and runs it. Without Xcode's Swift
// toolchain there is nothing to compile with, and the test says it skipped
// rather than passing on an empty run.
func TestChatClientTranscriptStateRule(t *testing.T) {
	root := repoRootDir(t)
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("SKIPPED, NOT PASSED: xcrun is not on PATH, so the picker's transcript rule was not compiled or run")
	}
	if out, err := exec.Command("xcrun", "--find", "swiftc").CombinedOutput(); err != nil {
		t.Skipf("SKIPPED, NOT PASSED: no Swift toolchain (xcrun --find swiftc: %v: %s), "+
			"so the picker's transcript rule was not compiled or run", err, strings.TrimSpace(string(out)))
	}
	script := filepath.Join(root, "client", "tests", "transcript-state.sh")
	out, err := exec.Command("bash", script).CombinedOutput()
	t.Logf("client/tests/transcript-state.sh:\n%s", out)
	if err != nil {
		t.Fatalf("the picker's transcript rule is broken (%v); see the output above", err)
	}
}

// TestChatClientTranscriptStateCompilesAlone holds the rule to a file of its
// own that names no framework type and imports nothing — which is what lets
// the check above compile and run it without a device — and holds the
// picker to reading the rule rather than the field, so the fold is decided
// in one place. The words are the picker's own, where
// chat_client_transcript_icon_test.go holds them.
func TestChatClientTranscriptStateCompilesAlone(t *testing.T) {
	root := repoRootDir(t)
	sources := clientSources(t, root)
	rule, ok := sources["TranscriptState.swift"]
	if !ok {
		t.Fatal("client/DessauChat/TranscriptState.swift is missing; the picker's transcript rule has no home of its own")
	}
	for _, forbidden := range []string{"import ", "SwiftUI", "AppModel", "ModelsResponse"} {
		if strings.Contains(rule, forbidden) {
			t.Errorf("client/DessauChat/TranscriptState.swift mentions %q; it must compile on its own, "+
				"which is what lets client/tests/transcript-state.sh check it", forbidden)
		}
	}
	for _, want := range []string{"func transcriptStates(", "func transcriptRecorded("} {
		if !strings.Contains(rule, want) {
			t.Errorf("client/DessauChat/TranscriptState.swift declares no %s; the picker would be doing its own deciding, which nothing checks", want)
		}
	}
	picker := sources["Picker.swift"]
	rows, ok := swiftFunctionBody(picker, "private var modelRows: some View")
	if !ok {
		t.Fatal("client/DessauChat/Picker.swift no longer declares modelRows")
	}
	if !strings.Contains(rows, "transcriptRecorded(") {
		t.Error("the picker's model rows do not call transcriptRecorded(); the icon is then drawn from a second reading of the field")
	}
	if !strings.Contains(sources["DessauChat.swift"], "transcriptStates(") {
		t.Error("client/DessauChat/DessauChat.swift does not build the transcript states with transcriptStates(); the connect path keeps its own copy of the rule")
	}
}

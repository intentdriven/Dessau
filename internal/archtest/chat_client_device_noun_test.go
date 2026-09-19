package archtest_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// One source is compiled for the Mac and for the iPad, so a sentence that
// names the device has to be told which device it is running on.
// BuiltInBackend.deviceNoun is that one place: "Mac" in the macOS branch,
// "iPad" everywhere else. A sentence that spells the device out instead reads,
// on an iPad, as a name for something the person is not holding
// (iss-2609190023129175).
//
// The guard is on the string literals rather than the whole text of the file:
// the prose that explains the code may call it the Mac's own model, because
// nobody is shown a comment.

// swiftLiteral is one string literal and the line its quote opened on.
type swiftLiteral struct {
	line int
	text string
}

// swiftStringLiterals reads the string literals out of one Swift source,
// skipping line comments and reading a `"""` block as a single literal. It
// keeps an interpolation's text (`\(deviceNoun)`) rather than resolving it,
// which is exactly what the guard needs to see.
func swiftStringLiterals(src string) []swiftLiteral {
	var out []swiftLiteral
	var block strings.Builder
	blockStart := 0
	inBlock := false

	for n, line := range strings.Split(src, "\n") {
		if inBlock {
			if before, _, found := strings.Cut(line, `"""`); found {
				block.WriteString(before)
				out = append(out, swiftLiteral{blockStart, block.String()})
				block.Reset()
				inBlock = false
			} else {
				block.WriteString(line + "\n")
			}
			continue
		}
		r := []rune(line)
		for i := 0; i < len(r); {
			switch {
			case r[i] == '/' && i+1 < len(r) && r[i+1] == '/':
				i = len(r) // the rest of the line explains, it does not speak
			case r[i] == '"' && i+2 < len(r) && r[i+1] == '"' && r[i+2] == '"':
				inBlock, blockStart = true, n+1
				i = len(r)
			case r[i] == '"':
				var lit strings.Builder
				for i++; i < len(r) && r[i] != '"'; i++ {
					if r[i] == '\\' && i+1 < len(r) {
						lit.WriteRune(r[i])
						i++
					}
					lit.WriteRune(r[i])
				}
				i++
				out = append(out, swiftLiteral{n + 1, lit.String()})
			default:
				i++
			}
		}
	}
	return out
}

// macOSBranchLines marks, for each 1-based line of a Swift source, whether some
// enclosing conditional-compilation branch is the macOS one -- the same reading
// TestChatClientGuardsTheMacOnlyCalls does, kept here as a per-line answer so a
// literal can be looked up by the line it opened on.
func macOSBranchLines(src string) []bool {
	lines := strings.Split(src, "\n")
	marks := make([]bool, len(lines)+1)
	type branch struct{ guardsMac, inMacBranch bool }
	var stack []branch
	for n, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "#if "):
			condition := strings.TrimPrefix(trimmed, "#if ")
			stack = append(stack, branch{
				guardsMac:   strings.Contains(condition, "os(macOS)"),
				inMacBranch: condition == "os(macOS)",
			})
			continue
		case trimmed == "#else":
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				top.inMacBranch = top.guardsMac && !top.inMacBranch
			}
			continue
		case trimmed == "#endif":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		for _, b := range stack {
			if b.guardsMac && b.inMacBranch {
				marks[n+1] = true
				break
			}
		}
	}
	return marks
}

// TestChatClientNamesTheDeviceThePersonIsHolding holds every sentence the
// built-in backend shows to the one noun that knows which device it is on.
//
// The failure it guards is silent on the machine the client is written on: a
// hard-coded "the Mac's own model" is right on a Mac and wrong on an iPad, and
// only the iPad build shows it. Nothing else in the file may spell the device
// out either -- the declaration of deviceNoun itself is inside the macOS
// branch, which is the one place the word belongs.
func TestChatClientNamesTheDeviceThePersonIsHolding(t *testing.T) {
	root := repoRootDir(t)
	src := readRepoFile(t, root, filepath.Join("client", "GropiusChat", "Backends.swift"))

	if !strings.Contains(src, `static let deviceNoun = "Mac"`) {
		t.Fatal("client/GropiusChat/Backends.swift declares no deviceNoun = \"Mac\"; " +
			"the one place the device is named has moved and this guard reads the wrong file")
	}

	macOS := macOSBranchLines(src)
	for _, lit := range swiftStringLiterals(src) {
		if lit.line < len(macOS) && macOS[lit.line] {
			continue
		}
		if strings.Contains(lit.text, "Mac") {
			t.Errorf("client/GropiusChat/Backends.swift:%d spells the device out: %q. "+
				"The same source is compiled for the iPad, where that sentence names a device "+
				"the person is not holding; interpolate deviceNoun instead", lit.line, lit.text)
		}
	}
}

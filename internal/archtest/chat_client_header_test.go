package archtest_test

import (
	"regexp"
	"strings"
	"testing"
)

// The chat client's header is one thin toolbar, whatever the scroll state and
// whatever the conversation holds (iss-2609190004097595). Three constructs in
// the client's own source decide that, and each of them is a promise a later
// edit could quietly undo:
//
//   - the toolbar's title display mode. Left automatic, macOS 26 and iPadOS 26
//     resolve it to a large title that collapses as the transcript scrolls, so
//     the header has two heights;
//   - the scroll-edge effect. Left automatic, it is a soft progressive blur
//     that reaches far past the bar and washes out the top of the transcript
//     under it;
//   - the transcript's default scroll anchor. The plain form sets the
//     ALIGNMENT role too, which pins content shorter than the window to its
//     bottom edge and opens an empty band between the toolbar and the first
//     bubble on a fresh conversation.
//
// None of the three is visible to `go test`, so they are read here out of the
// Swift the Mac and the iPad are both built from.

// transcriptView cuts ChatDetail's transcript out of the client's source.
func transcriptView(t *testing.T, src string) string {
	t.Helper()
	start := regexp.MustCompile(`private var transcript: some View \{`).FindStringIndex(src)
	if start == nil {
		t.Fatal("client/GropiusChat/GropiusChat.swift declares no transcript view")
	}
	end := regexp.MustCompile(`\n    private var composer: some View \{`).FindStringIndex(src[start[0]:])
	if end == nil {
		t.Fatal("the transcript view is not followed by the composer")
	}
	return src[start[0] : start[0]+end[0]]
}

// TestChatClientHeaderIsOneFixedHeight holds the header to one height.
func TestChatClientHeaderIsOneFixedHeight(t *testing.T) {
	root := repoRootDir(t)
	src := clientSources(t, root)["GropiusChat.swift"]

	t.Run("every title is pinned inline", func(t *testing.T) {
		titles := regexp.MustCompile(`\.navigationTitle\(`).FindAllStringIndex(src, -1)
		if len(titles) == 0 {
			t.Fatal("client/GropiusChat/GropiusChat.swift sets no navigation title")
		}
		inline := regexp.MustCompile(`TitleDisplayMode\(\.inline\)`)
		for i, at := range titles {
			// As far as the next title, or a few lines on — far enough to
			// carry the comment that says why the mode is pinned, near enough
			// that one view's modifier cannot stand in for another's.
			stop := min(at[0]+600, len(src))
			if i+1 < len(titles) && titles[i+1][0] < stop {
				stop = titles[i+1][0]
			}
			tail := src[at[0]:stop]
			line := strings.SplitN(tail, "\n", 2)[0]
			if !inline.MatchString(tail) {
				t.Errorf("%s is not followed by an inline title display mode; "+
					"an automatic title is drawn large and collapses as the transcript scrolls, "+
					"so the header has two heights", strings.TrimSpace(line))
			}
		}
		for _, grown := range []string{".inlineLarge", "toolbarTitleDisplayMode(.large)"} {
			if strings.Contains(src, grown) {
				t.Errorf("the client asks for %s; the header is one thin toolbar, as Messages' is", grown)
			}
		}
	})

	t.Run("the scroll-edge effect is confined to the toolbar", func(t *testing.T) {
		transcript := transcriptView(t, src)
		if !strings.Contains(transcript, ".scrollEdgeEffectStyle(.hard, for: .top)") {
			t.Error("the transcript does not give its top edge the hard scroll-edge style; " +
				"the automatic style is a soft progressive blur that reaches well past the toolbar " +
				"and washes out the transcript under it")
		}
	})

	t.Run("short content sits under the toolbar, not at the window's foot", func(t *testing.T) {
		transcript := transcriptView(t, src)
		if regexp.MustCompile(`\.defaultScrollAnchor\(\.bottom\)`).MatchString(transcript) {
			t.Error("the transcript sets the plain .defaultScrollAnchor(.bottom); that anchors the " +
				"ALIGNMENT role as well, so a conversation shorter than the window is pinned to its " +
				"bottom edge and an empty band opens between the toolbar and the first bubble")
		}
		if !strings.Contains(transcript, `.defaultScrollAnchor(.bottom, for: .initialOffset)`) {
			t.Error("the transcript does not open at its newest message; only the INITIAL OFFSET " +
				"role is anchored to the bottom, and the roles do not compose across two of these " +
				"modifiers — a second one replaces the first")
		}
	})

	t.Run("the toolbar's items do not change height", func(t *testing.T) {
		// The answerer button swaps a spinner for a symbol while the client is
		// connecting. The two do not measure the same, and a toolbar item that
		// changes height changes the header's height with it.
		icon := regexp.MustCompile(`(?s)\} icon: \{(.*?)\.labelStyle\(\.titleAndIcon\)`).FindStringSubmatch(src)
		if icon == nil {
			t.Fatal("the answerer button in the toolbar has no icon")
		}
		if !strings.Contains(icon[1], "model.connecting") {
			t.Fatal("the answerer button's icon no longer switches on the connecting state; this test is out of date")
		}
		if !regexp.MustCompile(`\.frame\(width: \d+, height: \d+\)`).MatchString(icon[1]) {
			t.Error("the answerer button's icon is not held to one size across the connecting state; " +
				"a spinner and a symbol do not measure the same, and the toolbar grows with the taller")
		}
	})
}

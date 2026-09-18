package archtest_test

import (
	"regexp"
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

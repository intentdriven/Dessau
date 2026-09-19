package discord

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// A channel's history is bounded by the window that channel's model is served
// at, judged with the gateway's own estimator — so the gateway never refuses
// a bridged conversation for its size. The oldest turns drop off; the newest
// stays.
func TestAConversationIsTrimmedToTheServedWindow(t *testing.T) {
	const served = 2048
	var turns []turn
	for i := range 200 {
		turns = append(turns, turn{Role: roleUser, Content: "turn " + strconv.Itoa(i) + " " + strings.Repeat("x", 200)})
	}
	turns[len(turns)-1].Content = "the newest thing said"

	body, err := buildRequest("a-model", turns, served)
	if err != nil {
		t.Fatal(err)
	}
	// The gateway's own arithmetic: the estimate plus the answer asked for
	// must be inside the window, or it would refuse this.
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	judged := int64(len(body)/bytesPerToken) + int64(req.MaxTokens)
	if judged > served {
		t.Errorf("the request would be judged at %d tokens against a window of %d — the gateway "+
			"would refuse a conversation the bridge was supposed to have trimmed", judged, served)
	}
	if len(req.Messages) == 0 {
		t.Fatal("the request carries no turns at all")
	}
	if last := req.Messages[len(req.Messages)-1].Content; last != "the newest thing said" {
		t.Errorf("the newest turn is %q; trimming drops the oldest, never the newest", last)
	}
	if len(req.Messages) >= len(turns) {
		t.Errorf("all %d turns were sent; the window is %d tokens", len(req.Messages), served)
	}
}

// A single message too large for the window on its own is truncated rather
// than refused: refusing would leave that person unable to say anything at all
// in that channel.
func TestASingleOversizeTurnIsTruncatedRatherThanRefused(t *testing.T) {
	const served = 1024
	body, err := buildRequest("a-model", []turn{{Role: roleUser, Content: strings.Repeat("y", 200_000)}}, served)
	if err != nil {
		t.Fatal(err)
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("the request carries %d turns", len(req.Messages))
	}
	judged := int64(len(body)/bytesPerToken) + int64(req.MaxTokens)
	if judged > served {
		t.Errorf("the truncated request would still be judged at %d against %d", judged, served)
	}
}

// A model this Mac cannot report a window for is given a small one, so the
// cost of not knowing is a shorter history rather than a refusal the person
// on Discord can do nothing about.
func TestAnUnknownWindowIsAssumedSmall(t *testing.T) {
	body, err := buildRequest("a-model", []turn{{Role: roleUser, Content: strings.Repeat("z", 200_000)}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(body)/bytesPerToken) > defaultWindow {
		t.Errorf("a request of about %d tokens was built for a model with no known window",
			len(body)/bytesPerToken)
	}
}

// The conversations a session holds are bounded. Anyone who can reach the bot
// may talk to it, so the number of channels is a stranger's to choose.
func TestTheNumberOfConversationsHeldIsBounded(t *testing.T) {
	c := newConversations()
	first := c.get("channel-0")
	first.setModel("the-first-model")
	for i := 1; i <= maxChannels+10; i++ {
		c.get("channel-" + strconv.Itoa(i))
	}
	c.mu.Lock()
	held := len(c.byID)
	c.mu.Unlock()
	if held > maxChannels {
		t.Errorf("the session holds %d conversations, and the bound is %d", held, maxChannels)
	}
	// The one that went is the least recently used, which for a channel
	// nobody is using means it starts afresh — exactly what /reset does.
	if got := c.get("channel-0").modelOf(); got == "the-first-model" {
		t.Error("the oldest conversation survived the bound")
	}
}

// A conversation keeps its own turns bounded too, so the slice cannot grow
// without limit on a model with a very large window.
func TestAConversationKeepsAtMostItsMaximumTurns(t *testing.T) {
	conv := &conversation{}
	for i := range maxTurns * 3 {
		conv.append(turn{Role: roleUser, Content: strconv.Itoa(i)})
	}
	if got := len(conv.history()); got != maxTurns {
		t.Errorf("the conversation holds %d turns, and the bound is %d", got, maxTurns)
	}
	if first := conv.history()[0].Content; first == "0" {
		t.Error("the oldest turn survived the bound")
	}
}

// `/reset` clears the history and keeps the model: it is "start this
// conversation again", not "undo what I chose".
func TestResetClearsTheHistoryAndKeepsTheModel(t *testing.T) {
	conv := &conversation{}
	conv.setModel("a-model")
	conv.append(turn{Role: roleUser, Content: "hello"})
	conv.reset()
	if len(conv.history()) != 0 {
		t.Error("the history survived a reset")
	}
	if conv.modelOf() != "a-model" {
		t.Errorf("the model became %q; a reset does not undo /model", conv.modelOf())
	}
}

// The cut lands on a paragraph when there is one, on a line or a space when
// there is not, and on the ceiling for text with none of the three — and never
// past Discord's limit.
func TestTheCutPrefersAParagraphAndNeverExceedsTheLimit(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want string
	}{
		{
			name: "a paragraph break",
			text: strings.Repeat("a", 1000) + "\n\n" + strings.Repeat("b", 1500),
			want: strings.Repeat("a", 1000),
		},
		{
			name: "a line break when there is no paragraph",
			text: strings.Repeat("a", 1000) + "\n" + strings.Repeat("b", 1500),
			want: strings.Repeat("a", 1000),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			head, tail := cutAtParagraph(tc.text)
			if head != tc.want {
				t.Errorf("the cut kept %d characters, want %d", len([]rune(head)), len([]rune(tc.want)))
			}
			if !strings.HasPrefix(tail, "b") {
				t.Errorf("the continuation starts %q", headRunes(tail, 10))
			}
		})
	}

	// A single very long word, or a language that does not space its words:
	// a message that cannot be sent is worse than one that breaks mid-word.
	head, tail := cutAtParagraph(strings.Repeat("c", 5000))
	if n := len([]rune(head)); n != messageLimit {
		t.Errorf("unbreakable text was cut at %d, want the ceiling of %d", n, messageLimit)
	}
	if len([]rune(tail)) != 3000 {
		t.Errorf("the continuation is %d characters", len([]rune(tail)))
	}

	// And the cut never splits a character in half, whatever the script.
	head, tail = cutAtParagraph(strings.Repeat("日", 5000))
	if !isWholeText(head) || !isWholeText(tail) {
		t.Error("the cut split a character")
	}
	if n := len([]rune(head)); n > messageLimit {
		t.Errorf("a message of %d characters was produced", n)
	}
}

func isWholeText(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

// An identifier is a snowflake or it is nothing. Everything the bridge puts
// into a URL or a log line goes through this.
func TestOnlyASnowflakeIsAcceptedAsAnIdentifier(t *testing.T) {
	for _, ok := range []string{"0", "444444444444444444", "99999999999999999999"} {
		if !validID(ok) {
			t.Errorf("validID(%q) = false", ok)
		}
	}
	for _, bad := range []string{
		"", "../../applications/1/commands", "4e4", " 444", "444 ",
		"999999999999999999999", "44444444444444444444444444", "4-4", "٤٤٤",
	} {
		if validID(bad) {
			t.Errorf("validID(%q) = true, and it is not a snowflake", bad)
		}
	}
}

package discord

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// opensOnTheUser fails unless a built request's history starts with a user
// turn and ends with the newest thing the person said. Gemma- and
// Mistral-style chat templates raise on a history that opens on the model's
// answer, and the channel's next message then fails (iss-2610032306154631).
func opensOnTheUser(t *testing.T, body []byte, newest func(string) bool) {
	t.Helper()
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Messages) == 0 {
		t.Fatal("the request carries no turns at all")
	}
	if first := req.Messages[0]; first.Role != roleUser {
		t.Errorf("the history sent opens on a %q turn (%q); it must open on the user", first.Role, headRunes(first.Content, 40))
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role != roleUser || !newest(last.Content) {
		t.Errorf("the history sent ends on a %q turn (%q), want the newest user message", last.Role, headRunes(last.Content, 40))
	}
}

func is(want string) func(string) bool { return func(got string) bool { return got == want } }

// The turn count is even and a question is appended before it is asked, so
// once a channel has had maxTurns/2 exchanges every trim drops exactly one
// turn: the oldest question, leaving its answer at the front. Driven end to
// end against the fake, so it is the history the gateway is actually handed.
func TestAHistoryTrimmedByTheTurnCountOpensOnTheUser(t *testing.T) {
	f := newFakeDiscord(t)
	b, asked := recording(t, f, "an answer")
	connected(t, f, b)

	exchanges := maxTurns/2 + 1
	for i := range exchanges {
		f.message("question "+strconv.Itoa(i), false, false)
		answered(t, f, b)
	}
	got := asked()
	if len(got) != exchanges {
		t.Fatalf("the gateway was asked %d times, want %d", len(got), exchanges)
	}
	body, err := json.Marshal(got[len(got)-1])
	if err != nil {
		t.Fatal(err)
	}
	opensOnTheUser(t, body, is("question "+strconv.Itoa(exchanges-1)))
}

// The store's byte budget drops the oldest turn across every channel, one at
// a time, so it can take a channel's question and leave its answer.
func TestAHistoryTrimmedByTheByteBudgetOpensOnTheUser(t *testing.T) {
	c := newConversations()
	conv := c.get("channel-0")
	conv.append(turn{Role: roleUser, Content: "the first question"})
	conv.append(turn{Role: roleAssistant, Content: "the first answer"})
	conv.append(turn{Role: roleUser, Content: "the second question"})

	// Fill the store to exactly its budget from other channels, then go one
	// byte over: the budget drops one turn, the oldest, which is channel-0's
	// first question.
	filler := 0
	add := func(content string) {
		c.get("channel-" + strconv.Itoa(1+filler%(maxChannels-1))).append(turn{Role: roleUser, Content: content})
		filler++
	}
	for room := maxStoreBytes - c.bytes(); room > 0; room = maxStoreBytes - c.bytes() {
		add(strings.Repeat("x", min(room, maxTurnBytes)))
	}
	add("x")
	if h := conv.history(); len(h) != 2 {
		t.Fatalf("channel-0 holds %d turns after the budget trimmed; the set-up meant to drop exactly one", len(h))
	}

	body, err := buildRequest("a-model", conv.history(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	opensOnTheUser(t, body, is("the second question"))
}

// buildRequest drops turns from the front until the body fits the window, so
// the window can cut between a question and its answer.
func TestAHistoryTrimmedToTheWindowOpensOnTheUser(t *testing.T) {
	const served = 1024 // half for the answer: a budget of 2,048 bytes
	turns := []turn{
		{Role: roleUser, Content: strings.Repeat("q", 3000)},
		{Role: roleAssistant, Content: "a short answer"},
		{Role: roleUser, Content: "the newest question"},
	}
	body, err := buildRequest("a-model", turns, served)
	if err != nil {
		t.Fatal(err)
	}
	opensOnTheUser(t, body, is("the newest question"))
}

// When nothing fits, only one turn is sent, truncated. That turn is the
// newest user message, never a trailing answer standing on its own.
func TestTheFallbackSendsTheNewestUserTurn(t *testing.T) {
	const served = 1024
	question := "the newest question " + strings.Repeat("q", 200_000)
	turns := []turn{
		{Role: roleUser, Content: question},
		{Role: roleAssistant, Content: strings.Repeat("a", 200_000)},
	}
	body, err := buildRequest("a-model", turns, served)
	if err != nil {
		t.Fatal(err)
	}
	opensOnTheUser(t, body, func(got string) bool { return got != "" && strings.HasSuffix(question, got) })
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Messages) != 1 {
		t.Errorf("the fallback sent %d turns, want the one", len(req.Messages))
	}
	if judged := int64(len(body)/bytesPerToken) + int64(req.MaxTokens); judged > served {
		t.Errorf("the fallback would be judged at %d tokens against a window of %d", judged, served)
	}
}

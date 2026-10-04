package discord

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/gateway"
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

// alternates fails unless a built request's turns take strict turns: user,
// assistant, user, and so on, ending on the user. Gemma- and Mistral-style
// chat templates raise on two user turns in a row (iss-2610042030092101).
func alternates(t *testing.T, req request) {
	t.Helper()
	for i, m := range req.Messages {
		want := roleUser
		if i%2 == 1 {
			want = roleAssistant
		}
		if m.Role != want {
			t.Errorf("turn %d of %d is a %q turn (%q), want %q: the history sent does not alternate",
				i, len(req.Messages), m.Role, headRunes(m.Content, 40), want)
		}
	}
	if n := len(req.Messages); n == 0 || req.Messages[n-1].Role != roleUser {
		t.Error("the history sent does not end on the user")
	}
}

// A request refused with nothing written, and an answer that comes back
// empty, each leave the question they appended unanswered. The channel's
// next message must still send a history that alternates, or a
// strict-alternation template refuses it and every message after it, and the
// channel is stuck until /reset (iss-2610042030092101). Driven end to end
// against the fake, so it is the history the gateway is actually handed.
func TestAnUnansweredQuestionDoesNotBreakTheAlternation(t *testing.T) {
	f := newFakeDiscord(t)
	var mu sync.Mutex
	var asked []request
	opts := f.options(time.Now)
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		var body request
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return err
		}
		mu.Lock()
		asked = append(asked, body)
		n := len(asked)
		mu.Unlock()
		switch n {
		case 2:
			return errors.New("refused before a token was written")
		case 3:
			return nil // an answer of nothing at all
		}
		payload, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{"content": "answer " + strconv.Itoa(n-1)}}},
		})
		req.OnEvent(payload)
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	for i := range 4 {
		f.message("question "+strconv.Itoa(i), false, false)
		answered(t, f, b)
	}
	mu.Lock()
	got := append([]request(nil), asked...)
	mu.Unlock()
	if len(got) != 4 {
		t.Fatalf("the gateway was asked %d times, want 4", len(got))
	}
	last := got[3]
	alternates(t, last)
	want := []turn{
		{Role: roleUser, Content: "question 0"},
		{Role: roleAssistant, Content: "answer 0"},
		{Role: roleUser, Content: "question 3"},
	}
	if !slices.Equal(last.Messages, want) {
		t.Errorf("the history sent is %v, want %v: the unanswered questions go, the newest stays", last.Messages, want)
	}
}

// buildRequest holds the alternation whatever the store hands it: every run
// of user turns is cut to its newest.
func TestOnlyTheNewestOfConsecutiveUserTurnsIsSent(t *testing.T) {
	turns := []turn{
		{Role: roleUser, Content: "unanswered 1"},
		{Role: roleUser, Content: "question 1"},
		{Role: roleAssistant, Content: "answer 1"},
		{Role: roleUser, Content: "unanswered 2"},
		{Role: roleUser, Content: "unanswered 3"},
		{Role: roleUser, Content: "question 4"},
	}
	body, err := buildRequest("a-model", turns, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	alternates(t, req)
	want := []turn{
		{Role: roleUser, Content: "question 1"},
		{Role: roleAssistant, Content: "answer 1"},
		{Role: roleUser, Content: "question 4"},
	}
	if !slices.Equal(req.Messages, want) {
		t.Errorf("the history sent is %v, want %v", req.Messages, want)
	}
}

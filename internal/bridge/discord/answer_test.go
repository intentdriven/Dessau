package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Gropius/internal/gateway"
)

// connected starts a bridge against the fake and waits until it is answering.
func connected(t *testing.T, f *fakeDiscord, b *Bridge) {
	t.Helper()
	b.Apply(true, "a-token")
	f.waitFrame(opIdentify)
	waitState(t, b, StateConnected)
	f.waitCall(http.MethodPut, "/commands")
	f.drainCalls()
}

// A direct message is answered, and the answer is the model's text.
func TestADirectMessageIsAnswered(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "the ", "answer")
	connected(t, f, b)

	f.message("hello there", false, false)
	call := f.waitCall(http.MethodPost, "/channels/"+channelID+"/messages")
	if got, _ := call.Body["content"].(string); !strings.Contains(got, "the ") {
		t.Errorf("the bot posted %q, want the model's answer", got)
	}
	if call.Auth != "Bot a-token" {
		t.Errorf("the REST call carried %q, want Discord's bot scheme", call.Auth)
	}
}

// In a channel, only a message that mentions the bot is answered — and the
// mention is Discord's own resolved list, not text that looks like one. This
// is why the privileged message-content intent is never asked for.
func TestAChannelMessageIsAnsweredOnlyWhenItMentionsTheBot(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "hello")
	connected(t, f, b)

	// Text that spells a mention is not one: Discord decides, not the text.
	f.message("<@"+botUserID+"> answer me", true, false)
	f.noCall()

	f.message("what do you think", true, true)
	call := f.waitCall(http.MethodPost, "/messages")
	if got, _ := call.Body["content"].(string); got != "hello" {
		t.Errorf("the bot posted %q", got)
	}
}

// A message from another bot, or from ourselves, is never answered: two bots
// answering each other is a loop that runs until somebody notices.
func TestABotIsNeverAnswered(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "hello")
	connected(t, f, b)

	f.dispatch("MESSAGE_CREATE", map[string]any{
		"id": "900000000000000002", "channel_id": channelID, "content": "hi", "type": 0,
		"author": map[string]any{"id": "666666666666666666", "bot": true},
	})
	f.noCall()

	f.dispatch("MESSAGE_CREATE", map[string]any{
		"id": "900000000000000003", "channel_id": channelID, "content": "hi", "type": 0,
		"author": map[string]any{"id": botUserID, "bot": false},
	})
	f.noCall()
}

// An identifier that is not a snowflake is not a channel to post into. It
// arrives over the network and everything the bridge puts in a URL goes
// through the same check.
func TestAChannelIdentifierThatIsNotANumberIsNotAnswered(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "hello")
	connected(t, f, b)

	for _, bad := range []string{"../../applications/x", "44444444444444444444444444", "", "4e4"} {
		f.dispatch("MESSAGE_CREATE", map[string]any{
			"id": "900000000000000004", "channel_id": bad, "content": "hi", "type": 0,
			"author": map[string]any{"id": humanID, "bot": false},
		})
	}
	f.noCall()
}

// The bot is shown typing before anything is written, so somebody waiting on
// a cold model sees that it is working.
func TestTypingIsShownBeforeTheFirstToken(t *testing.T) {
	f := newFakeDiscord(t)
	opts := f.options(time.Now)
	release := make(chan struct{})
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		<-release
		payload, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{"content": "hi"}}},
		})
		req.OnEvent(payload)
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	f.message("hello", false, false)
	call := f.waitCall(http.MethodPost, "/typing")
	if !strings.Contains(call.Path, channelID) {
		t.Errorf("typing was shown in %q", call.Path)
	}
	close(release)
	f.waitCall(http.MethodPost, "/messages")
}

// Nothing the bot posts may ping anybody. The answer is a model's text and may
// say anything at all, so the defence is telling Discord to resolve no
// mentions in it rather than trying to scrub the text.
func TestAnAnswerCanPingNobody(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "@everyone look at this")
	connected(t, f, b)

	f.message("hello", false, false)
	call := f.waitCall(http.MethodPost, "/messages")
	allowed, ok := call.Body["allowed_mentions"].(map[string]any)
	if !ok {
		t.Fatalf("the post carried no allowed_mentions: %s", call.Raw)
	}
	parse, ok := allowed["parse"].([]any)
	if !ok || len(parse) != 0 {
		t.Errorf("allowed_mentions.parse = %v, want an empty list so nothing is pinged", allowed["parse"])
	}
}

// A long answer is cut at a paragraph before Discord's ceiling and continues
// in a second message; no message exceeds the limit.
func TestALongAnswerIsCutAtAParagraphAndContinues(t *testing.T) {
	f := newFakeDiscord(t)
	para := strings.Repeat("word ", 300) // ~1,500 characters
	b := answering(t, f, para, "\n\n", para)
	connected(t, f, b)

	f.message("write me an essay", false, false)
	first := f.waitCall(http.MethodPost, "/messages")
	second := f.waitCall(http.MethodPost, "/messages")

	for _, call := range []restCall{first, second} {
		text, _ := call.Body["content"].(string)
		if n := len([]rune(text)); n == 0 || n > messageLimit {
			t.Errorf("a message of %d characters was posted; Discord's ceiling is %d", n, messageLimit)
		}
	}
	firstText, _ := first.Body["content"].(string)
	if strings.HasSuffix(firstText, "wor") {
		t.Errorf("the first message was cut mid-word: %q", firstText[len(firstText)-20:])
	}
}

// The editor edits no more often than every two seconds, whatever the model
// does. A token-by-token edit would be a request per token to somebody else's
// API.
func TestTheEditorEditsNoMoreOftenThanEveryTwoSeconds(t *testing.T) {
	f := newFakeDiscord(t)
	var mu sync.Mutex
	now := time.Now()
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	opts := f.options(clock)
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		for range 40 {
			payload, _ := json.Marshal(map[string]any{
				"choices": []any{map[string]any{"delta": map[string]any{"content": "tok "}}},
			})
			req.OnEvent(payload)
		}
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	f.message("hello", false, false)
	// The answer is over in a moment of frozen time, so exactly one create
	// and one final edit may go out: the throttle allows nothing between them.
	f.waitCall(http.MethodPost, "/messages")
	time.Sleep(quiet)
	edits := 0
	for {
		select {
		case c := <-f.calls:
			if c.Method == http.MethodPatch {
				edits++
			}
			continue
		default:
		}
		break
	}
	if edits > 1 {
		t.Errorf("the editor made %d edits within one frozen instant; the floor is one every %s",
			edits, minEditInterval)
	}
}

// The rate-limit headers raise the floor. What Discord says about its own
// buckets is obeyed, and the two-second floor is only that — a floor.
func TestTheRateLimitHeadersRaiseTheEditingFloor(t *testing.T) {
	var limit rateLimit
	limit = readRateLimit(http.Header{
		"X-Ratelimit-Remaining":   []string{"0"},
		"X-Ratelimit-Reset-After": []string{"7.5"},
	})
	if !limit.exhausted || limit.resetAfter != 7500*time.Millisecond {
		t.Fatalf("read %v from an emptied bucket", limit)
	}
	if got := nextInterval(limit); got != 7500*time.Millisecond {
		t.Errorf("the next edit waits %s, want what Discord asked for", got)
	}
	// A bucket with room left is the floor and nothing more.
	if got := nextInterval(readRateLimit(http.Header{"X-Ratelimit-Remaining": []string{"4"}})); got != minEditInterval {
		t.Errorf("the next edit waits %s, want the %s floor", got, minEditInterval)
	}
	// And a figure Discord names beyond all reason is clamped rather than
	// parking a worker for as long as the other end likes.
	long := readRateLimit(http.Header{
		"X-Ratelimit-Remaining":   []string{"0"},
		"X-Ratelimit-Reset-After": []string{"100000"},
	})
	if long.resetAfter > maxRetryAfter {
		t.Errorf("a reset of %s was believed; it is clamped at %s", long.resetAfter, maxRetryAfter)
	}
}

// `/reset` clears the channel's conversation, and `/model` sets what answers
// it. Both are answered inside Discord's three seconds, which means over
// HTTPS rather than queued behind an answer.
func TestTheSlashCommandsAnswerTheChannel(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "hello")
	connected(t, f, b)

	f.command(commandReset, nil)
	call := f.waitCall(http.MethodPost, "/interactions/")
	data, _ := call.Body["data"].(map[string]any)
	if text, _ := data["content"].(string); !strings.Contains(text, "cleared") {
		t.Errorf("/reset answered %q", text)
	}

	f.command(commandModel, nil)
	call = f.waitCall(http.MethodPost, "/interactions/")
	data, _ = call.Body["data"].(map[string]any)
	if text, _ := data["content"].(string); !strings.Contains(text, "mlx-community/Qwen3-8B-4bit") {
		t.Errorf("/model with no argument answered %q, want the channel's model", text)
	}

	f.command(commandModel, map[string]string{"name": "not-a-model-here"})
	call = f.waitCall(http.MethodPost, "/interactions/")
	data, _ = call.Body["data"].(map[string]any)
	text, _ := data["content"].(string)
	if !strings.Contains(text, "not a model this server offers") {
		t.Errorf("/model with an unknown name answered %q", text)
	}
	if strings.Contains(text, "not-a-model-here") {
		t.Error("the refusal echoed the name back, which puts a stranger's text in a message this bot posts")
	}
}

// The log records that a request was bridged, with the identifiers as plain
// numbers — and carries nothing of the message or the answer.
//
// This is adr-2609181004167097 condition 4 exercised rather than scanned: the
// archtest walks the source, and this walks what the source actually writes.
func TestNothingOfAMessageOrAnAnswerReachesTheLog(t *testing.T) {
	f := newFakeDiscord(t)
	const secretInTheMessage = "zebrafish-tangerine"
	const secretInTheAnswer = "narwhal-clementine"

	var buf lockedBuffer
	opts := f.options(time.Now)
	opts.Log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		payload, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{"content": secretInTheAnswer}}},
		})
		req.OnEvent(payload)
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	f.message("please remember "+secretInTheMessage, false, false)
	f.waitCall(http.MethodPost, "/messages")
	time.Sleep(quiet)

	written := buf.String()
	if !strings.Contains(written, "bridged a request") {
		t.Fatalf("the bridge recorded nothing about the request it relayed:\n%s", written)
	}
	for _, secret := range []string{secretInTheMessage, secretInTheAnswer} {
		if strings.Contains(written, secret) {
			t.Errorf("the log carries %q, which is a message being relayed or an answer being posted:\n%s",
				secret, written)
		}
	}
	if strings.Contains(written, "a-token") {
		t.Errorf("the log carries the bot token:\n%s", written)
	}
	// The identifiers are numbers, not names.
	if !strings.Contains(written, "channel="+channelID) || !strings.Contains(written, "user="+humanID) {
		t.Errorf("the line does not carry the channel and user as numbers:\n%s", written)
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// One busy channel does not stop every other conversation.
//
// A generation runs for minutes and there are a small, fixed number of
// workers, so a second message in a channel that is already being answered
// must be recognised rather than waited on: anyone who can reach the bot may
// talk to it, and waiting would let one person stop the bot answering anybody
// by pressing send twice (iss-2609190058038096).
func TestABusyChannelDoesNotStopEveryOtherConversation(t *testing.T) {
	f := newFakeDiscord(t)
	opts := f.options(time.Now)
	hold := make(chan struct{})
	var held sync.Once
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		// The first answer blocks; every later one returns at once.
		first := false
		held.Do(func() { first = true })
		if first {
			<-hold
		}
		payload, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{"content": "an answer"}}},
		})
		req.OnEvent(payload)
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	// One long answer in flight in this channel.
	f.message("the first question", false, false)
	f.waitCall(http.MethodPost, "/typing")

	// More messages than there are workers, into the same channel. Each must
	// be answered and release its worker rather than parking on the lock.
	for range answerWorkers + 2 {
		f.message("and another", false, false)
	}
	for range answerWorkers + 2 {
		call := f.waitCall(http.MethodPost, "/channels/"+channelID+"/messages")
		if got, _ := call.Body["content"].(string); got != stillAnswering {
			t.Fatalf("a second message in a busy channel was answered with %q, want the busy reply", got)
		}
	}

	// And a different channel is still answered while the first is busy,
	// which is the property that actually matters.
	f.dispatch("MESSAGE_CREATE", map[string]any{
		"id": "900000000000000009", "channel_id": "666666666666666666",
		"content": "hello from somewhere else", "type": 0,
		"author": map[string]any{"id": humanID, "bot": false},
	})
	call := f.waitCall(http.MethodPost, "/channels/666666666666666666/messages")
	if got, _ := call.Body["content"].(string); got != "an answer" {
		t.Errorf("another channel was answered with %q while one channel was busy", got)
	}
	close(hold)
}

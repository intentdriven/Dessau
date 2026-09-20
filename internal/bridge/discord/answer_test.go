package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/gateway"
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

// answered waits for a message to be answered AND for the answer to be
// finished with.
//
// Waiting on the POST alone is not enough, and the trap is a real one: the
// placeholder is posted from inside the streaming callback, while the
// channel's answering lock is still held, so a test that sent its next message
// then would be met with "I am still answering your last message here" — which
// is a POST to the same path, and would satisfy the next wait as though it
// were an answer (iss-2609190312182409). The lock is what says the answer is
// over, so the lock is what is waited on.
func answered(t *testing.T, f *fakeDiscord, b *Bridge) restCall {
	t.Helper()
	call := f.waitCall(http.MethodPost, "/channels/"+channelID+"/messages")
	if got, _ := call.Body["content"].(string); got == stillAnswering {
		t.Fatalf("the bot replied %q: the channel was still answering the message before this one", got)
	}
	// Against the bridge's OWN store, or not at all: conversations() mints a
	// throwaway when the bridge holds none, and waiting on a lock in a store
	// nothing else can reach would synchronise on nothing and pass
	// (iss-2609190312182409's second half). Every caller runs on a live
	// bridge; this says so rather than trusting it.
	b.mu.Lock()
	store := b.convos
	b.mu.Unlock()
	if store == nil {
		t.Fatal("the bridge holds no conversations: it is not running, and this wait would mean nothing")
	}
	conv := store.get(channelID)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if conv.answering.TryLock() {
			conv.answering.Unlock()
			return call
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the channel was still answering five seconds after it posted")
	return call
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
	if text, _ := data["content"].(string); !strings.Contains(text, firstModel) {
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

// `/model` picking a model is the acceptance criterion, and the criterion is
// the NEXT MESSAGE being answered by it: the command stores the choice on the
// channel's conversation and the answer reads it back on the way to the
// gateway. Nothing exercised that seam while the fake server offered one
// model, because the model a request carried was the default either way
// (iss-2609190242018424).
func TestAModelPickedWithTheCommandAnswersTheNextMessage(t *testing.T) {
	f := newFakeDiscord(t)
	b, asked := recording(t, f, "an answer")
	connected(t, f, b)

	// Bob asks what the channel is on, and is told the server's default.
	f.command(commandModel, nil)
	call := f.waitCall(http.MethodPost, "/interactions/")
	data, _ := call.Body["data"].(map[string]any)
	if text, _ := data["content"].(string); !strings.Contains(text, firstModel) {
		t.Fatalf("/model answered %q, want the default the channel starts on", text)
	}

	// Then he picks the other one.
	f.command(commandModel, map[string]string{"name": secondModel})
	call = f.waitCall(http.MethodPost, "/interactions/")
	data, _ = call.Body["data"].(map[string]any)
	if text, _ := data["content"].(string); !strings.Contains(text, secondModel) {
		t.Fatalf("/model %s answered %q, want the model it was set to", secondModel, text)
	}

	f.message("what do you think", false, false)
	answered(t, f, b)
	got := asked()
	if len(got) != 1 {
		t.Fatalf("the gateway was asked %d times, want once", len(got))
	}
	if got[0].Model != secondModel {
		t.Errorf("the message was answered by %q, want the model `/model` picked (%q)", got[0].Model, secondModel)
	}

	// And the short name works the same way, which is how most people will
	// type one.
	f.command(commandModel, map[string]string{"name": shortName(firstModel)})
	f.waitCall(http.MethodPost, "/interactions/")
	f.drainCalls()
	f.message("and now", false, false)
	answered(t, f, b)
	got = asked()
	if len(got) != 2 {
		t.Fatalf("the gateway was asked %d times, want twice", len(got))
	}
	if got[1].Model != firstModel {
		t.Errorf("after picking by short name the message was answered by %q, want %q", got[1].Model, firstModel)
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

// An interaction token is a fifteen-minute credential that lets its holder
// post and edit messages as the bot, without the bot token. It is in the path
// of one REST call, and no error or log line the bridge writes may carry it
// (iss-2609190106273150).
func TestAnInteractionTokenNeverReachesAnErrorOrTheLog(t *testing.T) {
	const secretToken = "ephemeral-credential-0123456789"
	var buf lockedBuffer

	// A REST server that refuses everything, so every call produces the error
	// that would carry the path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := newREST(srv.URL, "a-bot-token", srv.Client(), log)
	err := r.respondToInteraction(context.Background(), "800000000000000001", secretToken, "hello")
	if err == nil {
		t.Fatal("a refused interaction produced no error")
	}
	if strings.Contains(err.Error(), secretToken) {
		t.Errorf("the error carries the interaction token: %v", err)
	}
	if !strings.Contains(err.Error(), "{token}") {
		t.Errorf("the error does not name the endpoint by its template: %v", err)
	}
	log.Debug("could not answer a slash command", "bridge", bridgeName, "err", err)
	if strings.Contains(buf.String(), secretToken) {
		t.Errorf("the log carries the interaction token:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "a-bot-token") {
		t.Errorf("the log carries the bot token:\n%s", buf.String())
	}
}

// A refusal is recorded by its class and never by its text.
//
// The gateway's refusal texts describe this Mac, and a stranger on Discord can
// drive a refusal at the rate they can send messages — so the text would be a
// description of this Mac that a stranger can write into a file
// (iss-2609190106273104).
func TestARefusedBridgedRequestIsLoggedByClassAndNotByItsText(t *testing.T) {
	f := newFakeDiscord(t)
	var buf lockedBuffer
	opts := f.options(time.Now)
	opts.Log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		// What the gateway hands a caller for a pool refusal: a fixed class
		// and a text of its own, with the describing text kept behind.
		return errors.New("every model that fits the 25165824000-byte budget is serving")
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	f.message("hello", false, false)
	f.waitCall(http.MethodPost, "/messages")
	time.Sleep(quiet)

	written := buf.String()
	if !strings.Contains(written, "refused a bridged request") {
		t.Fatalf("the refusal was not recorded:\n%s", written)
	}
	if strings.Contains(written, "25165824000") || strings.Contains(written, "budget") {
		t.Errorf("the log carries the gateway's own refusal text, which describes this Mac:\n%s", written)
	}
}

// A slash command is answered inside Discord's three seconds in a channel
// whose answer is still being written.
//
// Discord gives a bot three seconds to answer an interaction. An answer runs
// for minutes, so a command that waited on the same lock would miss the
// deadline, post into a token that had expired, and occupy one of the two
// workers while it did (iss-2609190106414499).
func TestASlashCommandIsAnsweredWhileTheChannelIsBusy(t *testing.T) {
	f := newFakeDiscord(t)
	opts := f.options(time.Now)
	hold := make(chan struct{})
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		<-hold
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	connected(t, f, b)

	f.message("a long question", false, false)
	f.waitCall(http.MethodPost, "/typing")

	started := time.Now()
	f.command(commandReset, nil)
	call := f.waitCall(http.MethodPost, "/interactions/")
	if took := time.Since(started); took > 3*time.Second {
		t.Errorf("/reset took %s in a busy channel; Discord's deadline is three seconds", took)
	}
	data, _ := call.Body["data"].(map[string]any)
	if text, _ := data["content"].(string); !strings.Contains(text, "cleared") {
		t.Errorf("/reset answered %q", text)
	}
	close(hold)
}

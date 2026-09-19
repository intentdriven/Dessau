package discord

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/intentdriven/Gropius/internal/gateway"
)

// answering wires a bridge to the fake with a completion path that streams
// the given text back one piece at a time.
func answering(t *testing.T, f *fakeDiscord, pieces ...string) *Bridge {
	t.Helper()
	opts := f.options(time.Now)
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		for _, p := range pieces {
			payload, _ := json.Marshal(map[string]any{
				"choices": []any{map[string]any{"delta": map[string]any{"content": p}}},
			})
			req.OnEvent(payload)
		}
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	return b
}

// recording is answering with the requests kept: the completion path records
// the body it was handed before streaming the answer back, so a test can say
// which model answered a message and what history was sent with it.
func recording(t *testing.T, f *fakeDiscord, pieces ...string) (*Bridge, func() []request) {
	t.Helper()
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
		mu.Unlock()
		for _, p := range pieces {
			payload, _ := json.Marshal(map[string]any{
				"choices": []any{map[string]any{"delta": map[string]any{"content": p}}},
			})
			req.OnEvent(payload)
		}
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	return b, func() []request {
		mu.Lock()
		defer mu.Unlock()
		return append([]request(nil), asked...)
	}
}

// The bridge is off until the switch is thrown, which is condition 1 of
// adr-2609181004167097 stated as a property of the code: nothing leaves this
// Mac until the operator has pasted a token and turned it on.
func TestTheBridgeOpensNothingUntilItIsSwitchedOn(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f)

	b.Apply(false, "a-token")
	time.Sleep(quiet)
	if f.dials() != 0 {
		t.Fatalf("the bridge dialled %d times with the switch off", f.dials())
	}
	if state, _, _ := b.State(); state != StateOff {
		t.Errorf("state = %q, want %q", state, StateOff)
	}

	// The switch on with no token is not a connection either, and not a
	// failure: it is a state the panel shows.
	b.Apply(true, "")
	time.Sleep(quiet)
	if f.dials() != 0 {
		t.Fatalf("the bridge dialled %d times with no token", f.dials())
	}
	state, reason := waitState(t, b, StateStopped)
	if state != StateStopped || reason == "" {
		t.Errorf("state = %q (%q), want stopped with a reason", state, reason)
	}
}

// Identify carries the token and exactly the two unprivileged intents. Asking
// for MESSAGE_CONTENT would make the bridge a reader of every message in every
// channel it is invited to; it is never asked for
// (itd-2609180959397172).
func TestTheSessionIdentifiesWithTheTwoUnprivilegedIntents(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f)
	b.Apply(true, "a-token")

	fr := f.waitFrame(opIdentify)
	var d struct {
		Token   string `json:"token"`
		Intents int    `json:"intents"`
	}
	if err := json.Unmarshal(fr.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d.Token != "a-token" {
		t.Errorf("Identify carried token %q", d.Token)
	}
	const guildMessages, directMessages = 1 << 9, 1 << 12
	if d.Intents != guildMessages|directMessages {
		t.Errorf("Identify asked for intents %d, want exactly GUILD_MESSAGES|DIRECT_MESSAGES (%d)",
			d.Intents, guildMessages|directMessages)
	}
	const messageContent = 1 << 15
	if d.Intents&messageContent != 0 {
		t.Error("Identify asked for the privileged message-content intent")
	}
	waitState(t, b, StateConnected)
}

// A session that drops is resumed, not started again: the resume carries
// Discord's session id and the last sequence seen. This is the path a
// sleeping Mac takes.
func TestADroppedSessionIsResumedWithTheSequenceItHadSeen(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f)
	b.Apply(true, "a-token")
	f.waitFrame(opIdentify)
	waitState(t, b, StateConnected)

	// A dispatch, so there is a sequence to resume from.
	f.message("hello", false, false)
	f.waitCall(http.MethodPost, "/messages")

	f.drop()
	fr := f.waitFrame(opResume)
	var d struct {
		Token     string `json:"token"`
		SessionID string `json:"session_id"`
		Seq       int64  `json:"seq"`
	}
	if err := json.Unmarshal(fr.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d.SessionID != "sess-1" {
		t.Errorf("Resume carried session id %q, want the one READY gave", d.SessionID)
	}
	if d.Seq <= 0 {
		t.Errorf("Resume carried seq %d, want the last sequence the session saw", d.Seq)
	}
	if d.Token != "a-token" {
		t.Errorf("Resume carried token %q", d.Token)
	}
	waitState(t, b, StateConnected)
}

// The bridge heartbeats on the interval Hello names, carrying the last
// sequence.
func TestTheSessionHeartbeatsOnTheIntervalHelloNames(t *testing.T) {
	f := newFakeDiscord(t)
	f.heartbeatMS = 40
	b := answering(t, f)
	b.Apply(true, "a-token")
	f.waitFrame(opIdentify)
	f.waitFrame(opHeartbeat)
	f.waitFrame(opHeartbeat)
}

// 4004 is Discord refusing the token and 4014 is Discord refusing the
// intents. Neither comes right by being retried, so both stop the bridge and
// put the reason where the operator can read it.
func TestARefusedTokenStopsTheBridgeWithTheReasonOnThePanel(t *testing.T) {
	for _, tc := range []struct {
		name string
		code websocket.StatusCode
		says string
	}{
		{"a refused token", 4004, "token"},
		{"refused intents", 4014, "intents"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeDiscord(t)
			f.closeAfterIdentify = tc.code
			b := answering(t, f)
			b.Apply(true, "a-token")

			state, reason := waitState(t, b, StateStopped)
			if state != StateStopped {
				t.Fatalf("state = %q", state)
			}
			if reason == "" || !strings.Contains(reason, tc.says) {
				t.Errorf("the panel is told %q, want a reason naming the %s", reason, tc.says)
			}
			// And it stays stopped: a credential Discord refuses is not
			// weather, so nothing retries it into a connection loop.
			dials := f.dials()
			time.Sleep(quiet)
			if f.dials() != dials {
				t.Errorf("the bridge kept dialling after a refusal: %d then %d", dials, f.dials())
			}
		})
	}
}

// Switching the bridge off closes the connection, which is the second half of
// condition 1: an opt-in that cannot be opted out of is not one.
func TestSwitchingTheBridgeOffClosesTheConnection(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "hi")
	b.Apply(true, "a-token")
	f.waitFrame(opIdentify)
	waitState(t, b, StateConnected)
	// The slash commands are registered once per start, on a worker; drained
	// here so the assertion below is about messages and not about that.
	f.waitCall(http.MethodPut, "/commands")

	b.Apply(false, "")
	if state, _, _ := b.State(); state != StateOff {
		t.Fatalf("state = %q, want %q", state, StateOff)
	}
	// Nothing is answered any more.
	f.drainCalls()
	f.message("are you there", false, false)
	f.noCall()
}

// A resume URL that would move the session onto a plaintext socket is
// discarded. It arrives over the network from the other end, and it is the one
// value that could talk this connection — token and all — somewhere else.
func TestAResumeMayNotDowngradeTheTransport(t *testing.T) {
	for _, tc := range []struct {
		raw, base, want string
	}{
		{"wss://gateway.discord.gg", "wss://gateway.discord.gg/?v=10", "wss://gateway.discord.gg/?v=10&encoding=json"},
		// Discord's own resume URLs are hosts under its domains.
		{"wss://gateway-us-east1-b.discord.gg", "wss://gateway.discord.gg/?v=10", "wss://gateway-us-east1-b.discord.gg/?v=10&encoding=json"},
		// A downgrade, refused.
		{"ws://attacker.example", "wss://gateway.discord.gg/?v=10", ""},
		{"http://attacker.example", "wss://gateway.discord.gg/?v=10", ""},
		{"", "wss://gateway.discord.gg/?v=10", ""},
		// And a redirection, which is the one field that moves the credential
		// (iss-2609190106410918).
		{"wss://attacker.example", "wss://gateway.discord.gg/?v=10", ""},
		{"wss://discord.gg.attacker.example", "wss://gateway.discord.gg/?v=10", ""},
		{"wss://attacker.example/?host=discord.gg", "wss://gateway.discord.gg/?v=10", ""},
		// The configured host itself is always allowed, which is what a test
		// or a local proxy relies on.
		{"ws://127.0.0.1:1", "ws://127.0.0.1:1/?v=10", "ws://127.0.0.1:1/?v=10&encoding=json"},
		{"ws://127.0.0.1:2", "ws://127.0.0.1:1/?v=10", ""},
	} {
		if got := resumeURL(tc.raw, tc.base); got != tc.want {
			t.Errorf("resumeURL(%q, %q) = %q, want %q", tc.raw, tc.base, got, tc.want)
		}
	}
}

// The heartbeat interval named in Hello arrives over the network, and it is
// bounded as the integer it arrives as.
//
// A figure large enough to overflow int64 nanoseconds becomes a NEGATIVE
// duration, which a ceiling check alone waves through and which then panics
// in the jitter's Int64N — one crafted frame taking the whole process down
// (iss-2609190105084881). A figure of one millisecond is the other half: it
// passes every check and spins the heartbeat goroutine.
func TestTheHeartbeatIntervalIsBoundedAtBothEnds(t *testing.T) {
	for _, ms := range []int64{
		math.MaxInt64, math.MaxInt64 / 1000, 1 << 40, 1e13,
		-1, 0, 1, 40, 45_000, 1 << 62,
	} {
		got := heartbeatInterval(ms)
		if got < minHeartbeatIntervalMS*time.Millisecond {
			t.Errorf("heartbeatInterval(%d) = %s, which is under the floor — a figure off the "+
				"network became one this process would spin or panic on", ms, got)
		}
		if got > maxHeartbeatIntervalMS*time.Millisecond {
			t.Errorf("heartbeatInterval(%d) = %s, which is over the ceiling", ms, got)
		}
	}
	if got := heartbeatInterval(45_000); got != 45*time.Second {
		t.Errorf("a figure inside the range came out as %s", got)
	}
}

// And the whole way through: a gateway whose Hello names an overflowing
// interval is survived rather than crashed on.
func TestAHostileHelloDoesNotTakeTheProcessDown(t *testing.T) {
	f := newFakeDiscord(t)
	f.heartbeatMS = math.MaxInt64
	b := answering(t, f)
	b.Apply(true, "a-token")
	f.waitFrame(opIdentify)
	waitState(t, b, StateConnected)
}

// A mention is still answered after the session has resumed.
//
// Discord sends READY on an Identify and RESUMED on a resume, and only READY
// carries the bot's own user id — so an identity held on the connection is
// lost the first time a Wi-Fi blip or a lid close makes the bridge resume,
// and every mention check answers false from then on. The bridge goes on
// answering direct messages and silently ignores every mention in every
// channel, with nothing in the log and the panel reading connected
// (iss-2609190106402401).
func TestAMentionIsStillAnsweredAfterAResume(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f, "an answer")
	b.Apply(true, "a-token")
	f.waitFrame(opIdentify)
	waitState(t, b, StateConnected)
	f.waitCall(http.MethodPut, "/commands")
	f.drainCalls()

	// Answered before the drop.
	f.message("what do you think", true, true)
	f.waitCall(http.MethodPost, "/channels/"+channelID+"/messages")

	f.drop()
	f.waitFrame(opResume)
	waitState(t, b, StateConnected)
	f.drainCalls()

	// And after it. This is the assertion the bug was hiding behind: the
	// direct-message path never needed the id, so the suite was green.
	f.message("and now", true, true)
	call := f.waitCall(http.MethodPost, "/channels/"+channelID+"/messages")
	if got, _ := call.Body["content"].(string); got != "an answer" {
		t.Errorf("after a resume a mention was answered with %q", got)
	}
}

// A token pasted with a newline on it is the ordinary result of copying one
// from a browser or a terminal. It is trimmed rather than sent as it stands
// and refused with a message that will fail again for the same invisible
// reason (iss-2609190106563320).
func TestAPastedTokenIsTrimmed(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f)
	b.Apply(true, "  a-token\n")

	fr := f.waitFrame(opIdentify)
	var d struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(fr.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d.Token != "a-token" {
		t.Errorf("Identify carried %q, want the token without the whitespace around it", d.Token)
	}

	// And a token that is only whitespace is no token at all, so nothing is
	// opened and the panel says why.
	b2 := answering(t, newFakeDiscord(t))
	b2.Apply(true, "   \n\t ")
	if _, reason := waitState(t, b2, StateStopped); reason == "" {
		t.Error("a token of nothing but whitespace produced no reason on the panel")
	}
}

// A channel's conversation and the model it is on survive a reconnect.
//
// The bridge resumes by itself after the Mac sleeps (ac-10 of
// itd-2609180959397172), and the intent's scope condition says a conversation
// is gone when the BRIDGE STOPS — not when the socket does. Holding the
// conversations on the session made every drop a silent `/reset` that also
// forgot the model Bob had chosen, with nothing said in the channel
// (iss-2609190241509478).
func TestAChannelKeepsItsConversationAndModelAcrossAReconnect(t *testing.T) {
	f := newFakeDiscord(t)
	b, asked := recording(t, f, "an answer")
	connected(t, f, b)

	// Bob puts the channel on the second model and asks something.
	f.command(commandModel, map[string]string{"name": secondModel})
	f.waitCall(http.MethodPost, "/interactions/")
	f.message("first question", false, false)
	answered(t, f, b)

	// The socket drops and the session is resumed, which is the sleeping
	// Mac's own path.
	f.drop()
	f.waitFrame(opResume)
	waitState(t, b, StateConnected)
	f.drainCalls()

	f.message("second question", false, false)
	answered(t, f, b)

	got := asked()
	if len(got) != 2 {
		t.Fatalf("the gateway was asked %d times, want once before the reconnect and once after", len(got))
	}
	if got[1].Model != secondModel {
		t.Errorf("after the reconnect the channel was answered by %q, want the model `/model` chose (%q)",
			got[1].Model, secondModel)
	}
	if len(got[1].Messages) < 2 || got[1].Messages[0].Content != "first question" {
		t.Errorf("after the reconnect the request carried %v, want the channel's history in front of the new message",
			got[1].Messages)
	}
	if last := got[1].Messages[len(got[1].Messages)-1]; last.Content != "second question" {
		t.Errorf("the newest turn was %q, want the message that was just sent", last.Content)
	}
}

// And it goes when the bridge stops, which is the bound the intent actually
// asks for: nothing of a conversation outlives the switch.
func TestStoppingTheBridgeForgetsEveryChannelsConversation(t *testing.T) {
	f := newFakeDiscord(t)
	b, asked := recording(t, f, "an answer")
	connected(t, f, b)

	f.command(commandModel, map[string]string{"name": secondModel})
	f.waitCall(http.MethodPost, "/interactions/")
	f.message("first question", false, false)
	answered(t, f, b)

	b.Apply(false, "")
	if state, _, _ := b.State(); state != StateOff {
		t.Fatalf("state = %q, want %q", state, StateOff)
	}
	connected(t, f, b)

	f.message("after the restart", false, false)
	answered(t, f, b)

	got := asked()
	if len(got) != 2 {
		t.Fatalf("the gateway was asked %d times, want once before the switch and once after", len(got))
	}
	if got[1].Model != firstModel {
		t.Errorf("after a stop and a start the channel was answered by %q, want the server default (%q)",
			got[1].Model, firstModel)
	}
	if len(got[1].Messages) != 1 || got[1].Messages[0].Content != "after the restart" {
		t.Errorf("after a stop and a start the request carried %v, want the new message alone", got[1].Messages)
	}
}

// A socket that drops says so on the panel, and keeps the moment it last
// connected.
//
// The eleventh acceptance criterion of itd-2609180959397172 is that a dropped
// session resumes by itself and the panel shows when it last connected. The
// state only moved to connecting when the next attempt was made, which is up
// to half a minute of backoff later — so for that whole window the panel went
// on reading "connected since" for a session that was gone
// (iss-2609190242334438).
func TestADroppedSessionSaysSoAtOnceAndKeepsTheLastConnectedMoment(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f)
	connected(t, f, b)
	_, connectedAt, _ := b.State()
	if connectedAt.IsZero() {
		t.Fatal("a connected bridge reported no moment it connected at")
	}

	f.drop()

	// Well inside the shortest backoff: the panel is told the session is gone
	// when it goes, not when the next attempt is made.
	var state string
	var since time.Time
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state, since, _ = b.State()
		if state == StateConnecting {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if state != StateConnecting {
		t.Fatalf("a second after the socket dropped the panel still reads %q", state)
	}
	if !since.Equal(connectedAt) {
		t.Errorf("while reconnecting the bridge reports %v as its last connection, want the moment it did connect (%v)",
			since, connectedAt)
	}
}

// held is how many channels the bridge is keeping a conversation for, or -1
// when it holds no store at all.
func held(b *Bridge) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.convos == nil {
		return -1
	}
	b.convos.mu.Lock()
	defer b.convos.mu.Unlock()
	return len(b.convos.byID)
}

// A bridge Discord stops carries nothing of what was said to it.
//
// The switch is not the only way the bridge stops: a token revoked or
// regenerated, intents changed in the portal, or a READY too large to read
// end the run loop for good, with the reason on the panel. The conversations
// have to go on that path too — the goroutine is gone, no session exists, and
// what would be left is every stranger's message text held for the life of
// the process, with nothing the operator can do about it from Settings
// (iss-2609190312064731).
func TestAFatalStopForgetsEveryChannelsConversation(t *testing.T) {
	f := newFakeDiscord(t)
	b, _ := recording(t, f, "an answer")
	connected(t, f, b)
	f.message("a stranger's message", false, false)
	answered(t, f, b)
	if held(b) != 1 {
		t.Fatalf("the bridge holds %d conversations, want the one channel that was answered", held(b))
	}

	// 4004: Discord refusing the token, which is not weather.
	f.refuse(4004)
	if _, reason := waitState(t, b, StateStopped); reason == "" {
		t.Fatal("the bridge stopped with no reason on the panel")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if held(b) == -1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("Discord stopped the bridge and it is still holding %d channels' conversations", held(b))
}

package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
		{"ws://attacker.example", "wss://gateway.discord.gg/?v=10", ""},
		{"http://attacker.example", "wss://gateway.discord.gg/?v=10", ""},
		{"", "wss://gateway.discord.gg/?v=10", ""},
		{"ws://127.0.0.1:1", "ws://127.0.0.1:1/?v=10", "ws://127.0.0.1:1/?v=10&encoding=json"},
	} {
		if got := resumeURL(tc.raw, tc.base); got != tc.want {
			t.Errorf("resumeURL(%q, %q) = %q, want %q", tc.raw, tc.base, got, tc.want)
		}
	}
}

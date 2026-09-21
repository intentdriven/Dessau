package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/gateway"
)

// A model that keeps no transcript is not offered over a bridge
// (itd-2609091715089488, the 2026-09-20 decision): adr-2609181004167097
// admits the platform as a reader and keeper of the message and the answer,
// so Dessau would write no transcript while Discord kept one, and a panel
// saying "no transcript" beside that model would state what the product
// cannot honour. The exception is read per request through the seam the
// server fills with its folded reader, never remembered on the channel.

// excepting wires a bridge whose exception seam says what the set says at
// the moment it is asked, and records what the gateway was asked.
func excepting(t *testing.T, f *fakeDiscord, excepted *atomic.Value) (*Bridge, func() []request) {
	t.Helper()
	var mu sync.Mutex
	var asked []request
	opts := f.options(time.Now)
	opts.NoTranscript = func(model string) bool {
		set, _ := excepted.Load().(map[string]bool)
		return set[strings.ToLower(model)]
	}
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		var body request
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return err
		}
		mu.Lock()
		asked = append(asked, body)
		mu.Unlock()
		payload, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{"content": "an answer"}}},
		})
		req.OnEvent(payload)
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

func exceptedSet(ids ...string) *atomic.Value {
	var v atomic.Value
	set := map[string]bool{}
	for _, id := range ids {
		set[strings.ToLower(id)] = true
	}
	v.Store(set)
	return &v
}

// interactionReply is what the bot answered the last slash command with.
func interactionReply(t *testing.T, f *fakeDiscord) string {
	t.Helper()
	call := f.waitCall(http.MethodPost, "/interactions/")
	data, _ := call.Body["data"].(map[string]any)
	text, _ := data["content"].(string)
	return text
}

// Bare `/model` lists the models on offer, and an excepted model is not on
// offer: it is left out of the listing, and the listing says the models it
// does name are recorded — the word in the bridge's prose that is the
// picker's icon and the card's pill on the other two surfaces.
func TestTheModelCommandOmitsAnExceptedModel(t *testing.T) {
	f := newFakeDiscord(t)
	b, _ := excepting(t, f, exceptedSet(secondModel))
	connected(t, f, b)

	f.command(commandModel, nil)
	text := interactionReply(t, f)
	if !strings.Contains(text, firstModel) {
		t.Errorf("/model answered %q, want the recorded model listed", text)
	}
	if strings.Contains(text, secondModel) {
		t.Errorf("/model answered %q, which names a model that keeps no transcript", text)
	}
	if !strings.Contains(text, "recorded") {
		t.Errorf("/model answered %q, which does not say the listed models are recorded", text)
	}
}

// `/model` naming an excepted model — by repo id or by short name — is
// refused in the channel with the reason, and the channel stays on the model
// it had: the next message is answered by that one.
func TestTheModelCommandRefusesAnExceptedModel(t *testing.T) {
	f := newFakeDiscord(t)
	b, asked := excepting(t, f, exceptedSet(secondModel))
	connected(t, f, b)

	for _, name := range []string{secondModel, shortName(secondModel)} {
		f.command(commandModel, map[string]string{"name": name})
		text := interactionReply(t, f)
		if !strings.Contains(text, "keeps no transcript") || !strings.Contains(text, "Discord") {
			t.Errorf("/model %s answered %q, want the refusal with its reason", name, text)
		}
		if strings.Contains(text, "now answered by") {
			t.Errorf("/model %s answered %q: the excepted model was chosen", name, text)
		}
	}

	f.message("what do you think", false, false)
	answered(t, f, b)
	got := asked()
	if len(got) != 1 || got[0].Model != firstModel {
		t.Fatalf("the gateway was asked %+v, want once by the model the channel had (%s)", got, firstModel)
	}
}

// A channel already pointed at a model that is excepted afterwards is
// refused at its next message with the same reason, and the gateway is not
// asked: the exception is read per request, never at the time the channel
// chose, so a setting saved after `/model` still holds.
func TestAChannelExceptedAfterItChoseIsRefusedAtTheNextMessage(t *testing.T) {
	f := newFakeDiscord(t)
	excepted := exceptedSet()
	b, asked := excepting(t, f, excepted)
	connected(t, f, b)

	f.command(commandModel, map[string]string{"name": secondModel})
	if text := interactionReply(t, f); !strings.Contains(text, "now answered by") {
		t.Fatalf("/model %s answered %q, want it chosen while it is recorded", secondModel, text)
	}
	f.message("first", false, false)
	answered(t, f, b)
	if got := asked(); len(got) != 1 || got[0].Model != secondModel {
		t.Fatalf("the gateway was asked %+v, want once by %s", got, secondModel)
	}

	// Alice excepts the model in Settings.
	excepted.Store(map[string]bool{strings.ToLower(secondModel): true})
	f.drainCalls()
	f.message("second", false, false)
	call := f.waitCall(http.MethodPost, "/channels/"+channelID+"/messages")
	text, _ := call.Body["content"].(string)
	if !strings.Contains(text, "keeps no transcript") || !strings.Contains(text, "Discord") {
		t.Errorf("the channel was told %q, want the refusal with its reason", text)
	}
	if got := asked(); len(got) != 1 {
		t.Errorf("the gateway was asked %d times, want the excepted model never asked", len(got))
	}
}

// The model a channel starts on is the first on offer, and an excepted model
// is not on offer — so a server whose first model is excepted starts a
// channel on the next one rather than refusing the first message.
func TestAChannelStartsOnTheFirstModelOnOffer(t *testing.T) {
	f := newFakeDiscord(t)
	b, asked := excepting(t, f, exceptedSet(firstModel))
	connected(t, f, b)

	f.message("hello", false, false)
	answered(t, f, b)
	if got := asked(); len(got) != 1 || got[0].Model != secondModel {
		t.Fatalf("the gateway was asked %+v, want once by the first model on offer (%s)", got, secondModel)
	}
}

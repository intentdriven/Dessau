package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/gateway"
)

// A message admitted while every worker is busy is shown as typing within the
// two seconds the bridge promises, not when a worker frees up: the indicator
// goes up when the message is queued (the 2026-09-20 decision on
// iss-2609190242078205).
func TestAMessageWaitingForAWorkerIsShownTypingAtOnce(t *testing.T) {
	f := newFakeDiscord(t)
	opts := f.options(time.Now)
	release := make(chan struct{})
	opts.Ask = func(ctx context.Context, req gateway.AskRequest) error {
		select {
		case <-release:
		case <-ctx.Done():
			return ctx.Err()
		}
		payload, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{"content": "hi"}}},
		})
		req.OnEvent(payload)
		return nil
	}
	b := New(opts)
	t.Cleanup(func() { _ = b.Close() })
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	connected(t, f, b)

	// One channel per worker, each held answering, and then one more.
	channels := []string{"555555555555555551", "555555555555555552", "555555555555555553"}
	if len(channels) != answerWorkers+1 {
		t.Fatalf("the test needs one channel more than the %d workers", answerWorkers)
	}
	for i, ch := range channels {
		f.dispatch("MESSAGE_CREATE", map[string]any{
			"id": "90000000000000010" + string(rune('0'+i)), "channel_id": ch, "content": "hello", "type": 0,
			"author": map[string]any{"id": humanID, "bot": false}, "mentions": []any{},
		})
	}
	waiting := channels[len(channels)-1]
	deadline := time.After(2 * time.Second)
	for {
		select {
		case c := <-f.calls:
			if c.Method == http.MethodPost && strings.Contains(c.Path, "/channels/"+waiting+"/typing") {
				return
			}
		case <-deadline:
			t.Fatalf("the message waiting for a worker in %s was not shown typing within two seconds", waiting)
		}
	}
}

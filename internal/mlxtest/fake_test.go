package mlxtest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func post(t *testing.T, ctx context.Context, url string, body map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/chat/completions", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil
	}
	t.Cleanup(func() { resp.Body.Close() })
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func chat(content string) map[string]any {
	return map[string]any{
		"model":      "/models/m",
		"messages":   []map[string]string{{"role": "user", "content": content}},
		"max_tokens": 1, "stream": false,
	}
}

// With the knob on, the fake counts the prompt the way a tokenizer roughly
// would, so a bisection has something to converge on; and above a size it
// refuses, the way a server that ran out of room does.
func TestTheFakeCanCountThePromptAndRefuseAboveASize(t *testing.T) {
	s := Start(Options{ModelArg: "/models/m", PromptTokensFromBody: true, RefuseAbove: 100})
	t.Cleanup(s.Close)
	resp, out := post(t, context.Background(), s.URL(), chat(strings.Repeat("word ", 40)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a small prompt was refused: %d", resp.StatusCode)
	}
	usage, _ := out["usage"].(map[string]any)
	if pt, _ := usage["prompt_tokens"].(float64); pt < 40 || pt > 60 {
		t.Errorf("prompt_tokens = %v for 200 characters, want about 50", pt)
	}
	resp, _ = post(t, context.Background(), s.URL(), chat(strings.Repeat("word ", 400)))
	if resp.StatusCode == http.StatusOK {
		t.Error("a prompt above RefuseAbove was served")
	}
}

// Above HangAbove the fake never answers, which is what a prefill that
// outlasts the gateway's deadline looks like from outside; the response delay
// holds the non-streaming answer back for a measured time.
func TestTheFakeCanHangAndDelayOnTheNonStreamingPath(t *testing.T) {
	s := Start(Options{ModelArg: "/models/m", PromptTokensFromBody: true, HangAbove: 100, ResponseDelay: 50 * time.Millisecond})
	t.Cleanup(s.Close)
	started := time.Now()
	resp, _ := post(t, context.Background(), s.URL(), chat("short"))
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatal("a short prompt was not answered")
	}
	if time.Since(started) < 50*time.Millisecond {
		t.Error("the response delay was not honoured on the non-streaming path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if resp, _ := post(t, ctx, s.URL(), chat(strings.Repeat("word ", 400))); resp != nil {
		t.Errorf("a prompt above HangAbove was answered with %d", resp.StatusCode)
	}
}

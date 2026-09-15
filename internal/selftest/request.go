package selftest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// This file is the self-test's one reader and writer of a conversation, and
// it is on internal/archtest's allow-list for both sides of that boundary
// (prompt_content_test.go). It builds requests from the two constants below
// and reads nothing from a client: a self-test run is a conversation Gropius
// has with its own model server, and the only thing kept from the answer is
// when its first chunk arrived and how many chunks there were.

// The standard set. The names follow llama-bench — pp for prompt processing,
// tg for text generation, the number being the token count — so a figure here
// can be set beside a published one.
const (
	TestPP512 = "pp512"
	TestTG128 = "tg128"
	// TestTG128Parallel is tg128 sent Concurrency() times at once; the name
	// carries the count, e.g. "tg128x4".
	TestTG128Parallel = "tg128x"
)

// generationPrompt is what tg128 asks. It is short, so the answer is what is
// measured, and it asks for prose so the model does not stop early.
const generationPrompt = "Write a long story about a lighthouse keeper who finds a message in a bottle."

// promptSentence is repeated to make the pp512 prompt. Ordinary English at
// about four characters a token, so the prompt lands near 512 tokens on most
// tokenizers; the count the model server reports is what the result carries,
// never the target.
const promptSentence = "The lighthouse keeper climbed the spiral stairs every evening to light the lamp before the fishing boats came home. "

// promptProcessingChars is the length pp512's prompt is built to.
const promptProcessingChars = 512 * 4

// testSpec is one test of the set.
type testSpec struct {
	Name      string
	Prompt    string
	MaxTokens int
	Parallel  int
}

// standardSet is the tests every model gets, in the order they run. The
// concurrent test is only worth running when there is concurrency to measure.
func standardSet(concurrency int) []testSpec {
	set := []testSpec{
		{Name: TestPP512, Prompt: promptProcessingPrompt(), MaxTokens: 1},
		{Name: TestTG128, Prompt: generationPrompt, MaxTokens: 128},
	}
	if concurrency > 1 {
		set = append(set, testSpec{
			Name: fmt.Sprintf("%s%d", TestTG128Parallel, concurrency), Prompt: generationPrompt, MaxTokens: 128, Parallel: concurrency,
		})
	}
	return set
}

func promptProcessingPrompt() string {
	var b strings.Builder
	for b.Len() < promptProcessingChars {
		b.WriteString(promptSentence)
	}
	return b.String()
}

// measurement is what one request cost and how long it took.
type measurement struct {
	PromptTokens, CompletionTokens int
	FirstToken, Total              time.Duration
	// Counted says where the token counts came from: the model server's own
	// usage event, or — when it sent none — the number of chunks, which on
	// mlx-lm is one a token. A reader comparing two results should know.
	Counted string
}

// test turns a measurement into the recorded shape.
func (m measurement) test(name string, parallel int) Test {
	t := Test{
		Name:             name,
		Parallel:         parallel,
		PromptTokens:     m.PromptTokens,
		CompletionTokens: m.CompletionTokens,
		FirstTokenMs:     m.FirstToken.Milliseconds(),
		TotalMs:          m.Total.Milliseconds(),
		Counted:          m.Counted,
	}
	if m.FirstToken > 0 {
		t.PromptTokensPerSec = round1(float64(m.PromptTokens) / m.FirstToken.Seconds())
	}
	if gen := m.Total - m.FirstToken; gen > 0 && m.CompletionTokens > 1 {
		// The first token is paid for by the prefill, so it is the tokens after
		// it, over the time after it, that is the generation rate.
		t.TokensPerSec = round1(float64(m.CompletionTokens-1) / gen.Seconds())
	}
	return t
}

// aggregate folds the parallel test's measurements into one: the counts add
// up, the first-token figure is the mean, and the rate is every token the
// batch produced over the batch's wall time — what the machine delivered,
// which is the figure the concurrency setting is sized from.
func aggregate(name string, ms []measurement, wall time.Duration) Test {
	t := Test{Name: name, Parallel: len(ms), TotalMs: wall.Milliseconds()}
	var first time.Duration
	for _, m := range ms {
		t.PromptTokens += m.PromptTokens
		t.CompletionTokens += m.CompletionTokens
		first += m.FirstToken
		if t.Counted == "" || m.Counted == CountedChunks {
			t.Counted = m.Counted
		}
	}
	t.FirstTokenMs = (first / time.Duration(len(ms))).Milliseconds()
	if wall > 0 {
		t.TokensPerSec = round1(float64(t.CompletionTokens) / wall.Seconds())
	}
	return t
}

func round1(f float64) float64 { return float64(int64(f*10+0.5)) / 10 }

// measure sends one streaming request and times it.
func (r *Runner) measure(ctx context.Context, up Upstream, spec testSpec) (measurement, error) {
	ctx, cancel := context.WithTimeout(ctx, r.opts.RequestTimeout)
	defer cancel()

	body, err := json.Marshal(map[string]any{
		"model":       up.ModelArg,
		"messages":    []map[string]string{{"role": "user", "content": spec.Prompt}},
		"max_tokens":  spec.MaxTokens,
		"temperature": 0,
		"stream":      true,
		// The key is always written: a stream_options object without it
		// raises inside mlx-lm (internal/mlxtest).
		"stream_options": map[string]bool{"include_usage": true},
	})
	if err != nil {
		return measurement{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, up.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return measurement{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	started := time.Now()
	resp, err := r.opts.Client.Do(req)
	if err != nil {
		return measurement{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// The status is the whole of what is kept: the body is the model
		// server's and may say anything.
		return measurement{}, fmt.Errorf("model server answered %d", resp.StatusCode)
	}
	m, err := readStream(resp.Body, started)
	if err != nil {
		return measurement{}, err
	}
	return m, nil
}

// streamEvent is the part of a streamed chunk the self-test reads: whether it
// carries a piece of the answer, and the counts. What the piece says is never
// kept.
type streamEvent struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// readStream reads an SSE body to its end, timing the first chunk that
// carries text and counting the chunks, and takes the counts from the usage
// event when the server sends one.
func readStream(body io.Reader, started time.Time) (measurement, error) {
	var (
		m      measurement
		chunks int
		done   bool
	)
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(line[len("data:"):])
		if bytes.Equal(data, []byte("[DONE]")) {
			done = true
			break
		}
		var ev streamEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			continue
		}
		if len(ev.Choices) > 0 && ev.Choices[0].Delta.Content != "" {
			chunks++
			if m.FirstToken == 0 {
				m.FirstToken = time.Since(started)
			}
		}
		if ev.Usage != nil {
			m.PromptTokens, m.CompletionTokens = ev.Usage.PromptTokens, ev.Usage.CompletionTokens
			m.Counted = CountedUsage
		}
	}
	if err := sc.Err(); err != nil {
		return measurement{}, err
	}
	if !done {
		return measurement{}, errors.New("the stream ended without [DONE]")
	}
	m.Total = time.Since(started)
	if m.Counted == "" {
		m.CompletionTokens, m.Counted = chunks, CountedChunks
	}
	return m, nil
}

// Package toolprobe asks a served model one fixed question with one small
// tool declared, and records whether it answered with a tool call
// (itd-2609201445423499). The verdict lives beside the model's context
// measurement in the registry and is published on the models list as a
// plain field, so every client learns it once from the server rather than
// each finding it out for itself.
//
// It is not a client of the gateway. What it measures is a property of the
// model's chat template and weights on this runtime, which the gateway does
// not touch, and the run must appear in no request statistic and no request
// log line — so it takes the self-test's route: one request to the model
// server's own loopback address with the exact model argument that server
// expects (runtime.Upstream). Nothing counts it because nothing on that path
// counts anything.
//
// It runs once per model per runtime version, after the request that loaded
// the model has been served, one model at a time on a goroutine of its own.
// It never loads a model and never evicts one: it takes the same soft
// acquisition the self-test takes, so a client that needs the memory takes
// the model rather than being refused, and a model that has gone by the time
// the queue reaches it is dropped, to be queued again the next time it is
// served.
//
// This file is on internal/archtest's allow-list for both sides of the
// conversation boundary (prompt_content_test.go): it builds its one-line
// conversation from the constant below and reads nothing from a client, and
// it reads whether the choice carries a tool call and keeps nothing of what
// the answer says.
package toolprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
)

// Name is the probe's name, as the log carries it.
const Name = "tool-probe"

// Prompt is the one user message the probe sends. It asks for something the
// tool can answer and the model cannot, so a model that makes tool calls has
// every reason to make one. It carries no person's text and is generated
// from nothing.
const Prompt = "What is the time in Paris right now?"

// ToolName is the one function the probe declares.
const ToolName = "get_time"

// MaxTokens bounds the answer. A tool call is a few dozen tokens; the
// allowance leaves a model that reasons before it answers room to reach one,
// and stops a model that only talks from talking for long.
const MaxTokens = 256

// The defaults for the two durations in Options.
const (
	// DefaultRequestTimeout bounds the one request. The model is loaded and
	// the prompt is a sentence, so an answer that takes longer than this is
	// the server's, not the model's.
	DefaultRequestTimeout = 2 * time.Minute
	// DefaultPoll is how often a queued probe asks whether the request that
	// loaded the model has been served.
	DefaultPoll = 250 * time.Millisecond
)

// Upstream is a ready model server: where it answers, and the exact string
// its "model" field must carry (runtime.Upstream.ModelArg).
type Upstream struct {
	BaseURL  string
	ModelArg string
}

// Sources is what the probe needs from the app.
type Sources interface {
	// Resident reports whether the model's server is loaded and answering,
	// and how many requests are in flight on it. A model the pool is not
	// holding is (false, 0).
	Resident(repoID string) (loaded bool, inFlight int)
	// Acquire holds the model until the release function is called, the way
	// the self-test's own acquisition does: a soft hold a client's load may
	// take back. The probe asks Resident first and never acquires a model
	// that is not loaded, since the pool's acquisition would load it.
	Acquire(ctx context.Context, repoID string) (Upstream, func(), error)
	// Runtime is the runtime version in force, stamped on the verdict.
	Runtime() string
	// Save records the verdict on the model.
	Save(repoID string, tc *registry.ToolCalling) error
}

// Options configures a Probe. Only Sources is required.
type Options struct {
	Sources Sources
	Log     *slog.Logger
	// Client sends the request. Nil means a plain client that follows no
	// redirect: the one place a request may go is the model server the pool
	// named.
	Client *http.Client
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// RequestTimeout bounds the one request; zero means the default.
	RequestTimeout time.Duration
	// Poll is how often a queued probe looks again; zero means the default.
	Poll time.Duration
}

// ErrGone is returned by Run for a model that is not loaded: the probe
// never loads one.
var ErrGone = errors.New("the model is not loaded")

// Probe is the job and its queue.
type Probe struct {
	opts Options

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu sync.Mutex
	// queue holds the models waiting for a probe, the one in progress first.
	queue []string
	// running says the drain goroutine is up.
	running bool
}

// New builds a Probe. It starts nothing until the first Enqueue.
func New(opts Options) *Probe {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Client == nil {
		opts.Client = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = DefaultRequestTimeout
	}
	if opts.Poll <= 0 {
		opts.Poll = DefaultPoll
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Probe{opts: opts, ctx: ctx, cancel: cancel}
}

// Enqueue queues one probe of the model and reports whether it was added: a
// model already queued, or in progress, is not queued again. The queue is
// drained one model at a time on the probe's own goroutine.
func (p *Probe) Enqueue(repoID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctx.Err() != nil {
		return false
	}
	key := config.FoldRepoID(repoID)
	for _, q := range p.queue {
		if config.FoldRepoID(q) == key {
			return false
		}
	}
	p.queue = append(p.queue, repoID)
	if !p.running {
		p.running = true
		p.wg.Add(1)
		go p.drain()
	}
	return true
}

// Queued lists the models waiting for a probe, the one in progress first.
func (p *Probe) Queued() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.queue...)
}

// Close stops the queue where it stands and waits for the goroutine. A probe
// in flight is cancelled and records nothing.
func (p *Probe) Close() {
	p.cancel()
	p.wg.Wait()
}

// drain serves the queue head by head until it is empty or the probe closes.
func (p *Probe) drain() {
	defer p.wg.Done()
	for {
		p.mu.Lock()
		if len(p.queue) == 0 || p.ctx.Err() != nil {
			p.running = false
			p.mu.Unlock()
			return
		}
		model := p.queue[0]
		p.mu.Unlock()
		p.serve(model)
		p.dequeue(model)
	}
}

// serve waits for the request that loaded the model to be served, then runs
// one probe. A model that has gone meanwhile is dropped: it is queued again
// the next time it is served.
func (p *Probe) serve(model string) {
	for {
		loaded, inFlight := p.opts.Sources.Resident(model)
		if !loaded {
			p.opts.Log.Debug("tool-call probe dropped: the model is no longer loaded; it is queued again at its next serve", "model", model)
			return
		}
		if inFlight == 0 {
			break
		}
		select {
		case <-time.After(p.opts.Poll):
		case <-p.ctx.Done():
			return
		}
	}
	if _, err := p.Run(p.ctx, model); err != nil && p.ctx.Err() == nil {
		p.opts.Log.Info("tool-call probe recorded nothing: the model server did not answer", "model", model)
		p.opts.Log.Debug("tool-call probe failed", "model", model, "err", err)
	}
}

func (p *Probe) dequeue(model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := config.FoldRepoID(model)
	for i, q := range p.queue {
		if config.FoldRepoID(q) == key {
			p.queue = append(p.queue[:i], p.queue[i+1:]...)
			return
		}
	}
}

// Run probes one model now: one acquisition, one request, one verdict
// recorded under the runtime in force. It returns the verdict it recorded,
// or an error and no verdict when the server did not answer — a transport
// error, a non-200 or a timeout is no evidence about the model, and the model
// stays unprobed. A model that is not loaded is ErrGone.
func (p *Probe) Run(ctx context.Context, model string) (*registry.ToolCalling, error) {
	if loaded, _ := p.opts.Sources.Resident(model); !loaded {
		return nil, ErrGone
	}
	up, release, err := p.opts.Sources.Acquire(ctx, model)
	if err != nil {
		return nil, err
	}
	defer release()
	can, err := p.request(ctx, up)
	if err != nil {
		return nil, err
	}
	tc := &registry.ToolCalling{Can: can, At: p.opts.Now().Unix(), Runtime: p.opts.Sources.Runtime()}
	if err := p.opts.Sources.Save(model, tc); err != nil {
		p.opts.Log.Warn("tool-call probe: could not record the verdict", "model", model, "err", err)
		return nil, err
	}
	p.opts.Log.Info("tool-call probe recorded", "model", model, "can", can)
	return tc, nil
}

// request sends the fixed request to the model server's own address and
// reads one thing of the answer: whether the choice carries a tool call.
func (p *Probe) request(ctx context.Context, up Upstream) (bool, error) {
	body, err := json.Marshal(map[string]any{
		"model":       up.ModelArg,
		"messages":    []map[string]string{{"role": "user", "content": Prompt}},
		"max_tokens":  MaxTokens,
		"temperature": 0,
		"stream":      false,
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name":        ToolName,
				"description": "The current time in a city.",
				"parameters": map[string]any{
					"type":       "object",
					"properties": map[string]any{"city": map[string]any{"type": "string"}},
					"required":   []string{"city"},
				},
			},
		}},
	})
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(ctx, p.opts.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, up.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.opts.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// The status is the whole of what is kept: the body is the model
		// server's and may say anything.
		return false, fmt.Errorf("model server answered %d", resp.StatusCode)
	}
	// Only the shape of the choice is decoded: whether it carries a tool
	// call and how it finished. The message's text is not read.
	var out struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				ToolCalls []json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return false, err
	}
	if len(out.Choices) == 0 {
		// No choice at all: text, an empty message or nothing each record
		// cannot, and a choice-less answer is the last of those.
		return false, nil
	}
	c := out.Choices[0]
	return len(c.Message.ToolCalls) > 0 || c.FinishReason == "tool_calls", nil
}

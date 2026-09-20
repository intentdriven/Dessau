package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/stats"
)

// Ask serves one completion to a caller inside this process, without a network
// hop.
//
// It exists for a bridge (adr-2609181004167097): a conversation that arrives
// over an outbound connection the Mac itself opened has no socket to be
// admitted on, no Host and no Origin, and posting it back to this server's own
// LAN address would need the API key the bridge is not holding. What it must
// NOT be is a second request path. Everything a request is subjected to on the
// way to a model server is subjected to it here, in the same order and by the
// same code: the model is resolved through resolveModel, the size is judged
// through judgeServedContext, the pool is acquired the same way, the load-
// bearing "model" rewrite happens (mlx-lm reads that field as an instruction
// to LOAD, which is why it is rewritten and not relayed — see DECISIONS.md),
// system messages are merged for a model the operator switched merging on for,
// and the whole thing is observed and recorded exactly as an HTTP request is.
// The one difference is the record's source, which says a bridge brought it.
//
// It carries none of the HTTP admission, and that is deliberate rather than an
// omission: withAuth decides who may reach this server over the network, and
// the caller here is this process. What decides who may talk to a bridge is
// the bridge's own reach — where the operator invited the bot — which is the
// maintainer's decision recorded in itd-2609180959397172.
//
// Nothing of the body is read here. Ask is handed bytes the caller encoded and
// hands back the bytes the model server produced; the one field it looks at is
// "model", and the merge is the grant adr-2609061610102325 already made.
func (g *Gateway) Ask(ctx context.Context, req AskRequest) error {
	started := time.Now()
	cfg := g.cfg()
	obs := g.observe(cfg.Statistics, started)
	obs.sourced(req.Source)
	defer obs.finish(ctx)

	if len(req.Body) > maxRequestBody {
		obs.failed(stats.ClassClientError)
		return &AskError{detail: "the request is larger than this server will read", public: genericRefusal}
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(req.Body, &payload); err != nil {
		obs.failed(stats.ClassGatewayError)
		return &AskError{detail: "the request is not valid JSON", public: genericRefusal}
	}

	model, err := g.resolveModel(req.Model)
	if err != nil {
		obs.failed(stats.ClassClientError)
		// The detailed text describes this Mac — which models are here and
		// which are still downloading — and the caller is about to post its
		// answer onto a platform. It goes to the operator's log and the
		// generic one travels.
		return &AskError{detail: err.Error(), public: genericRefusal}
	}
	obs.resolved(model)
	obs.streaming(true)

	msg, verdict := g.judgeServedContext(cfg, model, len(req.Body), payload)
	if obs.recording() {
		obs.judged(verdict.declared, verdict.served, verdict.estimate, verdict.judged)
		obs.overrides(overriddenSampling(payload))
	}
	if msg != "" {
		obs.failed(stats.ClassClientError)
		// The one refusal whoever asked can act on, so it travels as a
		// sentence of this package's own rather than as the generic text
		// (iss-2609190106563320). Not the operator's text, which says to
		// raise this model's served context in Settings — that is an
		// instruction to the operator, and the person on the other end is a
		// stranger who cannot follow it and should not be told to.
		return &AskError{detail: msg, class: "too_large", public: tooLongRefusal}
	}

	up, release, err := g.pool.Acquire(ctx, model)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			obs.failed(stats.ClassCancelled)
			return &AskError{detail: "the request was cancelled", public: genericRefusal}
		}
		obs.failed(classifyAcquireError(err))
		var noRoom *runtime.NoRoomError
		if errors.As(err, &noRoom) {
			obs.waited(runtime.AcquireStats{QueueWait: noRoom.Waited})
		}
		// The pool's own text is not handed back, and that is the same rule
		// handleCompletions follows for an unentitled client
		// (iss-2609190106273104). A LaunchError's text is the child process's
		// verbatim and carries absolute local paths under the serving
		// account's home directory; the other refusals name the resident
		// memory budget in bytes, which says roughly how much memory this Mac
		// has. The caller of Ask is relaying to somebody else's platform, so
		// what it gets is the CLASS — a fixed word — and the operator keeps
		// the text, here, at the level they asked for it at and no more often
		// than they would have got it over HTTP.
		var launchErr *runtime.LaunchError
		if errors.As(err, &launchErr) {
			g.log.Error("model launch failed", "model", model)
			g.log.Debug("model launch failed", "model", model, "err", err)
			return &AskError{detail: "the model could not be started", class: refusalClass(err), public: genericRefusal}
		}
		if g.refusalLog.allow(model) {
			class := refusalClass(err)
			g.log.Info("refused an in-process request, and told the caller only that it could not be served",
				"model", model, "class", class)
			g.log.Debug("refused an in-process request, and told the caller only that it could not be served",
				"model", model, "class", class, "err", err)
		}
		return &AskError{detail: "this model cannot be served right now", class: refusalClass(err), public: genericRefusal}
	}
	defer release()
	obs.waited(up.Waits)

	rewritten, err := json.Marshal(up.ModelArg)
	if err != nil {
		obs.failed(stats.ClassGatewayError)
		return &AskError{detail: "could not re-encode the request", public: genericRefusal}
	}
	payload["model"] = rewritten

	if cfg.Models[model].MergeSystemMessages {
		if mergeSystemMessagesInto(payload) == mergeRefused {
			g.log.Debug("relayed a request unmerged: its messages carry something merging cannot rebuild faithfully", "model", model)
		}
	}
	// Always streamed and always counted. The caller is editing a message as
	// the answer is written, so there is no non-streamed shape to serve; and
	// the counts are what the record is for, so they are asked for here the
	// way handleCompletions asks for them on a client's behalf.
	payload["stream"] = json.RawMessage("true")
	mergeIncludeUsage(payload)

	body, err := json.Marshal(payload)
	if err != nil {
		obs.failed(stats.ClassGatewayError)
		return &AskError{detail: "could not re-encode the request", public: genericRefusal}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, up.BaseURL+chatCompletionsPath, bytes.NewReader(body))
	if err != nil {
		obs.failed(stats.ClassGatewayError)
		return &AskError{detail: err.Error(), public: genericRefusal}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.ContentLength = int64(len(body))

	// The same bound handleCompletions applies, for the same reason: prefill
	// scales with the prompt and generation legitimately runs for minutes
	// after the first token, so the wait for HEADERS is what is bounded.
	budget := prefillBudget(len(body), cfg.UpstreamHeaderTimeoutSec)
	hdrCtx, cancelHdr := context.WithCancel(ctx)
	defer cancelHdr()
	timer := time.AfterFunc(budget, cancelHdr)
	resp, err := g.tr.RoundTrip(httpReq.WithContext(hdrCtx))
	timer.Stop()
	if err != nil {
		if ctx.Err() != nil {
			obs.failed(stats.ClassCancelled)
			return &AskError{detail: "the request was cancelled", public: genericRefusal}
		}
		obs.failed(stats.ClassUnreachable)
		g.log.Error("upstream request failed", "model", model, "err", err)
		return &AskError{detail: "the model server did not respond", public: genericRefusal}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		obs.failed(stats.ClassUpstreamStatus)
		// The model server's own body is not read: it is the model server's
		// text about this Mac, and nothing here needs it.
		return &AskError{
			detail: fmt.Sprintf("the model server answered %d", resp.StatusCode),
			public: genericRefusal,
		}
	}

	out := g.streamToCallback(resp.Body, up.ModelArg, req.Model, req.OnEvent)
	if out.oversizeLine {
		g.log.Error("ended a streamed answer: the model server sent a line beyond the relay's limit",
			"model", model, "limit", maxStreamLine)
	}
	obs.relayed(out)
	if obs.recording() {
		obs.footprint(g.pool.Footprint(model))
	}
	if out.upstreamCut {
		return &AskError{detail: "the answer stopped part-way", public: genericRefusal}
	}
	return nil
}

// tooLongRefusal is what whoever asked is told when the conversation is
// larger than the window this model is served at. It names no figure: the
// window is a setting on this Mac.
const tooLongRefusal = "that conversation is too long for this model"

// AskRequest is one completion asked for inside this process.
type AskRequest struct {
	// Model is the name to serve, resolved exactly as a network client's is:
	// the full repo id or the short name.
	Model string
	// Body is the encoded chat-completions request. It is the caller's to
	// build, so a caller that is allowed to read a conversation builds it and
	// this package does not have to.
	Body []byte
	// OnEvent receives each streamed event's payload, already rewritten so
	// that the model server's own absolute path is nowhere in it. It is called
	// on the calling goroutine, in order, and never after Ask returns. A
	// caller that reads the answer out of it is a reader of generated content
	// and answers to adr-2609061610102325 for that.
	OnEvent func(payload []byte)
	// Source is how this request reached the Mac, for the record.
	Source stats.Source
}

// AskError is a refusal of an in-process request, carrying the two texts a
// refusal has: what the operator's log is told, and what may be repeated to
// whoever asked.
//
// The split is the whole reason this type exists. A bridge posts its answer
// onto somebody else's platform, so the informative refusals — which model is
// loading, how much memory the budget is, what to raise in Settings — must not
// travel: they describe this Mac to a stranger, which is the same rule
// genericRefusal already applies to an unentitled network client.
type AskError struct {
	detail string
	public string
	class  string
}

func (e *AskError) Error() string { return e.detail }

// Class is a fixed word naming what kind of refusal this was, for a caller
// that wants to record one without recording the text.
//
// It exists because the text of a refusal is not a caller's to repeat — not to
// a platform, and not into a log line a stranger can cause at the rate they can
// send messages (iss-2609190106273104). A class is a closed set of this
// package's own words, so it is bounded and it says nothing about this Mac.
func (e *AskError) Class() string {
	if e.class == "" {
		return "refused"
	}
	return e.class
}

// Public is what may be relayed off this Mac.
func (e *AskError) Public() string {
	if e.public == "" {
		return genericRefusal
	}
	return e.public
}

// streamToCallback reads the model server's SSE body and hands each event's
// payload to the caller, rewriting the backend's "model" value out of it on
// the way, and reporting what the relay saw exactly as the HTTP relay does.
//
// It is a second loop rather than a reuse of streamRewriteSSE because that one
// writes to an http.ResponseWriter and flushes it; everything below it —
// readBoundedLine, cutDataPrefix, decodeEvent, renderEvent, readUsage — is the
// same code, so the two agree about what an event is, what a line's limit is,
// and when the first token happened.
func (g *Gateway) streamToCallback(src io.Reader, modelArg, requested string, onEvent func([]byte)) relayOutcome {
	var out relayOutcome
	br := bufio.NewReader(src)
	for {
		line, err := readBoundedLine(br, maxStreamLine)
		if errors.Is(err, errLineTooLong) {
			out.upstreamCut = true
			out.oversizeLine = true
			return out
		}
		if len(line) > 0 {
			if _, payload, ok := cutDataPrefix(line); ok {
				ev, parsed := decodeEvent(payload)
				if parsed {
					if u := readUsage(ev); u != nil {
						out.usage = u
					}
					if out.firstToken.IsZero() && carriesGeneration(ev) {
						out.firstToken = time.Now()
					}
				}
				if isTerminalEvent(payload) {
					out.complete = true
				} else if onEvent != nil {
					onEvent(renderEvent(payload, ev, parsed, modelArg, requested))
				}
			}
		}
		if err != nil {
			out.upstreamCut = !errors.Is(err, io.EOF) && !out.complete
			return out
		}
	}
}

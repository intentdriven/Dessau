package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultAPIBase is Discord's REST root at API version 10.
const defaultAPIBase = "https://discord.com/api/v10"

// restTimeout bounds one REST call. An edit that hangs would otherwise stall
// the answer it belongs to for as long as the other end cared to hold it.
const restTimeout = 30 * time.Second

// maxRESTBody bounds what is read back from a REST call. The bodies this
// bridge reads are a message object and an error object, both small; the cap
// is what keeps a hostile or broken peer from growing this process by
// answering with a stream.
const maxRESTBody = 1 << 20

// rest is Discord's HTTP API, as much of it as this bridge uses.
//
// The token goes on every request as the Authorization header and NOWHERE
// else: it is never put in a URL, never logged, and never returned in an
// error — every error below is built from the status and the path, so a
// failure that reaches the operator's log or the panel cannot carry the
// credential (adr-2609181004167097 condition 3).
type rest struct {
	base   string
	token  string
	client *http.Client
	log    *slog.Logger
}

func newREST(base, token string, client *http.Client, log *slog.Logger) *rest {
	return &rest{base: strings.TrimSuffix(base, "/"), token: token, client: client, log: log}
}

// rateLimit is what a response said about how soon the next call may go.
type rateLimit struct {
	// resetAfter is how long until the bucket refills, set only when the
	// response said this call used the last of it.
	resetAfter time.Duration
	// exhausted says the remaining allowance is zero.
	exhausted bool
}

// do makes one call and returns the decoded body and what the rate-limit
// headers said.
//
// A 429 is honoured once, by waiting what Discord asked for and trying again.
// Once, not in a loop: the caller's throttle is what keeps this from happening
// in the first place, and a retry loop on a shared bucket is how a client gets
// itself banned.
func (r *rest) do(ctx context.Context, method, path string, body any, out any) (rateLimit, error) {
	limit, status, raw, err := r.call(ctx, method, path, body)
	if err != nil {
		return limit, err
	}
	if status == http.StatusTooManyRequests {
		wait := retryAfter(raw)
		if wait <= 0 || wait > maxRetryAfter {
			return limit, fmt.Errorf("discord rate-limited %s %s", method, path)
		}
		select {
		case <-ctx.Done():
			return limit, ctx.Err()
		case <-time.After(wait):
		}
		limit, status, raw, err = r.call(ctx, method, path, body)
		if err != nil {
			return limit, err
		}
	}
	if status >= 300 {
		return limit, fmt.Errorf("discord answered %d to %s %s", status, method, path)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return limit, fmt.Errorf("discord's answer to %s %s could not be read: %w", method, path, err)
		}
	}
	return limit, nil
}

// maxRetryAfter bounds how long a 429 may park a call. Beyond it the call
// fails and the answer carries on without that edit, which is better than
// holding a worker for as long as the other end names.
const maxRetryAfter = 30 * time.Second

func (r *rest) call(ctx context.Context, method, path string, body any) (rateLimit, int, []byte, error) {
	var buf io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return rateLimit{}, 0, nil, err
		}
		buf = bytes.NewReader(encoded)
	}
	ctx, cancel := context.WithTimeout(ctx, restTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, r.base+path, buf)
	if err != nil {
		return rateLimit{}, 0, nil, err
	}
	// "Bot " is Discord's own scheme for a bot token and is not a bearer
	// token; sending it as one is refused.
	req.Header.Set("Authorization", "Bot "+r.token)
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		// Wrapped without the URL's query, and the URL never carries the
		// token, so nothing here can leak it.
		return rateLimit{}, 0, nil, fmt.Errorf("calling discord %s %s: %w", method, path, redactURLError(err))
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxRESTBody))
	if err != nil {
		return readRateLimit(resp.Header), resp.StatusCode, nil, err
	}
	return readRateLimit(resp.Header), resp.StatusCode, raw, nil
}

// userAgent is what Discord asks a library-less client to identify itself as.
const userAgent = "DiscordBot (https://github.com/intentdriven/Gropius, 1.0)"

// readRateLimit reads the two headers that decide how soon the next call may
// go. A header that is absent or unreadable means no constraint, which is the
// same answer as a full bucket: the editor's own two-second floor is what
// keeps that from being a licence to spin.
func readRateLimit(h http.Header) rateLimit {
	var out rateLimit
	remaining, err := strconv.Atoi(strings.TrimSpace(h.Get("X-RateLimit-Remaining")))
	if err != nil || remaining > 0 {
		return out
	}
	out.exhausted = true
	secs, err := strconv.ParseFloat(strings.TrimSpace(h.Get("X-RateLimit-Reset-After")), 64)
	if err != nil || secs <= 0 {
		return out
	}
	if secs > maxRetryAfter.Seconds() {
		secs = maxRetryAfter.Seconds()
	}
	out.resetAfter = time.Duration(secs * float64(time.Second))
	return out
}

// retryAfter reads how long a 429 body asked us to wait.
func retryAfter(raw []byte) time.Duration {
	var payload struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if json.Unmarshal(raw, &payload) != nil || payload.RetryAfter <= 0 {
		return 0
	}
	return time.Duration(payload.RetryAfter * float64(time.Second))
}

// redactURLError strips the URL out of a transport error.
//
// The URL this bridge builds never carries the token — it goes in a header —
// but an *url.Error quotes the whole request line, and this is the file where
// a URL would one day come to carry something it should not. Unwrapping it to
// the transport's own error is one line and removes the question.
func redactURLError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err
	}
	return err
}

// message is the only part of a Discord message object this bridge reads back:
// the id, so a placeholder can be edited.
type message struct {
	ID string `json:"id"`
}

// createMessage posts a new message in a channel, optionally as a reply.
func (r *rest) createMessage(ctx context.Context, channelID, text, replyTo string) (message, rateLimit, error) {
	body := map[string]any{
		"content": text,
		// Nothing this bot writes may ping anybody. The answer is a model's
		// text and may contain anything at all, so the one defence that
		// actually holds is telling Discord to resolve no mentions in it:
		// without this a model could be talked into writing @everyone.
		"allowed_mentions": map[string]any{"parse": []string{}},
	}
	if replyTo != "" {
		body["message_reference"] = map[string]any{"message_id": replyTo, "fail_if_not_exists": false}
	}
	var out message
	limit, err := r.do(ctx, http.MethodPost, "/channels/"+channelID+"/messages", body, &out)
	return out, limit, err
}

// editMessage replaces a message's text.
func (r *rest) editMessage(ctx context.Context, channelID, messageID, text string) (rateLimit, error) {
	body := map[string]any{
		"content":          text,
		"allowed_mentions": map[string]any{"parse": []string{}},
	}
	return r.do(ctx, http.MethodPatch, "/channels/"+channelID+"/messages/"+messageID, body, nil)
}

// typing shows the bot as typing in a channel for about ten seconds.
func (r *rest) typing(ctx context.Context, channelID string) error {
	_, err := r.do(ctx, http.MethodPost, "/channels/"+channelID+"/typing", struct{}{}, nil)
	return err
}

// respondToInteraction answers a slash command with a message only the person
// who ran it sees.
//
// Type 4 is "reply with a message now"; flag 64 is ephemeral. Both `/model`
// and `/reset` answer this way: what model a channel is on, and that its
// history was cleared, are answers to the person who asked and not
// announcements to everyone in the channel.
func (r *rest) respondToInteraction(ctx context.Context, id, token, text string) error {
	body := map[string]any{
		"type": 4,
		"data": map[string]any{
			"content":          text,
			"flags":            1 << 6,
			"allowed_mentions": map[string]any{"parse": []string{}},
		},
	}
	_, err := r.do(ctx, http.MethodPost, "/interactions/"+id+"/"+token+"/callback", body, nil)
	return err
}

// overwriteCommands registers this bridge's application commands, replacing
// whatever was registered before.
//
// A bulk overwrite rather than a create per command: it is idempotent, so
// running it once per start cannot accumulate duplicates, and a command
// removed from this list is removed from Discord by the next start.
func (r *rest) overwriteCommands(ctx context.Context, appID string, commands []any) error {
	_, err := r.do(ctx, http.MethodPut, "/applications/"+appID+"/commands", commands, nil)
	return err
}

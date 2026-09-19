package discord

import (
	"context"
	"errors"
	"strings"
	"time"
)

// messageLimit is Discord's ceiling on one message, in characters. An answer
// is cut before it and continued in a new message.
const messageLimit = 2000

// cutBefore is where the editor starts looking for a paragraph break. The
// margin under the ceiling is what lets the cut land on a paragraph rather
// than on the last character that fits: without it the only cut available at
// 2,000 is a hard one mid-word.
const cutBefore = 1900

// minEditInterval is the floor on how often a placeholder is edited.
//
// "At least every two seconds" is the promise in itd-2609180959397172, and it
// is a floor rather than a cadence: the rate-limit headers may ask for longer
// and are obeyed, but nothing makes an edit go out sooner. A token-by-token
// edit would be a request per token to somebody else's API and would have the
// bot rate-limited within a sentence.
const minEditInterval = 2 * time.Second

// editor writes a streamed answer into Discord, as a placeholder message that
// fills in.
//
// It is not safe for concurrent use and is not used concurrently: one editor
// belongs to one answer, and answers in a channel are serialised by the
// conversation's mutex.
type editor struct {
	rest      *rest
	channelID string
	// replyTo is the message being answered, so the first message of the
	// answer is a reply to it and the rest continue from there.
	replyTo string
	now     func() time.Time

	// messageID is the message currently being filled in, empty before the
	// first token has produced one.
	messageID string
	// pending is what the current message should say, and sent is what it
	// says. They differ between the moment a token arrives and the moment the
	// throttle lets the edit go.
	pending string
	sent    string
	// nextEdit is the earliest the next edit may go out.
	nextEdit time.Time
	// err is the first failure. The answer carries on being generated — the
	// model server is already working and the record is already being kept —
	// but nothing more is written to Discord.
	err error
}

func newEditor(r *rest, channelID, replyTo string, now func() time.Time) *editor {
	return &editor{rest: r, channelID: channelID, replyTo: replyTo, now: now}
}

// add takes the next piece of the answer and writes it out if the throttle
// allows, cutting and continuing in a new message when the current one is
// full.
func (e *editor) add(ctx context.Context, delta string) {
	if e.err != nil || delta == "" {
		return
	}
	e.pending += delta
	for len([]rune(e.pending)) > messageLimit {
		head, tail := cutAtParagraph(e.pending)
		e.pending = head
		// The finished message is written in full before the next one starts,
		// whatever the throttle says: it will never change again, and leaving
		// it short while a second message appeared below it would show the
		// answer out of order.
		if !e.write(ctx, true) {
			return
		}
		e.messageID, e.sent, e.pending = "", "", tail
		e.nextEdit = time.Time{}
	}
	e.write(ctx, false)
}

// finish writes whatever is left, ignoring the throttle: the answer is over
// and what Discord shows has to be what the model wrote.
func (e *editor) finish(ctx context.Context) error {
	if e.err == nil {
		e.write(ctx, true)
	}
	return e.err
}

// wrote reports whether anything has been posted, which is how the caller
// knows a refusal still needs a message of its own.
func (e *editor) wrote() bool { return e.messageID != "" }

// write pushes the pending text to Discord. force ignores the throttle.
func (e *editor) write(ctx context.Context, force bool) bool {
	if e.err != nil {
		return false
	}
	if e.pending == "" || e.pending == e.sent {
		return true
	}
	if !force && e.now().Before(e.nextEdit) {
		return true
	}
	var limit rateLimit
	var err error
	if e.messageID == "" {
		var msg message
		msg, limit, err = e.rest.createMessage(ctx, e.channelID, e.pending, e.replyTo)
		if err == nil && !validID(msg.ID) {
			// The id of the message to edit comes back over the network and
			// goes straight into the next call's path
			// (iss-2609190057562775). Discord always answers with one, so an
			// id that is not a snowflake means this is not Discord answering
			// — and carrying on would post the whole answer again, growing,
			// once per throttle interval. The answer stops here.
			e.err = errors.New("discord answered a created message with an id that is not a snowflake")
			return false
		}
		if err == nil {
			e.messageID = msg.ID
			// Only the first message of an answer is a reply; the rest
			// continue below it, which is how a long answer reads.
			e.replyTo = ""
		}
	} else {
		limit, err = e.rest.editMessage(ctx, e.channelID, e.messageID, e.pending)
	}
	if err != nil {
		e.err = err
		return false
	}
	e.sent = e.pending
	e.nextEdit = e.now().Add(nextInterval(limit))
	return true
}

// nextInterval is how long to wait before the next edit: the floor, or what
// the rate-limit headers asked for when that is longer.
func nextInterval(limit rateLimit) time.Duration {
	if limit.exhausted && limit.resetAfter > minEditInterval {
		return limit.resetAfter
	}
	return minEditInterval
}

// cutAtParagraph splits text so the first part is a whole message and the
// second continues it.
//
// A paragraph break is looked for first, then a line break, then a space —
// each within cutBefore, so the cut lands somewhere a reader would have
// paused. Text with none of the three (a single very long word, or a language
// that does not space its words) is cut at the ceiling, because a message that
// cannot be sent is worse than one that breaks mid-word.
//
// Everything is counted in runes, not bytes: Discord's limit is a count of
// characters, and a byte-wise cut would both mis-measure a message with any
// accented or non-Latin text in it and be able to split a character in half.
func cutAtParagraph(text string) (head, tail string) {
	runes := []rune(text)
	window := runes[:min(len(runes), cutBefore)]
	for _, sep := range [][]rune{[]rune("\n\n"), []rune("\n"), []rune(" ")} {
		if i := lastIndexRunes(window, sep); i > 0 {
			return string(runes[:i]), strings.TrimLeft(string(runes[i+len(sep):]), " \n")
		}
	}
	hard := min(len(runes), messageLimit)
	return string(runes[:hard]), string(runes[hard:])
}

// lastIndexRunes is strings.LastIndex over runes: the index of the last
// occurrence of needle in hay, or -1.
func lastIndexRunes(hay, needle []rune) int {
	for i := len(hay) - len(needle); i >= 0; i-- {
		if string(hay[i:i+len(needle)]) == string(needle) {
			return i
		}
	}
	return -1
}

// typingEvery is how often the bot is shown as typing while nothing has been
// written yet. Discord's indicator lasts about ten seconds, so eight keeps it
// unbroken while a cold model loads.
const typingEvery = 8 * time.Second

// showTyping keeps the bot shown as typing in a channel until ctx is done.
//
// It posts once immediately, so the indicator is up within the two seconds
// itd-2609180959397172 promises rather than eight seconds later, and then on
// the cadence. It is a best-effort courtesy: a failure is not logged and
// never affects the answer.
func (e *editor) showTyping(ctx context.Context, r *rest) {
	_ = r.typing(ctx, e.channelID)
	ticker := time.NewTicker(typingEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = r.typing(ctx, e.channelID)
		}
	}
}

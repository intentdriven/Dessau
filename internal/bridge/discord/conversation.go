package discord

import (
	"encoding/json"
	"sync"
)

// maxChannels is how many conversations one session holds. Anyone who can
// reach the bot may talk to it (itd-2609180959397172), so the number of
// channels is a stranger's to choose: without a bound this map is a way to
// grow the process one direct message at a time. The least recently used
// conversation goes when the bound is reached, which for a channel nobody is
// using means it starts afresh next time — exactly what `/reset` does.
const maxChannels = 256

// maxTurns is how many turns one conversation keeps, before the window is
// even considered. The window bound below is what actually decides what is
// sent; this is what stops the slice itself growing without limit on a model
// with a very large window.
const maxTurns = 64

// maxMessageRunes bounds what is taken from one incoming message. Discord's
// own ceiling is 2,000 characters for most accounts and 4,000 for some, and
// the request is built from this text — so it is bounded here rather than
// trusted to be whatever the platform currently allows.
const maxMessageRunes = 8000

// turn is one side of a conversation.
type turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// The two roles a turn can have.
const (
	roleUser      = "user"
	roleAssistant = "assistant"
)

// conversation is one channel's history and the model answering it.
//
// TWO LOCKS, AND THEY ARE NOT THE SAME ONE. `answering` says a channel has an
// answer in flight and is held for the whole of a generation, which is
// minutes; `mu` guards the turns and the model and is held for as long as it
// takes to copy a slice. A slash command takes the second and never the
// first, so `/model` and `/reset` are answered inside Discord's three seconds
// in a channel whose answer is still being written — and they do not occupy
// one of the two workers waiting to be (iss-2609190106414499).
type conversation struct {
	answering sync.Mutex

	mu    sync.Mutex
	model string
	turns []turn
}

// conversations is every channel the session has seen, bounded.
type conversations struct {
	mu    sync.Mutex
	byID  map[string]*conversation
	order []string
}

func newConversations() *conversations {
	return &conversations{byID: map[string]*conversation{}}
}

// get returns the conversation for a channel, creating it if this is the
// first message there and evicting the least recently used one if the bound
// has been reached.
func (c *conversations) get(channelID string) *conversation {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conv, ok := c.byID[channelID]; ok {
		c.touchLocked(channelID)
		return conv
	}
	if len(c.order) >= maxChannels {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.byID, oldest)
	}
	conv := &conversation{}
	c.byID[channelID] = conv
	c.order = append(c.order, channelID)
	return conv
}

func (c *conversations) touchLocked(channelID string) {
	for i, id := range c.order {
		if id == channelID {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, channelID)
}

// modelOf reports the model a channel is on, empty when it has not chosen one.
func (conv *conversation) modelOf() string {
	conv.mu.Lock()
	defer conv.mu.Unlock()
	return conv.model
}

// setModel puts a channel on a model and clears nothing: the history is the
// conversation, and changing which model continues it is not starting again.
func (conv *conversation) setModel(model string) {
	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.model = model
}

// reset clears a channel's history. The model it is on is kept: `/reset` is
// "start this conversation again", not "undo what I chose".
func (conv *conversation) reset() {
	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.turns = nil
}

// append adds a turn, holding the slice to maxTurns.
func (conv *conversation) append(t turn) {
	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.turns = append(conv.turns, t)
	if len(conv.turns) > maxTurns {
		conv.turns = append([]turn(nil), conv.turns[len(conv.turns)-maxTurns:]...)
	}
}

// history is a copy of the turns, safe to build a request from while another
// goroutine appends.
func (conv *conversation) history() []turn {
	conv.mu.Lock()
	defer conv.mu.Unlock()
	return append([]turn(nil), conv.turns...)
}

// request is the body sent to the gateway. It is the ordinary OpenAI shape
// and nothing more; every other field a client could send is one this bridge
// does not offer, so a person on Discord cannot set the sampling of a model
// on somebody else's Mac.
type request struct {
	Model     string `json:"model"`
	Messages  []turn `json:"messages"`
	MaxTokens int    `json:"max_tokens"`
	Stream    bool   `json:"stream"`
}

// answerTokens is how much of a window is left for the answer. It is also the
// max_tokens the request asks for, so the figure the gateway judges the
// request by is the figure this bridge reserved.
const answerTokens = 1024

// defaultWindow is the window assumed for a model this Mac cannot report one
// for. Deliberately small: the cost of assuming too little is a shorter
// history, and the cost of assuming too much is a refusal the person on
// Discord can do nothing about.
const defaultWindow = 8192

// bytesPerToken is the gateway's own estimator, restated here because the
// bridge has to reach the same verdict the gateway will: the gateway
// estimates a request's size from its encoded bytes at four per token, so a
// history trimmed by any other rule would be trimmed to the wrong size and
// the gateway would refuse it. internal/archtest holds the two equal.
const bytesPerToken = 4

// buildRequest encodes the newest turns that fit the model's served window.
//
// It is built and measured rather than counted, because what the gateway
// judges is the encoded body: turns are dropped from the front until the body
// this bridge is about to send is one the gateway will admit, so a long
// conversation is shortened here and never refused there. A single turn too
// large for the window on its own is truncated, because refusing it would
// leave the person unable to say anything at all in that channel.
func buildRequest(model string, turns []turn, served int64) ([]byte, error) {
	window := served
	if window <= 0 {
		window = defaultWindow
	}
	answer := answerTokens
	if int64(answer) > window/2 {
		answer = int(window / 2)
	}
	budget := (window - int64(answer)) * bytesPerToken

	for start := 0; start < len(turns); start++ {
		body, err := json.Marshal(request{
			Model: model, Messages: turns[start:], MaxTokens: answer, Stream: true,
		})
		if err != nil {
			return nil, err
		}
		if int64(len(body)) <= budget {
			return body, nil
		}
	}
	// Even the newest turn alone is over the window. Keep its tail — the end
	// of what somebody typed is the part they are asking about.
	last := turn{Role: roleUser}
	if len(turns) > 0 {
		last = turns[len(turns)-1]
	}
	last.Content = tailRunes(last.Content, int(budget/bytesPerToken))
	return json.Marshal(request{
		Model: model, Messages: []turn{last}, MaxTokens: answer, Stream: true,
	})
}

// tailRunes keeps the last n runes of s, cutting on a rune boundary so the
// result is still text.
func tailRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

// headRunes keeps the first n runes of s.
func headRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

package discord

import (
	"encoding/json"
	"sync"
	"unicode/utf8"
)

// maxChannels is how many conversations one bridge holds. Anyone who can
// reach the bot may talk to it (itd-2609180959397172), so the number of
// channels is a stranger's to choose: without a bound this map is a way to
// grow the process one direct message at a time. The least recently used
// conversation goes when the bound is reached, which for a channel nobody is
// using means it starts afresh next time — exactly what `/reset` does.
const maxChannels = 64

// maxTurns is how many turns one conversation keeps, before the window is
// even considered. The window bound below is what actually decides what is
// sent; this is what stops the slice itself growing without limit on a model
// with a very large window.
const maxTurns = 32

// maxTurnBytes bounds one turn, in bytes: what is taken from an incoming
// message and what is kept of an answer. Discord's own ceiling is 4,000
// characters for the accounts allowed the most, which is 16,000 bytes at four
// a rune, so no message Discord delivers is cut; a bound in runes would have
// been four times this in the scripts where a rune is three or four bytes.
const maxTurnBytes = 16 << 10

// maxStoreBytes is the one budget every channel's turns share. When a turn
// would take the store past it, the oldest turns go first, whichever channel
// holds them; a channel that loses every turn keeps its model.
const maxStoreBytes = 8 << 20

// WHAT THE BOUNDS ABOVE COST, SAID OUT LOUD (iss-2609190312188937, decided
// 2026-09-20). Without the budget, 64 channels × 32 turns × 16 KiB would be
// 32 MiB; the budget holds the turns' text to 8 MiB whatever a stranger
// sends, a turn stored before its completion is asked for included. Each
// conversation, map entry and turn header adds well under a kilobyte on top.
// An ordinary channel — 32 turns of a few kilobytes — is around 100 KiB, so
// the budget bites only when dozens of channels are busy at once, and then on
// the oldest of what they said. The figure was 512 MiB before; the decision
// took the tightest of the four shapes put to the maintainer.

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

	// store is the set this conversation's bytes are counted in; nil for a
	// conversation built on its own, and cleared when the store lets it go.
	// Read and written under the store's lock.
	store *conversations

	mu    sync.Mutex
	model string
	turns []turn
	// seqs[i] is when turns[i] was added, in the store's order, so the
	// oldest turn across every channel can be found.
	seqs []uint64
}

// conversations is every channel the bridge has seen while on, bounded.
//
// LOCK ORDER: the store's mu before any conversation's mu, never the other
// way round. A conversation's own methods take only its mu unless they change
// what the store counts, and then they take the store's first.
type conversations struct {
	mu    sync.Mutex
	byID  map[string]*conversation
	order []string
	// total is the bytes of every counted turn; seq orders turns across
	// channels.
	total int
	seq   uint64
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
		c.releaseLocked(c.byID[oldest])
		delete(c.byID, oldest)
	}
	conv := &conversation{store: c}
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
	unlock := conv.lockWithStore()
	defer unlock()
	conv.dropLocked(len(conv.turns))
}

// append adds a turn, cut to maxTurnBytes, holding the slice to maxTurns and
// the store to maxStoreBytes.
func (conv *conversation) append(t turn) {
	t.Content = headBytes(t.Content, maxTurnBytes)
	unlock := conv.lockWithStore()
	defer unlock()
	var seq uint64
	if c := conv.store; c != nil {
		c.seq++
		seq = c.seq
		c.total += len(t.Content)
	}
	conv.turns = append(conv.turns, t)
	conv.seqs = append(conv.seqs, seq)
	if over := len(conv.turns) - maxTurns; over > 0 {
		conv.dropLocked(over)
	}
	if c := conv.store; c != nil {
		c.trimLocked(conv)
	}
}

// lockWithStore takes the store's lock, when the conversation is counted in
// one, and then the conversation's own, in the order every path takes them.
// The store is read under the store's lock, so a conversation the store lets
// go meanwhile is seen as gone.
func (conv *conversation) lockWithStore() (unlock func()) {
	for {
		c := conv.storeRef()
		if c == nil {
			conv.mu.Lock()
			return conv.mu.Unlock
		}
		c.mu.Lock()
		if conv.store != c {
			c.mu.Unlock()
			continue
		}
		conv.mu.Lock()
		return func() { conv.mu.Unlock(); c.mu.Unlock() }
	}
}

// storeRef reads the store pointer without the store's lock, as a hint
// lockWithStore then confirms under it.
func (conv *conversation) storeRef() *conversations {
	conv.mu.Lock()
	defer conv.mu.Unlock()
	return conv.store
}

// dropLocked removes the oldest n turns and gives their bytes back. Both
// locks are held, or only the conversation's when it has no store.
func (conv *conversation) dropLocked(n int) {
	if n <= 0 {
		return
	}
	if c := conv.store; c != nil {
		for _, t := range conv.turns[:n] {
			c.total -= len(t.Content)
		}
	}
	conv.turns = append([]turn(nil), conv.turns[n:]...)
	conv.seqs = append([]uint64(nil), conv.seqs[n:]...)
}

// trimLocked drops the oldest turns across every channel until the store is
// within its budget. mu is held, and so is the lock of held, the
// conversation that just grew; every other conversation's is taken here, in
// the store-first order. The newest turn is never dropped: it is at most
// maxTurnBytes, which is far below the budget.
func (c *conversations) trimLocked(held *conversation) {
	for c.total > maxStoreBytes {
		var oldest *conversation
		var at uint64
		for _, conv := range c.byID {
			if conv != held {
				conv.mu.Lock()
			}
			if len(conv.seqs) > 0 && (oldest == nil || conv.seqs[0] < at) {
				oldest, at = conv, conv.seqs[0]
			}
			if conv != held {
				conv.mu.Unlock()
			}
		}
		if oldest == nil {
			return
		}
		if oldest != held {
			oldest.mu.Lock()
		}
		oldest.dropLocked(1)
		if oldest != held {
			oldest.mu.Unlock()
		}
	}
}

// releaseLocked lets a conversation go from the store: its bytes are no
// longer counted, and an answer still holding it appends to it uncounted,
// bounded by maxTurns and the answers in flight. mu is held.
func (c *conversations) releaseLocked(conv *conversation) {
	if conv == nil {
		return
	}
	conv.mu.Lock()
	for _, t := range conv.turns {
		c.total -= len(t.Content)
	}
	conv.store = nil
	conv.mu.Unlock()
}

// bytes is the store's running total of counted turn text.
func (c *conversations) bytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

// counted adds up every counted turn afresh, for a test to hold the running
// total to.
func (c *conversations) counted() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, conv := range c.byID {
		conv.mu.Lock()
		for _, t := range conv.turns {
			n += len(t.Content)
		}
		conv.mu.Unlock()
	}
	return n
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

// headBytes keeps as much of the start of s as fits in n bytes, cutting on a
// rune boundary so the result is still text.
func headBytes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
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

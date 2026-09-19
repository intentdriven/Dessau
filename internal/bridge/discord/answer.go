package discord

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/intentdriven/Gropius/internal/gateway"
	"github.com/intentdriven/Gropius/internal/stats"
)

// incoming is as much of a MESSAGE_CREATE as this bridge reads.
//
// The content is read because relaying it is what a bridge is
// (adr-2609181004167097 condition 4, and the readers' list in
// internal/archtest admits this package by name for it). Nothing of it is
// logged, recorded or kept on disk.
type incoming struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
	Content   string `json:"content"`
	Author    struct {
		ID  string `json:"id"`
		Bot bool   `json:"bot"`
	} `json:"author"`
	Mentions []struct {
		ID string `json:"id"`
	} `json:"mentions"`
	// Type distinguishes an ordinary message from the dozens of system
	// messages Discord sends down the same event — a pin, a join, a boost.
	// Only the two ordinary kinds are answered.
	Type int `json:"type"`
}

// The two message types this bridge answers: a plain message, and a reply.
const (
	messageTypeDefault = 0
	messageTypeReply   = 19
)

// onMessage decides whether a message is for us and queues the answer.
//
// The decision is made here, on the read loop, and it is deliberately narrow:
// a direct message, or a guild message whose MENTIONS array — Discord's own,
// not the text — names this bot. Text that merely looks like a mention is not
// one, which is what keeps somebody from making the bot answer by typing an
// id; and a message in a channel that does not mention the bot is not read at
// all, which is why the privileged message-content intent is never asked for
// (itd-2609180959397172).
func (s *session) onMessage(ctx context.Context, data json.RawMessage) {
	var msg incoming
	if json.Unmarshal(data, &msg) != nil {
		return
	}
	if msg.Type != messageTypeDefault && msg.Type != messageTypeReply {
		return
	}
	// Never answer a bot, and never answer ourselves: two bots that answer
	// each other are a loop that runs until somebody notices.
	botID, _ := s.resume.who()
	if msg.Author.Bot || msg.Author.ID == "" || msg.Author.ID == botID {
		return
	}
	if !validID(msg.ChannelID) || !validID(msg.Author.ID) {
		// An identifier that is not a snowflake is not something to build a
		// URL from. Discord sends numbers; anything else arrived from
		// somewhere this bridge is not talking to.
		return
	}
	direct := msg.GuildID == ""
	if !direct && !mentions(msg, botID) {
		return
	}
	text := headRunes(msg.Content, maxMessageRunes)
	if text == "" {
		return
	}
	conv := s.convos.get(msg.ChannelID)
	queued := s.submit(func() { s.answer(ctx, conv, msg, text) })
	if !queued {
		// Said once, with nothing of the message in it. A channel busy enough
		// to fill the queue is the one case an operator would want to see.
		s.bridge.log.Info("dropped a bridged message: too many are already being answered",
			"bridge", bridgeName, "channel", numericID(msg.ChannelID))
	}
}

// mentions reports whether Discord itself resolved a mention of this bot.
//
// An empty botID answers false, which is the safe direction — a bot that does
// not know its own id must not answer everything — and is exactly why the id
// has to survive a resume (iss-2609190106402401): READY carries it and
// RESUMED does not.
func mentions(msg incoming, botID string) bool {
	if botID == "" {
		return false
	}
	for _, m := range msg.Mentions {
		if m.ID == botID {
			return true
		}
	}
	return false
}

// validID reports whether an identifier is a Discord snowflake: digits, and
// short enough to be one. Everything this bridge puts in a URL or a log line
// goes through here first.
func validID(id string) bool {
	if id == "" || len(id) > 20 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// numericID renders an identifier for a log line as the number it is, or 0
// when it is not one. The log carries channel and user identifiers as opaque
// numbers and never as names (adr-2609181004167097 condition 4).
func numericID(id string) uint64 {
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// answer runs one bridged request end to end: the typing indicator, the
// request through the gateway's own path, the placeholder filling in, and the
// one log line that says it happened.
func (s *session) answer(ctx context.Context, conv *conversation, msg incoming, text string) {
	// One answer at a time per channel, and a channel that is already being
	// answered is recognised rather than waited on.
	//
	// WAITING HERE IS A DENIAL OF SERVICE (iss-2609190058038096). A generation
	// runs for minutes and there are a small, fixed number of workers; a job
	// that blocked on this lock would hold a worker for the whole of it, so
	// two messages sent into one channel would occupy every worker and no
	// other channel or direct message would be answered until the first
	// answer finished. Anyone who can reach the bot may talk to it, which
	// makes that something one person can cause by pressing send twice.
	if !conv.answering.TryLock() {
		s.say(ctx, msg.ChannelID, msg.ID, stillAnswering)
		return
	}
	defer conv.answering.Unlock()

	model := conv.modelOf()
	if model == "" {
		model = s.defaultModel()
		conv.setModel(model)
	}
	if model == "" {
		s.say(ctx, msg.ChannelID, msg.ID, "This server has no chat model to answer with yet.")
		return
	}

	// Appended and read back under the short lock, not held across the
	// generation: what this channel is about to be asked is decided here, and
	// then the lock is somebody else's to take (iss-2609190106414499).
	conv.append(turn{Role: roleUser, Content: text})
	body, err := buildRequest(model, conv.history(), s.bridge.opts.ServedContext(model))
	if err != nil {
		s.say(ctx, msg.ChannelID, msg.ID, genericProblem)
		return
	}

	ed := newEditor(s.rest, msg.ChannelID, msg.ID, s.bridge.now)
	typingCtx, stopTyping := context.WithCancel(ctx)
	go ed.showTyping(typingCtx, s.rest)

	started := time.Now()
	var answer string
	askErr := s.bridge.opts.Ask(ctx, gateway.AskRequest{
		Model:  model,
		Body:   body,
		Source: stats.SourceBridge,
		OnEvent: func(payload []byte) {
			delta := deltaText(payload)
			if delta == "" {
				return
			}
			if answer == "" {
				// The first token: the placeholder is about to exist, so the
				// typing indicator has done its job.
				stopTyping()
			}
			answer += delta
			ed.add(ctx, delta)
		},
	})
	stopTyping()

	if askErr != nil {
		// The CLASS, not the text (iss-2609190106273104). The gateway's
		// refusal texts describe this Mac — a launch failure carries the
		// child process's own paths, and the pool's refusals name the memory
		// budget in bytes — and a stranger on Discord can drive a refusal at
		// the rate they can send messages, so the text would be a
		// description of this Mac a stranger could write into a file. The
		// gateway logs it there, once, at the level the operator asked for;
		// here only the fixed word travels.
		s.bridge.log.Info("refused a bridged request",
			"bridge", bridgeName, "channel", numericID(msg.ChannelID),
			"user", numericID(msg.Author.ID), "model", model, "reason", refusalClass(askErr))
		if !ed.wrote() {
			s.say(ctx, msg.ChannelID, msg.ID, publicRefusal(askErr))
			return
		}
	}
	if err := ed.finish(ctx); err != nil {
		s.bridge.log.Debug("could not finish a bridged answer",
			"bridge", bridgeName, "channel", numericID(msg.ChannelID), "err", err)
	}
	if answer != "" {
		conv.append(turn{Role: roleAssistant, Content: answer})
	} else if askErr == nil {
		s.say(ctx, msg.ChannelID, msg.ID, "The model answered with nothing.")
	}

	// The one line a bridged request writes. It names the bridge, the channel
	// and the user as numbers, the model, the sizes and the timing — and no
	// part of the message or the answer. The sizes are lengths, which is a
	// count, not content.
	s.bridge.log.Info("bridged a request",
		"bridge", bridgeName,
		"channel", numericID(msg.ChannelID),
		"user", numericID(msg.Author.ID),
		"model", model,
		"prompt_bytes", len(body),
		"answer_bytes", len(answer),
		"duration_ms", time.Since(started).Milliseconds())
}

// genericProblem is what a stranger is told when something went wrong on this
// Mac that is none of their business.
const genericProblem = "Something went wrong answering that."

// stillAnswering is what somebody is told when they send a second message
// into a channel whose answer is still being written.
const stillAnswering = "I am still answering your last message here."

// publicRefusal is what may be repeated to whoever asked.
func publicRefusal(err error) string {
	var askErr *gateway.AskError
	if errors.As(err, &askErr) {
		return askErr.Public()
	}
	return genericProblem
}

// refusalClass is the fixed word a refusal is recorded under. It is the
// gateway's own class where there is one, so the log's vocabulary is the
// gateway's rather than a second one kept in step by hand.
func refusalClass(err error) string {
	var askErr *gateway.AskError
	if errors.As(err, &askErr) {
		return askErr.Class()
	}
	return "error"
}

// say posts one message, best effort. It is used for the refusals and the
// answers that are not a stream.
func (s *session) say(ctx context.Context, channelID, replyTo, text string) {
	if _, _, err := s.rest.createMessage(ctx, channelID, text, replyTo); err != nil {
		s.bridge.log.Debug("could not post a bridged message",
			"bridge", bridgeName, "channel", numericID(channelID), "err", err)
	}
}

// deltaText reads the generated text out of one streamed event.
//
// This is the other half of what the bridge is admitted to read: the answer it
// is posting. It reads the delta's text and the non-streamed field beside it
// (some servers send the whole piece rather than a delta on the first event),
// and nothing else in the event.
func deltaText(payload []byte) string {
	var ev struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			Text string `json:"text"`
		} `json:"choices"`
	}
	if json.Unmarshal(payload, &ev) != nil || len(ev.Choices) == 0 {
		return ""
	}
	if c := ev.Choices[0].Delta.Content; c != "" {
		return c
	}
	return ev.Choices[0].Text
}

// defaultModel is the model a channel starts on: the first the server offers.
func (s *session) defaultModel() string {
	models := s.bridge.opts.ChatModels()
	if len(models) == 0 {
		return ""
	}
	return models[0]
}

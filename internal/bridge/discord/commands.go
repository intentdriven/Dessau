package discord

import (
	"context"
	"encoding/json"
	"strings"
)

// The two application commands this bridge registers.
const (
	commandModel = "model"
	commandReset = "reset"
)

// commandDefinitions is what is registered with Discord, once per start.
//
// `/model` takes one optional string option, so running it bare asks what the
// channel is on and running it with a name sets it. The option is a plain
// string rather than a set of choices because the models on this Mac change
// as the operator downloads them, and a registered choice list would be stale
// the moment they did.
func commandDefinitions() []any {
	return []any{
		map[string]any{
			"name":        commandModel,
			"description": "Show or set the model that answers in this channel",
			"type":        1,
			"options": []any{
				map[string]any{
					"name":        "name",
					"description": "The model to answer with",
					"type":        3,
					"required":    false,
				},
			},
		},
		map[string]any{
			"name":        commandReset,
			"description": "Forget this channel's conversation and start again",
			"type":        1,
		},
	}
}

// registerCommands publishes the commands for this application.
//
// A failure is logged and nothing else: the bridge still answers messages and
// mentions, which is the part somebody is waiting on, and a command that
// could not be registered is a thing the operator can see in the log rather
// than a reason to refuse the whole bridge.
func (s *session) registerCommands(ctx context.Context) {
	if err := s.rest.overwriteCommands(ctx, s.appID, commandDefinitions()); err != nil {
		s.bridge.log.Info("could not register the bridge's slash commands",
			"bridge", bridgeName, "err", err)
	}
}

// interaction is as much of an INTERACTION_CREATE as this bridge reads.
type interaction struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Type      int    `json:"type"`
	ChannelID string `json:"channel_id"`
	Data      struct {
		Name    string `json:"name"`
		Options []struct {
			Name  string          `json:"name"`
			Value json.RawMessage `json:"value"`
		} `json:"options"`
	} `json:"data"`
}

// interactionCommand is the interaction type for a slash command being run.
const interactionCommand = 2

// onInteraction answers a slash command.
//
// Discord gives a bot three seconds to answer an interaction or it shows the
// person an error, so this never queues behind the answers: it takes a place
// of its own, does one REST call and returns. A full queue drops the
// interaction, which Discord shows as a command that did not respond — the
// honest outcome, and better than answering four seconds late into a
// conversation that has moved on.
func (s *session) onInteraction(ctx context.Context, data json.RawMessage) {
	var in interaction
	if json.Unmarshal(data, &in) != nil || in.Type != interactionCommand {
		return
	}
	if in.ID == "" || in.Token == "" || !validID(in.ChannelID) {
		return
	}
	if !s.submit(func() { s.runCommand(ctx, in) }) {
		s.bridge.log.Debug("dropped a slash command: too many requests are already being answered",
			"bridge", bridgeName, "channel", numericID(in.ChannelID))
	}
}

func (s *session) runCommand(ctx context.Context, in interaction) {
	conv := s.convos.get(in.ChannelID)
	var reply string
	switch in.Data.Name {
	case commandReset:
		conv.reset()
		reply = "This channel's conversation is cleared."
	case commandModel:
		reply = s.modelCommand(conv, in)
	default:
		return
	}
	if err := s.rest.respondToInteraction(ctx, in.ID, in.Token, reply); err != nil {
		s.bridge.log.Debug("could not answer a slash command",
			"bridge", bridgeName, "channel", numericID(in.ChannelID), "err", err)
	}
}

// modelCommand shows or sets the model answering a channel.
//
// A name that is not one of the server's chat models is refused with the list,
// rather than being stored and failing at the next message. The list is what
// this Mac serves, which is a fact about the models rather than about the
// machine, and is the same list /v1/models publishes to every client.
func (s *session) modelCommand(conv *conversation, in interaction) string {
	models := s.bridge.opts.ChatModels()
	asked := strings.TrimSpace(optionString(in, "name"))
	if asked == "" {
		current := conv.modelOf()
		if current == "" {
			current = s.defaultModel()
		}
		if current == "" {
			return "This server has no chat models to offer."
		}
		return "This channel is answered by `" + current + "`.\n" + modelList(models)
	}
	for _, m := range models {
		if strings.EqualFold(m, asked) || strings.EqualFold(shortName(m), asked) {
			conv.setModel(m)
			return "This channel is now answered by `" + m + "`."
		}
	}
	return "That is not a model this server offers.\n" + modelList(models)
}

// modelList renders the models on offer, bounded: a server with a hundred
// models would otherwise produce a reply too long for Discord to accept.
func modelList(models []string) string {
	if len(models) == 0 {
		return "This server has no chat models to offer."
	}
	const most = 25
	shown := models
	suffix := ""
	if len(shown) > most {
		shown = shown[:most]
		suffix = "\n…and more."
	}
	return "Available: `" + strings.Join(shown, "`, `") + "`" + suffix
}

// shortName is the part of a repo id after the slash, which is how most
// OpenAI-compatible clients show a model and how somebody will type it.
func shortName(repoID string) string {
	if i := strings.LastIndex(repoID, "/"); i >= 0 {
		return repoID[i+1:]
	}
	return repoID
}

// optionString reads one string option off a command, bounded to what a model
// name can be. It is somebody else's text arriving over the network, so it is
// read as a string or not at all.
func optionString(in interaction, name string) string {
	for _, opt := range in.Data.Options {
		if opt.Name != name {
			continue
		}
		var value string
		if json.Unmarshal(opt.Value, &value) != nil {
			return ""
		}
		return headRunes(value, maxModelNameRunes)
	}
	return ""
}

// maxModelNameRunes bounds what is taken from a `/model` argument. A repo id
// is short; anything longer is not one, and this is the value that would
// otherwise be echoed back into a reply.
const maxModelNameRunes = 200

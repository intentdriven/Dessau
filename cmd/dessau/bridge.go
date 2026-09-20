package main

import (
	"log/slog"
	"sort"

	"github.com/intentdriven/Dessau/internal/app"
	"github.com/intentdriven/Dessau/internal/bridge/discord"
	"github.com/intentdriven/Dessau/internal/gateway"
	"github.com/intentdriven/Dessau/internal/registry"
)

// newDiscordBridge wires the Discord bridge to this server.
//
// It lives here rather than in internal/app because the bridge asks the
// gateway for completions and the gateway is built from the app, so the app
// cannot build the bridge without the import running in a circle. The three
// closures below are the whole of what the bridge knows about this Mac: how
// to ask for a completion, which models a channel may pick, and how large a
// window each is served at.
//
// Building it opens no connection. The bridge is off until App.SetBridge puts
// the stored settings in force, and off is what the stored settings say until
// the operator turns it on (adr-2609181004167097 condition 1).
func newDiscordBridge(a *app.App, g *gateway.Gateway, log *slog.Logger) app.Bridge {
	return discord.New(discord.Options{
		Ask:           g.Ask,
		ChatModels:    func() []string { return chatModels(a) },
		ServedContext: func(model string) int64 { return servedContext(a, model) },
		Log:           log,
	})
}

// chatModels is the models a channel may be answered by: the ready models this
// Mac holds that the chat rule calls conversational, in a stable order.
//
// The same rule /v1/models publishes `chat` from, read live so a model
// downloaded while the bridge is running is offered without a restart. Sorted
// by repo id rather than left in the registry's order, because the first
// entry is the default a channel starts on and a default that moved when an
// unrelated model finished downloading would be a default nobody chose.
func chatModels(a *app.App) []string {
	rule := a.Config().EffectiveChatRule()
	var out []string
	for _, m := range a.Registry.Ready() {
		if rule.Matches(m.PipelineTag, m.Tags) {
			out = append(out, m.RepoID)
		}
	}
	sort.Strings(out)
	return out
}

// servedContext is the window a model is served at on this Mac, as the app
// resolves it: the operator's setting, or the default derived to fit the
// budget when they have set none. It is the figure the gateway judges a
// request against, so it is the figure the bridge bounds a conversation to.
func servedContext(a *app.App, model string) int64 {
	m, err := a.Registry.Get(model)
	if err != nil {
		m = registry.Model{RepoID: model}
	}
	window, _ := a.ServedWindow(m)
	return window
}

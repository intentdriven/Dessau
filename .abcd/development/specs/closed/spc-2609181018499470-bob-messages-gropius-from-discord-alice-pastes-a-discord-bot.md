---
id: spc-2609181018499470
slug: bob-messages-gropius-from-discord-alice-pastes-a-discord-bot
intent: itd-2609180959397172
origin: researcher-authored
production_mode: hand-written
---
# Bob messages Gropius from Discord

## Summary

A new package, `internal/bridge/discord`, keeps one outbound WebSocket to
Discord's gateway while the bridge is switched on, answers direct messages
and mentions with a model on this Mac through the gateway's own request
path, streams the answer by editing a placeholder message on a throttle read
from Discord's rate-limit headers, and lets a channel choose its model with
`/model` and clear its history with `/reset`. The switch, the token and the
bridge's state live on the three surfaces; the token is a secret on every
one; nothing of a message is recorded. The gateway is hand-rolled on one
zero-dependency module, `github.com/coder/websocket` (the maintainer's
sign-off, 2026-09-18), and speaks API v10.

## Scope

In scope: `internal/bridge/discord` (gateway session, REST calls, the
streaming editor, the per-channel conversations, the two slash commands);
`internal/config` (`discord_bridge` on/off, `discord_token`, the token's
three secret obligations); `internal/app` (start, stop and reconnect on
settings apply; the bridge's state on the snapshot); `internal/gateway`
(a `source` on the request record and a way for the bridge to make a
completion request in-process with the model, the messages and the
streaming callback, without a network hop); `internal/stats` (the fixed
`source` class on the request record and its line on the reference page);
`internal/ui` (the Settings pane: switch, token field, the egress
sentence, last-connected state; the archtest that holds the sentence to the
switch); `internal/archtest` (the readers' list admits the bridge package
by name; the no-content test covers the bridge's log lines; the token's
three tests); `docs/discord-bridge.md` (a how-to: make the bot in Discord's
portal with `bot_public` off, the invite link's permissions, paste the
token, what leaves the Mac); the changelog.

Out of scope: the privileged message-content intent (a channel message that
does not mention the bot is never read); embeds; threads created by the
bridge (a later refinement); WhatsApp (iss-2609180959406917); an allow-list
(addable as a setting without revisiting adr-2609181004167097); persistence
of any conversation.

## Approach

### The session

`Session` opens `wss://gateway.discord.gg/?v=10&encoding=json`, reads Hello,
sends Identify with intents `GUILD_MESSAGES | DIRECT_MESSAGES` (neither
privileged) and the token, heartbeats on the interval with jitter, and keeps
`session_id`, `resume_gateway_url` and the last sequence for Resume. Close
codes decide the retry: resumable codes resume with backoff; 4004 (bad
token) and 4014 (disallowed intents) stop the bridge and put the reason on
the panel; Identify is rate-limited to the budget Discord states. The
session lives in one goroutine owned by `Bridge`, which `App` starts when
the switch is on at start or turns on at a save and stops when it turns off,
under the lock order of adr-2609091239058072 (the settings handler's lock,
then the save lock; the bridge takes neither).

### Answering

`MESSAGE_CREATE` for a direct message, or a guild message whose mentions
include the bot's user id, becomes a turn in the channel's `Conversation`:
the most recent turns that fit the channel's model's served window, judged
with the gateway's own estimate, so the gateway never refuses for size.
The request is made in-process through a new `gateway.Ask` that takes the
model, the messages and a streaming callback and runs the same admission,
merge and observation as an HTTP request would; `source` is `bridge`. The
bridge posts a typing indicator every eight seconds until the first token,
creates the placeholder message on the first token, then edits it with the
accumulated text no more often than the rate-limit headers allow and at
least every two seconds, cutting at a paragraph before 2,000 characters
and continuing in a new message. A gateway refusal reaches Discord as the
generic refusal text (`genericRefusal`) and the log as the detailed one.

`/model` and `/reset` are application commands registered once per start
with the bot's application id, received over the gateway as
`INTERACTION_CREATE` and answered over HTTPS within Discord's three
seconds; `/model` with no argument shows the channel's model, with one sets
it from the server's chat models (the server's default to begin with).

### The token

`config.Config.DiscordToken` joins `APIKey` and `HFToken` in the three
places the API key already is: the redaction in snapshots, the panel and
`gropius config show`; the placeholder swap on save, so an unrelated save
never overwrites it with the placeholder; and `secretSettingKeys`, so it is
never compared against a posted value. A token that is absent, malformed or
rejected by Discord never refuses a save: the bridge alone reports its state
on the snapshot (`bridge: off | connecting | connected since | stopped:
reason`).

### What is recorded

The request record gains one fixed class, `source`, valued `http` or
`bridge`; the reference page names it. The log line for a bridged request
carries the bridge's name, the channel and user identifiers as numbers, the
model, the sizes and the timing, at the level the existing request line
uses; the prompt-content readers' test admits `internal/bridge/discord` by
name with its reason, and the no-content test walks its log lines.

## How the acceptance criteria are met

1. Typing within two seconds, placeholder at the first token — the typing
   loop and the editor.
2. Saves never refused over the token — the secret obligations and the
   state on the snapshot; `control_untouched_test` gains the token.
3. Mentions answered, unmentioned messages unread — the intents chosen and
   the mention check.
4. Long answers continue in a second message — the editor's cut.
5. `/model` per channel — the command and the `Conversation`'s model.
6. Switch off closes the connection — `Bridge.Stop`.
7. No content in log or store — the tests named above.
8. The token redacted, round-tripped and excluded from comparison — three
   tests beside the API key's.
9. Resume across sleep — the session's resume path; checked by hand with a
   sleeping Mac and recorded in the shipping decision line.
10. The egress sentence beside the switch — the pane and its archtest; the
    docs page.
11. Bounded history and `/reset` — the `Conversation`.

## Verification

`make test`, gofmt, vet, docs lint; a fake gateway in the test package that
speaks Hello, Ready, heartbeats and Resume, so the session's state machine
and the editor's throttle are tested without Discord; the live checks (a
real bot, a mention, a long answer, a sleep) by hand on the maintainer's Mac,
recorded in the shipping decision line. Security review before presentation,
as adr-2609181004167097 obliges.

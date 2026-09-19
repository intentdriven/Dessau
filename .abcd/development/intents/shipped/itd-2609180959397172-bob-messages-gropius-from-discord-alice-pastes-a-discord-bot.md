---
id: itd-2609180959397172
slug: bob-messages-gropius-from-discord-alice-pastes-a-discord-bot
spec_id: spc-2609181018499470
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob messages Gropius from Discord

## Press Release

Bob messages Gropius from Discord. Alice pastes a Discord bot token into
Settings, reads the sentence beside it that says messages to the bot and the
model's answers pass through Discord and are kept under its terms, and
switches the bridge on. From then on a direct message to that bot, or a
mention of it in a channel it has been invited to, is answered by a model on
her Mac: a placeholder appears at once and fills in as the reply is written,
long answers continue in a second message, and the conversation carries on
per channel or direct message — anyone who can reach the bot may talk to
it. In any channel, `/model` lists the chat models Alice's server offers and
sets which one answers there. The bridge connects out from the Mac to
Discord; nothing on the Mac is opened to the internet, and switching the
bridge off closes the connection. Alice sees every bridged request in the
log and the statistics like any other, by channel and user identifier, never
by content. Bob, on his phone, chats with Alice's models from an app he
already has.

## Why This Matters

Gropius has no phone client and no way in for someone who will not install
one. Discord is where many people already are, and a bot is the smallest
possible client: nothing to install, nothing to configure, on every device
they own. The price — the conversation passes through Discord — is stated
where the switch is, and decided in adr-2609181004167097 before any code.

## Mechanism

We expect a Discord bot to reach a Mac behind a home router because the
Gateway is one outbound WebSocket the bot opens itself, over which direct
messages, mentions and slash-command interactions all arrive, and answers go
out over HTTPS; and we expect direct messages and mentions to need no
privileged intent because Discord delivers their content to any bot without
one. Streaming is a placeholder message edited on a throttle the server
reads from Discord's rate-limit headers, cut at a paragraph before 2,000
characters. What would show this wrong: Discord delivering empty content
for mentions without the privileged intent after all; an edit cadence the
rate limits will not bear; or the gateway session failing to resume across
Alice's Mac sleeping.

## Scope Conditions

- Discord only, under adr-2609181004167097, which decides the egress: the <!-- cond: cond-2609181018499668 -->
  bridge is opt-in and off by default, and connects outbound only.
- The token is a secret, redacted everywhere the API key is, as <!-- cond: cond-2609181018491359 -->
  itd-2609081259493890 established for the key.
- The bridge reads what it must: Bob's message to build the request and <!-- cond: cond-2609181018495407 -->
  the streamed answer to edit the Discord message. That makes it the first
  second reader of prompt and answer content beside the merge that
  adr-2609061610102325 grants, and the architecture test that holds the
  readers' list admits the bridge package by name with that reason; a
  separate test holds that no record and no log line carries a message's
  content.
- The statistics record of a bridged request carries one new fixed class, <!-- cond: cond-2609181018498110 -->
  the source (`bridge`), and nothing a platform supplied: the channel and
  user identifiers go to the log line only, as opaque numbers, never names,
  so the store keeps its rule that nothing in it is a person's or a client's
  to write.
- A conversation's history is bounded: per channel the bridge keeps the <!-- cond: cond-2609181018490230 -->
  most recent turns that fit the model's served window, and `/reset` clears
  it; a refusal from the gateway reaches Discord as the generic refusal and
  the log as the detailed one, so no operator guidance is relayed to a
  stranger.
- The egress is stated beside the switch, in the pane and in the docs. <!-- cond: cond-2609181018494794 -->
- WhatsApp is held for its own intent (iss-2609180959406917). <!-- cond: cond-2609181018495140 -->
- Anyone who can reach the bot may talk to it; reach is Alice's to control <!-- cond: cond-2609181018499374 -->
  by where she invites it and by the portal's "public bot" switch, which the
  docs tell her to turn off. The maintainer's decision at the interview,
  2026-09-18.
- Each channel or direct message is its own conversation, held in memory <!-- cond: cond-2609181018490128 -->
  and gone when the bridge stops; nothing of a message is kept on disk.
- The model that answers a channel is set there with `/model`, from the <!-- cond: cond-2609181018495938 -->
  server's chat models, starting from the server's default; the maintainer's
  decision at the interview.
- Direct messages and mentions only: a message in a channel that does not <!-- cond: cond-2609181018495500 -->
  mention the bot is not read, so the privileged message-content intent is
  never requested.
- The dependency: a bridge needs one WebSocket client the standard library <!-- cond: cond-2609181018497578 -->
  lacks; whether that is a hand-rolled gateway on one small library or an
  existing Discord library is the maintainer's sign-off at planning, under
  the repository's rule for new dependencies.
- The bind mode and every inbound surface are untouched; the bridge lives <!-- cond: cond-2609181018498135 -->
  in the Go server under the three-surfaces rule, with its switch, token and
  state on the panel, in `config.json` and in Go alike.

## Acceptance Criteria

- Given a token pasted in Settings and the bridge switched on, when Bob
  sends the bot a direct message, then the bot shows typing within two
  seconds, a placeholder reply appears with the model's first token — after
  the load, when the model is cold — and fills in as the model writes, and
  the finished reply is the model's answer.
- Given a channel whose history has grown past the model's window, when Bob
  sends the next message, then the oldest turns are dropped and the reply
  arrives; and when he runs `/reset`, then the channel starts afresh.
- Given a bridge token that is absent, malformed or rejected by Discord,
  when Alice saves an unrelated setting, then the save lands and the bridge
  alone reports its state on the panel; a save is never refused over the
  bridge.
- Given the bot mentioned in a channel, when Bob's message mentions it, then
  it answers in that channel; when a message there does not mention it,
  then nothing is read or answered.
- Given a reply longer than a Discord message, when the model keeps
  writing, then the text continues in a second message cut at a paragraph,
  and no message exceeds Discord's limit.
- Given Bob runs `/model` in a channel, when he picks a model, then the next
  message there is answered by it, and `/model` with no choice shows the
  channel's current one.
- Given the bridge is switched off, when the save lands, then the
  connection closes, the bot shows offline, and no message is answered
  until it is switched on again.
- Given a bridged request, when it is recorded, then the log line and the
  statistics record carry the bridge's name, the channel and user
  identifiers, the model, the sizes and the timing, and no prompt or answer;
  held by the existing no-content tests extended to the bridge.
- Given the token, when it would appear in any snapshot, log, panel page
  or `gropius config show`, then it is redacted the way the API key is
  (itd-2609081259493890); when the operator saves any other setting, then
  the redaction placeholder is swapped back for the stored token and the
  token is never compared against a posted value; three tests, one for
  each obligation, beside the API key's.
- Given Alice's Mac sleeps and wakes, when the gateway session drops, then
  the bridge resumes or reconnects by itself and the panel shows when it
  last connected.
- Given the Settings pane, when the token field is shown, then the sentence
  stating what leaves the Mac is beside the switch, and `docs/` has a page
  that says the same and how to make the bot in Discord's portal.

## Open Questions

- The dependency sign-off: a hand-rolled gateway on a zero-dependency
  WebSocket library, or discordgo (one dependency, on API v9, quiet since
  February 2026); disgo is nine modules and out.

## Grounds

- pursued: we expect a Discord bot to bring Alice's models to people who would never install a client, and to be the phone client this project does not have — wrong if nobody but Alice ever messages it, or if the round trip through Discord is too slow or lossy to feel like chat

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-91c0608082bd -->
Fidelity review OWED (receipt rcp-91c0608082bd).

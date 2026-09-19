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

<!-- abcd-review: INGESTED receipt=rcp-91c0608082bd -->
Fidelity review — receipt rcp-91c0608082bd (verifier intent-auditor claude-opus-5[1m]).

Provenance: intent-auditor@claude-opus-5[1m] · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:18700aeed9c3a77f177d044ff69a0db462256cdd3d62a28a73a6286c43fcc105
Input attestations: tree:origin/main...origin/feat/discord-bridge (PR 94, 15 commits, 63 files) as carried by this worktree's HEAD; every file:line pointer is read from the shipped tree in the worktree, not from the diff@-; intent:.abcd/development/intents/shipped/itd-2609180959397172-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md@-; spec:.abcd/development/specs/closed/spc-2609181018499470-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md@-; request:.abcd/.work.local/reviews/rcp-91c0608082bd.request.md@-; test-run:go test ./internal/bridge/... ./internal/archtest/ ./internal/app/ ./internal/gateway/ ./internal/stats/ ./internal/lifecycle/ — all six packages ok@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 6 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The typing indicator is posted once immediately and then every eight seconds, the placeholder message is created on the first delta and edited as the answer grows, and a fake-gateway test shows the posted content is the model's text; but every part of this that is about Discord itself — a real bot, a real DM, the indicator appearing within two seconds on a phone — is owed by hand and the record names it OWED, and under load the answer job waits behind two workers before any typing is shown.
  evidence: internal/bridge/discord/editor.go:209 — "_ = r.typing(ctx, e.channelID)"
  evidence: internal/bridge/discord/answer_test.go:103 — "func TestTypingIsShownBeforeTheFirstToken(t *testing.T) {"
  evidence: internal/bridge/discord/answer_test.go:30 — "func TestADirectMessageIsAnswered(t *testing.T) {"
- ac-2 — MET: buildRequest drops turns from the front until the encoded body fits the model's served window, with a test that grows a history past the window and checks the oldest turns went and the newest survived, and /reset clears the channel's turns with tests on both the command path and the conversation.
  evidence: internal/bridge/discord/conversation.go:188 — "for start := 0; start < len(turns); start++ {"
  evidence: internal/bridge/discord/conversation_test.go:14 — "func TestAConversationIsTrimmedToTheServedWindow(t *testing.T) {"
  evidence: internal/bridge/discord/commands.go:118 — "conv.reset()"
  evidence: internal/bridge/discord/conversation_test.go:123 — "func TestResetClearsTheHistoryAndKeepsTheModel(t *testing.T) {"
- ac-3 — MET: Validate says nothing about the token, the control plane accepts a save carrying an empty, malformed, overlong or accented token with 200, the app-level save lands the unrelated setting, and the bridge alone reports 'no bot token' or Discord's 4004 refusal as its own state.
  evidence: internal/gateway/control_discord_test.go:121 — "func TestASaveIsNeverRefusedOverTheDiscordToken(t *testing.T) {"
  evidence: internal/app/bridge_test.go:140 — "func TestASaveIsNeverRefusedOverTheBridgesSettings(t *testing.T) {"
  evidence: internal/bridge/discord/bridge.go:192 — "b.setStateLocked(StateStopped, "no bot token — paste one in Settings")"
  evidence: internal/bridge/discord/session.go:506 — "return sessionOutcome{fatal: "Discord refused the bot token — paste a fresh one in Settings"}"
- ac-4 — MET_WITH_CONCERNS: Identify asks for GUILD_MESSAGES|DIRECT_MESSAGES and neither privileged intent, a guild message is answered only when Discord's own resolved mentions array names the bot, and a test shows text that merely spells a mention is not answered; that no content is delivered for an unmentioned channel message is a fact about Discord's servers that no test here can establish and that the record itself names as what would show the mechanism wrong.
  evidence: internal/bridge/discord/session.go:48 — "const intents = 1<<9 | 1<<12"
  evidence: internal/bridge/discord/answer.go:74 — "if !direct && !mentions(msg, botID) {"
  evidence: internal/bridge/discord/answer_test.go:48 — "func TestAChannelMessageIsAnsweredOnlyWhenItMentionsTheBot(t *testing.T) {"
- ac-5 — MET: The editor cuts whenever the pending text passes 2,000 runes, preferring a paragraph then a line then a space within a 1,900-rune window, and continues in a new message; one test drives a ~3,000-character answer through it and asserts both messages are non-empty, under the limit and not cut mid-word, and a table test covers the fallbacks.
  evidence: internal/bridge/discord/editor.go:79 — "for e.pendingRunes > messageLimit {"
  evidence: internal/bridge/discord/answer_test.go:150 — "func TestALongAnswerIsCutAtAParagraphAndContinues(t *testing.T) {"
  evidence: internal/bridge/discord/conversation_test.go:139 — "func TestTheCutPrefersAParagraphAndNeverExceedsTheLimit(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: A matching /model stores the model on the channel's conversation and the next answer reads it back before asking the gateway, and a test covers bare /model showing the current model and an unknown name being refused without echoing it — but the fake server offers exactly one chat model, so nothing exercises picking a SECOND model and seeing the next message answered by it, and the stored choice is lost at every reconnect because the conversations are rebuilt per gateway session.
  evidence: internal/bridge/discord/commands.go:152 — "conv.setModel(m)"
  evidence: internal/bridge/discord/answer.go:155 — "model := conv.modelOf()"
  evidence: internal/bridge/discord/answer_test.go:253 — "func TestTheSlashCommandsAnswerTheChannel(t *testing.T) {"
  evidence: internal/bridge/discord/fake_test.go:92 — "ChatModels: func() []string { return []string{"mlx-community/Qwen3-8B-4bit"} },"
  evidence: internal/bridge/discord/session.go:176 — "convos: newConversations(),"
- ac-7 — MET_WITH_CONCERNS: Apply(false, "") cancels the session and waits for its goroutine, the state goes to off, and a test shows a message sent afterwards produces no REST call at all; 'the bot shows offline' is Discord's own presence and is only observable against a real bot, which the record lists among the checks owed by hand.
  evidence: internal/bridge/discord/session_test.go:182 — "func TestSwitchingTheBridgeOffClosesTheConnection(t *testing.T) {"
  evidence: internal/bridge/discord/bridge.go:237 — "func (b *Bridge) stopLocked() {"
- ac-8 — MET_WITH_CONCERNS: The one log line carries the bridge's name, the channel and user as opaque numbers, the model, both byte counts and the duration, held both by an AST scan of every log call in the package and by a live test asserting no message or answer text reaches the log; the divergence is the statistics record, which carries only the new fixed 'source' class and NOT the bridge's name or the channel and user identifiers the criterion promises — those reach the log alone, which is what the intent's own scope condition requires.
  evidence: internal/bridge/discord/answer.go:232 — "s.bridge.log.Info("bridged a request","
  evidence: internal/archtest/bridge_content_test.go:127 — "func TestTheBridgeRecordsTheFactOfABridgedRequest(t *testing.T) {"
  evidence: internal/archtest/bridge_content_test.go:65 — "func TestTheBridgeWritesNoMessageContent(t *testing.T) {"
  evidence: internal/stats/stats.go:157 — "Source Source `json:"source"`"
  evidence: internal/gateway/ask_test.go:60 — "func TestAskIsRecordedAsABridgedRequest(t *testing.T) {"
- ac-9 — MET: The token is blanked in the snapshot's redactConfig beside the API key and the HuggingFace token, swapped back from the placeholder on save, added to secretSettingKeys so it is never compared with a posted value, and masked by `gropius config show`; the three tests the criterion asks for exist, one per obligation, beside the API key's.
  evidence: internal/gateway/control.go:839 — "c.DiscordToken = "********""
  evidence: internal/gateway/control_discord_test.go:17 — "func TestTheDiscordTokenIsRedactedInState(t *testing.T) {"
  evidence: internal/gateway/control_discord_test.go:43 — "func TestSavingTheRedactedPlaceholderKeepsTheRealDiscordToken(t *testing.T) {"
  evidence: internal/gateway/control_discord_test.go:87 — "func TestARefusalDoesNotSayWhetherAGuessedDiscordTokenWasRight(t *testing.T) {"
  evidence: internal/lifecycle/config.go:50 — "{"discord_token", func(c config.Config) string { return c.DiscordToken }, func(c *config.Config, v string) { c.DiscordToken = v }},"
- ac-10 — MET_WITH_CONCERNS: A dropped connection is resumed with Discord's session id and the last sequence seen, the bot's own id survives the resume so mentions are still answered afterwards, and the panel renders 'Connected since < moment>' from BridgeState.Since; but a resume across a Mac actually sleeping and waking is explicitly OWED by hand in the record, and the resume silently discards every channel's history and /model choice because the conversations belong to the session object.
  evidence: internal/bridge/discord/session_test.go:99 — "func TestADroppedSessionIsResumedWithTheSequenceItHadSeen(t *testing.T) {"
  evidence: internal/bridge/discord/session_test.go:279 — "func TestAMentionIsStillAnsweredAfterAResume(t *testing.T) {"
  evidence: internal/app/bridge.go:42 — "Since int64 `json:"since,omitempty"`"
  evidence: internal/ui/static/app.js:1103 — "? `Connected since ${new Date(b.since * 1000).toLocaleString()}.`"
  evidence: internal/bridge/discord/session.go:176 — "convos: newConversations(),"
- ac-11 — MET: The egress sentence sits inside the same Settings fieldset as the switch and the token field, an architecture test scoped to that fieldset (not the page) holds each phrase, and docs/discord-bridge.md says the same thing and walks through making the bot in Discord's portal with Public Bot off.
  evidence: internal/ui/static/index.html:589 — "and the model&rsquo;s answers pass through Discord and are kept under Discord&rsquo;s"
  evidence: internal/archtest/bridge_egress_test.go:24 — "func TestTheBridgeSwitchCarriesTheEgressSentence(t *testing.T) {"
  evidence: internal/archtest/bridge_egress_test.go:67 — "func TestTheBridgeDocsPageSaysWhatLeavesTheMac(t *testing.T) {"
  evidence: docs/discord-bridge.md:8 — "**What leaves this Mac.** While the bridge is on, messages to the bot and the"
  evidence: docs/discord-bridge.md:23 — "3. Turn **Public Bot** *off*. While it is on, anybody who finds your"

Gap audit:
- honoured:
  - The bridge is opt-in and off by default, and opens no connection until the switch is thrown.
    evidence: internal/bridge/discord/session_test.go:38 — "func TestTheBridgeOpensNothingUntilItIsSwitchedOn(t *testing.T) {"
    evidence: internal/config/config.go:592 — "DiscordBridge bool `json:"discord_bridge"`"
  - The bot token is a secret on every surface the API key is, and no save is ever refused over it.
    evidence: internal/gateway/control.go:839 — "c.DiscordToken = "********""
    evidence: internal/gateway/control_discord_test.go:121 — "func TestASaveIsNeverRefusedOverTheDiscordToken(t *testing.T) {"
  - No part of a message or an answer reaches a log line, a record or the disk.
    evidence: internal/archtest/bridge_content_test.go:65 — "func TestTheBridgeWritesNoMessageContent(t *testing.T) {"
    evidence: internal/bridge/discord/answer.go:232 — "s.bridge.log.Info("bridged a request","
  - The egress sentence is beside the switch in the pane and repeated on a docs page, held by a fieldset-scoped test.
    evidence: internal/archtest/bridge_egress_test.go:24 — "func TestTheBridgeSwitchCarriesTheEgressSentence(t *testing.T) {"
    evidence: docs/discord-bridge.md:8 — "**What leaves this Mac.** While the bridge is on, messages to the bot and the"
  - A long answer is cut at a paragraph and continued, and no message exceeds Discord's limit.
    evidence: internal/bridge/discord/editor.go:79 — "for e.pendingRunes > messageLimit {"
    evidence: internal/bridge/discord/answer_test.go:150 — "func TestALongAnswerIsCutAtAParagraphAndContinues(t *testing.T) {"
  - A bridged request takes the gateway's own admission, size judgement, model rewrite, merge and observation, with only the record's source differing.
    evidence: internal/gateway/ask.go:44 — "func (g *Gateway) Ask(ctx context.Context, req AskRequest) error {"
    evidence: internal/gateway/ask_test.go:60 — "func TestAskIsRecordedAsABridgedRequest(t *testing.T) {"
- diverged:
  - The criterion promises the statistics record carries the bridge's name and the channel and user identifiers; the delivered record carries only the new fixed 'source' class, and the identifiers reach the log alone.
    evidence: internal/stats/stats.go:157 — "Source Source `json:"source"`"
    evidence: internal/bridge/discord/answer.go:232 — "s.bridge.log.Info("bridged a request","
  - The scope condition says a gateway refusal reaches Discord as the generic refusal; the over-the-window refusal travels as its own public sentence instead.
    evidence: internal/gateway/ask.go:209 — "const tooLongRefusal = "that conversation is too long for this model""
  - A channel's conversation and its /model choice are discarded on every reconnect, not only when the bridge stops, because the conversations belong to the per-connection session object.
    evidence: internal/bridge/discord/session.go:176 — "convos: newConversations(),"
    evidence: internal/bridge/discord/session.go:191 — "// survives a resumed session only as long as the session object does — which"
  - A channel starts on the alphabetically first ready chat model rather than a server default, because no server default model exists.
    evidence: cmd/gropius/bridge.go:49 — "sort.Strings(out)"
- missing:
  - Every live check against real Discord is owed and was not performed: a real bot with a real token connecting, a mention in a real channel answered, a long answer as Discord renders it, a resume across the Mac actually sleeping and waking, the slash commands in Discord's own UI, and the edit cadence against Discord's real rate limits. The shipping record names them OWED and never claims them done.
    evidence: .abcd/work/DECISIONS.md:301 — "- 2026-09-19 — **The Discord bridge is SHIPPED on feat/discord-bridge** (itd-2609180959397172, spc-2609181018499470, under adr-2609181004167097; fidelity receipt rcp-91c0608082bd is OWED). A new `internal/bridge/discord` keeps one outbound WebSocket to Discord's gateway at v10 while the switch is on — Hello, Identify with the two unprivileged intents, jittered heartbeats, Resume across a drop, and 4004 and 4014 stopping the bridge with the reason on the panel — and answers direct messages and mentions through a new `gateway.Ask`: the same admission, size judgement, load-bearing model rewrite, merge and observation an HTTP request takes, with the record's one new fixed class, the `source`, saying a bridge brought it. The answer is a placeholder edited no faster than every two seconds and no faster than the rate-limit headers allow, cut at a paragraph before 2,000 characters. The dependency is `github.com/coder/websocket` v1.8.15, one zero-dependency module, the maintainer's sign-off of 2026-09-18 recorded in the spec's Summary; `go mod verify` clean. **A refusal has two texts and that is the shape the bridge forced**: the informative ones describe this Mac — a launch failure carries the child process's own paths, the pool's refusals name the memory budget in bytes — so `Ask` classifies a refusal for the caller and keeps the text for the operator's log, behind the same per-model rate limiter `handleCompletions` already used, because a stranger on a platform can drive a refusal at the rate they can send messages. **Verified by test here**: `go test -race ./...`, gofmt, `go vet`, `abcd docs lint`, `make build`; a fake gateway speaking Hello, Identify, Ready, heartbeats, Resume, MESSAGE_CREATE and INTERACTION_CREATE holds the state machine, the mention filter, the editor's throttle and cut, the conversation bound, `/model` and `/reset`, and that no message, answer or credential reaches a log line — Discord itself is never contacted from a test. The adversarial review the ADR obliges returned BLOCK on five findings, all captured first and fixed in the same change, the worst of them a crafted Hello overflowing an unbounded interval into a negative duration and panicking the whole process. **OWED to the maintainer's own Mac, never claimed as done**: a real bot with a real token connecting at all; a mention in a real channel answered; a long answer cut and continued as Discord renders it; a resume across the Mac actually sleeping and waking; the slash commands appearing in Discord's own UI; and the edit cadence holding against Discord's real rate limits rather than a fake's headers. What would show this wrong: Discord delivering empty content for mentions without the privileged intent after all, an edit cadence the real limits will not bear, or a session that will not resume across sleep."
  - The closed spec's Verification and its acceptance-criteria map state the live checks as 'checked by hand with a sleeping Mac and recorded in the shipping decision line', while the decision line it points at records them as owed — the spec was closed asserting a verification that has not happened.
    evidence: .abcd/development/specs/closed/spc-2609181018499470-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md:120 — "9. Resume across sleep — the session's resume path; checked by hand with a"
    evidence: .abcd/development/specs/closed/spc-2609181018499470-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md:131 — "real bot, a mention, a long answer, a sleep) by hand on the maintainer's Mac,"
  - No test shows /model setting a second model and the next message in that channel being answered by it: the fake server offers exactly one chat model, so the set-then-answer path is established by reading the code only.
    evidence: internal/bridge/discord/fake_test.go:92 — "ChatModels: func() []string { return []string{"mlx-community/Qwen3-8B-4bit"} },"
    evidence: internal/bridge/discord/answer_test.go:253 — "func TestTheSlashCommandsAnswerTheChannel(t *testing.T) {"
  - Nothing holds the promise that the bot shows offline once the bridge is switched off; the tests establish only that the socket closes and that nothing is answered afterwards.
    evidence: internal/bridge/discord/session_test.go:182 — "func TestSwitchingTheBridgeOffClosesTheConnection(t *testing.T) {"

Scope-condition dispositions:
- cond-2609181018499668 — survived: The bridge is Discord only, is a plain false in config.json until the operator turns it on, and New() opens nothing: a test drives the whole constructor and asserts no dial happens until Apply is called with the switch on, and every connection is outbound with no listener added anywhere.
  evidence: internal/config/config.go:592 — "DiscordBridge bool `json:"discord_bridge"`"
  evidence: internal/bridge/discord/session_test.go:38 — "func TestTheBridgeOpensNothingUntilItIsSwitchedOn(t *testing.T) {"
  evidence: docs/discord-bridge.md:8 — "**What leaves this Mac.** While the bridge is on, messages to the bot and the"
- cond-2609181018491359 — survived: The token joins the API key and the HuggingFace token in all three of the key's obligations: blanked in the snapshot, swapped back from the placeholder on save, and listed in secretSettingKeys so it is never compared against a posted value; `gropius config show` masks it through the same table.
  evidence: internal/gateway/control.go:839 — "c.DiscordToken = "********""
  evidence: internal/lifecycle/config.go:50 — "{"discord_token", func(c config.Config) string { return c.DiscordToken }, func(c *config.Config, v string) { c.DiscordToken = v }},"
  evidence: internal/gateway/control_discord_test.go:87 — "func TestARefusalDoesNotSayWhetherAGuessedDiscordTokenWasRight(t *testing.T) {"
- cond-2609181018495407 — survived: The readers' list admits the package by name — with a trailing slash, which the test's own comment marks as the wider package-level grant — and gives the reason, and a separate AST scan of every log call in the package plus a live test hold that no record and no log line carries a message or an answer.
  evidence: internal/archtest/prompt_content_test.go:33 — ""internal/bridge/discord/": "the Discord bridge, admitted by name under adr-2609181004167097 condition 4: a bridge is by its nature a reader of the message it relays, because building the request IS the relaying. It is the second such reader beside the merge. What it may do with what it reads is unchanged — no prompt reaches a log line, a record or the disk, which TestTheBridgeWritesNoMessageContent holds separately","
  evidence: internal/archtest/bridge_content_test.go:65 — "func TestTheBridgeWritesNoMessageContent(t *testing.T) {"
- cond-2609181018498110 — survived: stats.Record gains exactly one new field, a fixed Source class of two values that the recorder drops when it is not one of them, and the channel and user identifiers are parsed to uint64 for the log line alone and never reach the store.
  evidence: internal/stats/stats.go:157 — "Source Source `json:"source"`"
  evidence: internal/bridge/discord/answer.go:127 — "func numericID(id string) uint64 {"
  evidence: internal/bridge/discord/answer.go:232 — "s.bridge.log.Info("bridged a request","
- cond-2609181018490230 — narrowed: The history bound and /reset hold as assumed, and a refusal is logged by class with the detailed text kept for the operator — but one refusal class does not travel to Discord as the generic refusal: a conversation over the served window gets its own public sentence instead, a deliberate divergence taken under iss-2609190106563320.
  narrowing: The 'generic refusal reaches Discord' half holds for every refusal except the over-the-window one, which travels as its own fixed sentence, 'that conversation is too long for this model'.
  evidence: internal/bridge/discord/conversation.go:188 — "for start := 0; start < len(turns); start++ {"
  evidence: internal/bridge/discord/commands.go:118 — "conv.reset()"
  evidence: internal/gateway/ask.go:209 — "const tooLongRefusal = "that conversation is too long for this model""
- cond-2609181018494794 — survived: The sentence is inside the switch's own fieldset, an architecture test scoped to that fieldset holds each phrase of it, and the docs page carries the same words with its own test.
  evidence: internal/ui/static/index.html:589 — "and the model&rsquo;s answers pass through Discord and are kept under Discord&rsquo;s"
  evidence: internal/archtest/bridge_egress_test.go:24 — "func TestTheBridgeSwitchCarriesTheEgressSentence(t *testing.T) {"
  evidence: internal/archtest/bridge_egress_test.go:67 — "func TestTheBridgeDocsPageSaysWhatLeavesTheMac(t *testing.T) {"
- cond-2609181018495140 — survived: WhatsApp is not in the delivered tree at all and its issue is still in the open ledger, held as a seed with its prerequisites written down.
  evidence: .abcd/work/issues/open/iss-2609180959406917-whatsapp-as-a-way-of-messaging-gropius-held-the-official-clo.md:13 — "WhatsApp as a way of messaging Gropius, held: the official Cloud API needs a Meta business account, verification and a public HTTPS webhook, and the unofficial libraries breach its terms; to be filed as its own intent with those prerequisites answered, after the Discord bridge and its egress ADR."
- cond-2609181018499374 — survived: Nothing in the bridge restricts who may talk to the bot — there is no allow-list — and the docs page's step 3 tells the operator to turn Public Bot off and says outright that where the bot lives is what decides who can use her models.
  evidence: docs/discord-bridge.md:23 — "3. Turn **Public Bot** *off*. While it is on, anybody who finds your"
  evidence: internal/ui/static/index.html:589 — "and the model&rsquo;s answers pass through Discord and are kept under Discord&rsquo;s"
- cond-2609181018490128 — narrowed: Each channel is its own in-memory conversation and nothing of a message is written to disk — but the conversations are built fresh inside runSession, so they are gone at every dropped connection and resume, not only when the bridge stops, which is the narrower of the two promises and is the ordinary path after a Wi-Fi blip or the Mac sleeping.
  narrowing: A channel's history and its /model choice survive only for as long as one gateway connection does: they are discarded on every reconnect or resume, not merely when the bridge is switched off.
  evidence: internal/bridge/discord/session.go:176 — "convos: newConversations(),"
  evidence: internal/bridge/discord/conversation.go:71 — "func (c *conversations) get(channelID string) *conversation {"
- cond-2609181018495938 — narrowed: The models offered are the server's ready chat models under the same rule /v1/models publishes, read live, and /model sets one per channel — but the server has no default-model setting for a channel to start from, so the bridge starts a channel on the alphabetically first chat model, a bridge-local choice rather than a server default.
  narrowing: 'Starting from the server's default' holds only as 'starting from the first of the server's chat models sorted by repo id'; Gropius has no configured default model for the bridge to inherit.
  evidence: cmd/gropius/bridge.go:49 — "sort.Strings(out)"
  evidence: internal/bridge/discord/commands.go:152 — "conv.setModel(m)"
- cond-2609181018495500 — survived: Only the two unprivileged intents are asked for, a guild message that Discord's mentions array does not name the bot in is dropped on the read loop before its content is used, and the docs page tells the operator to leave all three privileged intents off.
  evidence: internal/bridge/discord/session.go:48 — "const intents = 1<<9 | 1<<12"
  evidence: internal/bridge/discord/answer.go:74 — "if !direct && !mentions(msg, botID) {"
  evidence: internal/bridge/discord/answer_test.go:48 — "func TestAChannelMessageIsAnsweredOnlyWhenItMentionsTheBot(t *testing.T) {"
- cond-2609181018497578 — survived: Exactly one new module was added, github.com/coder/websocket v1.8.15, as a direct requirement with no transitive additions, and the spec's Summary records the maintainer's hand-rolled-on-a-small-library sign-off of 2026-09-18.
  evidence: go.mod:8 — "github.com/coder/websocket v1.8.15"
  evidence: .abcd/development/specs/closed/spc-2609181018499470-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md:20 — "zero-dependency module, `github.com/coder/websocket` (the maintainer's"
- cond-2609181018498135 — survived: Nothing in the change touches the bind plan or opens a listener — the bridge dials out over one WebSocket with the stdlib's own verification, held by its own architecture test — and the switch, token and state are on all three surfaces: the panel fieldset, config.json, and the Go composition root.
  evidence: internal/archtest/bridge_content_test.go:266 — "func TestTheBridgeNeverWeakensTheOutboundHandshake(t *testing.T) {"
  evidence: internal/config/config.go:592 — "DiscordBridge bool `json:"discord_bridge"`"
  evidence: cmd/gropius/bridge.go:24 — "func newDiscordBridge(a *app.App, g *gateway.Gateway, log *slog.Logger) app.Bridge {"
  evidence: internal/ui/static/index.html:589 — "and the model&rsquo;s answers pass through Discord and are kept under Discord&rsquo;s"
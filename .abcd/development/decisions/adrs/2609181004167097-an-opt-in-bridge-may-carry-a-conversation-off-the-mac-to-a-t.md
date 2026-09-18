---
id: adr-2609181004167097
slug: an-opt-in-bridge-may-carry-a-conversation-off-the-mac-to-a-t
status: accepted
date: 2026-09-18
supersedes: null
superseded_by: null
related_intents: [itd-2609180959397172]
related_rfcs: []
related_adrs: [adr-2609061503319212, adr-2609061610102325, adr-2609111126115848, adr-2609091123526871]
---

# ADR-2609181004167097: An opt-in bridge may carry a conversation off the Mac to a third-party platform

## Context

Until now nothing a person types into Gropius, and nothing a model answers,
leaves the local network: the server binds the LAN, the bind-mode intent
narrowed it further on purpose, adr-2609061503319212 rules out telemetry to
the project, a vendor or any third party, and the chat client sends a
prompt only to a server the person picked. The maintainer asked, on
2026-09-18, for a way to reach Gropius from messaging apps people already
use — Discord first, WhatsApp held. A bridge to such a platform is a
different kind of thing from everything before it: the server keeps an
outbound connection open to a company's servers, the prompt Bob types and
the answer a model writes travel through that company and are retained
under its terms, and a bot token — a bearer credential for the bridge —
becomes a second secret beside the API key. That is a trust-boundary
change, and it is decided here before any code.

This refines adr-2609061503319212: that decision bars anything leaving the
Mac *about* the operator — usage, hardware, errors, models, configuration —
to the project, a vendor or a third party. A bridge carries only what a
person deliberately typed into the platform they chose, and the model's
answer to it; it reports nothing to the project or a vendor, and the model's
name reaches the platform only because the person asked for it there.

The maintainer's decisions at the interview: anyone who can reach the bot
may talk to it, with each channel or direct message its own conversation;
the model answering a channel is chosen there with a slash command, from
the server's chat models.

## Decision

We will allow a bridge to carry a conversation off the Mac to a
third-party platform, under these conditions, each of which a test or a
review holds:

1. **Opt-in, per bridge, and off by default.** Nothing leaves the Mac until
   the operator has pasted a token and switched that bridge on in Settings;
   the panel, `config.json` and Go all carry the switch (the three-surfaces
   rule). Switching it off closes the connection.
2. **Outbound only.** A bridge connects out from the Mac to the platform;
   it opens no inbound port and changes nothing about the bind mode. The
   LAN posture of the server is untouched by a bridge being on.
3. **The token is a secret.** It is stored in `config.json` beside the API
   key, redacted wherever the API key is redacted (snapshots, logs, the
   panel, `gropius config show`), never written to the statistics store,
   and rotated by pasting a new one.
4. **The log records the fact of a bridged request, never its content.** A
   bridge is, by its nature, a reader of the message it relays and of the
   answer it posts — the second such reader beside the merge that
   adr-2609061610102325 grants — and that reading is admitted by name in
   the test that holds the readers' list. What is recorded is another
   matter: the statistics store gains one fixed class, the source, and
   nothing a platform supplied; the log line carries the bridge's name, the
   platform's channel and user identifiers as opaque numbers, the model, the
   sizes and the timing — no prompt, no answer, on the same rule every other
   record follows.
5. **What leaves is said in the product's own words.** The Settings pane
   that takes the token states, beside the switch, that messages to the bot
   and the model's answers pass through the platform and are kept under its
   terms; the docs page says the same. The switch cannot be rendered without
   that sentence in the same pane; held by a test.
6. **A platform that needs an inbound public endpoint, a business account
   or a breach of its terms is not bridged.** WhatsApp's official API needs
   the first two; its unofficial libraries are the third. Such a platform
   waits for its own decision; held by review at each bridge's planning and
   recorded in that bridge's intent.

## Alternatives Considered

- **No bridges: the LAN-only posture is absolute.** Rejected by the
  maintainer's ask; the posture is kept for everything that is not an
  explicit, labelled opt-in, which is what conditions 1, 2 and 5 preserve.
- **A bridge as a separate program that talks to the OpenAI endpoint from
  outside.** Fair, and it keeps the server's code untouched, but it moves
  the token and the switch outside the three surfaces the product promises
  to keep in sync, and it makes the log blind to what is bridged. Rejected.
- **End-to-end encryption to the platform.** Not available to bots on
  Discord; a promise the product cannot make. Not pursued.
- **Allow-listing who may talk to the bot by default.** Considered at the
  interview; the maintainer chose "anyone who can reach the bot", with the
  operator controlling reach by where the bot is invited. An allow-list can
  be added as a setting without revisiting this decision.

## Consequences

- Easier: a phone, and any person already on the platform, can reach
  Alice's models with nothing installed.
- Harder: a new secret to handle on every surface; a long-lived outbound
  connection to keep healthy (reconnect, rate limits) inside a server that
  had none; a docs page and a Settings sentence that must say plainly what
  leaves the Mac.
- Obligations: an adversarial security review of every bridge before it
  lands; a test that the token is redacted everywhere the API key is; a
  test that no record or log line carries a bridged message's content;
  the changelog names a bridge as `impact: additive` with the egress
  stated; a maintainer's sign-off on any dependency a bridge needs, before
  it is added.

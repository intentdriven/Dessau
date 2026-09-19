---
schema_version: 1
id: "iss-2609190312188937"
slug: "the-discord-bridge-s-conversation-bound-is-stated-in-runes-a"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the conversation-store change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/conversation.go"
---

The Discord bridge's conversation bound is stated in runes and nobody has stated what it costs. internal/bridge/discord/conversation.go holds maxChannels 256 conversations of maxTurns 64 turns, each turn bounded at maxMessageRunes 8000 runes — and a rune is up to four UTF-8 bytes, so the store's ceiling is 512 MiB of text a stranger on Discord can drive, measured live rather than estimated. The turn is appended before the completion is asked for, so a refused request stores the text too: one person with 256 channels and a mention in each reaches the ceiling without a model ever answering. Until iss-2609190241509478 every dropped socket emptied it, which hid the figure; now it stands for the whole time the bridge is on. On a product whose model-pool budget is derived from this Mac's memory, half a gigabyte of held heap is not neutral. Either the arithmetic belongs in those comments as an accepted cost, or the turn should be bounded in bytes rather than runes.

## Deferral 2026-09-19

Left for the maintainer rather than changed in the lane that surfaced it. The
arithmetic is now stated beside the constants in
`internal/bridge/discord/conversation.go` and recorded in the decisions ledger,
so nobody is reading the bounds without the figure. What it waits on is a
product decision: whether a bridge may hold 256 channels at all, and whether a
turn should be bounded in bytes rather than runes — which would cut a message
in a script where a rune is three bytes to a third of what a Latin one keeps.


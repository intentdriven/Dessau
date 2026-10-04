---
schema_version: 1
id: "iss-2610041945030524"
slug: "the-discord-bridge-ignores-edited-messages-and-the-docs-do"
severity: "nitpick"
category: "documentation"
source: "manual-test"
found_during: "live Discord checks (iss-2609190242198542), 2026-10-04, v0.9.3"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/discord-bridge.md"
remedy: "State in docs/discord-bridge.md that the bot reads a message only when it is sent, never an edit, so a mention or question added by editing is not answered and must be sent again as a new message."
---

The Discord bridge ignores edited messages, and the docs do not say so. A message that gains its mention of the bot by an edit, or whose question is changed after sending, is never answered: the bridge reads a message only when it is created. That is a reasonable design, but docs/discord-bridge.md is silent on it, so to a user it looks like the bot has stopped working. Seen 2026-10-04 with an edited '@Dessau tell me more about epoché'.

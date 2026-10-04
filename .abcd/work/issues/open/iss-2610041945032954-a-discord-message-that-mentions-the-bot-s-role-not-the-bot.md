---
schema_version: 1
id: "iss-2610041945032954"
slug: "a-discord-message-that-mentions-the-bot-s-role-not-the-bot"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "live Discord checks (iss-2609190242198542), 2026-10-04, v0.9.3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/answer.go"
remedy: "Find out whether Discord delivers the content of a message that mentions the bot's managed role to a bot without the message-content intent; if it does, answer it as a mention of the bot; if it does not, say in docs/discord-bridge.md that the bot must be picked from the popup, not the role."
---

A Discord message that mentions the bot's role, not the bot, is silently ignored. When the bot joins a server Discord creates a managed role with the same name ('Dessau'). Typing @Dessau without picking from the popup can attach the mention to that role. The bridge answers only when msg.Mentions holds the bot user (internal/bridge/discord/answer.go), so the message gets no answer and no hint why. Confirmed 2026-10-04: three such messages went unanswered, and clicking one showed Discord's role pop-up ('Dessau — 1').

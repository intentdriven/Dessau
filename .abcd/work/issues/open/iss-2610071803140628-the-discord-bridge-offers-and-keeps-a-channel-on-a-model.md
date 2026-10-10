---
schema_version: 1
id: "iss-2610071803140628"
slug: "the-discord-bridge-offers-and-keeps-a-channel-on-a-model"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "Discord bridge use, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord"
remedy: "Leave a model with a recorded load failure out of /model's list (or list it marked as not loading) and refuse naming it there; when a channel's model has a recorded load failure, reply that it does not load on this server and to choose another with the /model command, without the log's detail; recognise a mention whose text is only '/model' and point to the slash command; tests for each in internal/bridge/discord."
---

The Discord bridge offers, and keeps a channel on, a model that has a recorded load failure, and its refusal gives no way out. On 2026-10-07 a channel set to eins78/Kolibri-1-mlx-mixed-4-8-bit (model type kolibri1, which the pinned mlx-lm 0.32.0 does not support, iss-2610070546301424) answered every mention with the generic 'cannot serve this model right now': the first request tried the load and failed in 1.5 s, and the next ones were refused at once (class not_ready) because the load failure is recorded. /model still listed the model, and nothing in the reply suggested choosing another; the user then typed '@Dessau /model' as text, which the bridge passes to the model rather than to the slash command. Nothing in internal/bridge/discord reads a model's load failure.

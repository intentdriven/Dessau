---
schema_version: 1
id: "iss-2609190242403123"
slug: "docs-discord-bridge-md-states-a-fact-about-discord-that-nobo"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/discord-bridge.md"
---

docs/discord-bridge.md states a fact about Discord that nobody here has checked and that reads the wrong way round. Step 4 of 'Make the bot in Discord' tells the reader to leave the three Privileged Gateway Intents off and says 'Discord refuses the connection if it is configured to require an intent the bridge does not request'. Discord's documented 4014 close is the other direction: it refuses a connection that REQUESTS a privileged intent the application is not approved for. Turning an intent on in the portal that the bridge never asks for is not documented to refuse anything. The bridge's own code is right — internal/bridge/discord/session.go identifies with the two unprivileged intents and treats 4014 as fatal with the portal named in the reason — so this is the docs sentence alone. It should be checked against a real bot with the message-content intent switched on in the portal, and then either corrected or evidenced.

## Triage 2026-09-19

This one waits on the maintainer's hand: settling it means a real application
in Discord's portal with the message-content intent switched on and a real
connection attempted, which no test here may make. The decision it needs is
whether to check it and keep the sentence, or to drop the causal claim and keep
only the instruction to leave the privileged intents off, which is sound advice
either way.

---
schema_version: 1
id: "iss-2609190312313645"
slug: "the-panel-tells-the-operator-a-token-that-has-never-connecte"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial security review of the last-connected-moment change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/bridge.go"
resolution: "The last-connected moment is now cleared whenever the token changes — before Apply's branches, so it covers the switch on under a new token, a token changed under a running bridge, and a token taken away — and kept across a drop as before. The panel therefore credits a freshly pasted token with no moment until it connects. Watched to fail: 'a freshly pasted token is connecting and reports last connecting at <the old token's moment>, want no moment at all'."
impact: fix
---

The panel tells the operator a token that has never connected last connected at a moment. internal/bridge/discord/bridge.go clears the last-connected moment only when the switch goes off, so pasting a different bot token — the restart path through Apply — carries the previous credential's moment into the new one's states. The card then reads 'Stopped: Discord refused the bot token — paste a fresh one in Settings. Last connected at <moment>.', which is the token the operator has just pasted being credited with a session it never had, on exactly the diagnostic path the moment was added for (iss-2609190242334438). It also contradicts internal/app/bridge.go's own sentence that the moment is absent only when the bridge has not connected since the switch was thrown. The moment is a fact about the credential that connected, so a different token should start with none.

## Grounds

- pursued: we expect the moment to be a fact about the credential that connected because the operator reads it to answer 'did the token I just pasted work?'; wrong if an operator turns out to want the bridge's whole history of connecting rather than the current credential's.

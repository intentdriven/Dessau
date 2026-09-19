---
schema_version: 1
id: "iss-2609190242334438"
slug: "the-panel-never-shows-when-the-discord-bridge-last-connected"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
---

The panel never shows when the Discord bridge last connected except while it is still connected. The eleventh acceptance criterion of itd-2609180959397172 says that when the gateway session drops the bridge resumes by itself and the panel shows when it last connected, but internal/app/bridge.go declares BridgeState.Since absent unless the bridge is connected, and internal/ui/static/app.js renders 'Connected since <moment>' only in the connected case — the connecting case says 'Connecting to Discord...' and the stopped case says only the reason. So on exactly the path the criterion describes, a dropped session reconnecting, the last-connected moment disappears from the panel instead of being shown. Keeping Since across a drop, and rendering it under connecting and stopped as 'last connected <moment>', is what the criterion asks for.

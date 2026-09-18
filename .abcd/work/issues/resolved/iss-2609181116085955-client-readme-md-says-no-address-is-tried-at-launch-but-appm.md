---
schema_version: 1
id: "iss-2609181116085955"
slug: "client-readme-md-says-no-address-is-tried-at-launch-but-appm"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/README.md"
resolution: "client/README.md now says what happens at launch: the browse for servers runs only while the picker is open, and a client whose last answerer was a server reconnects to its stored address at launch — so the local-network question can come then."
impact: fix
---

client/README.md says no address is tried at launch, but AppModel.init still calls connect() against the stored address whenever the last choice was a server.

## Grounds

- pursued: the README describes the launch the code performs. Wrong if AppModel.init stops reconnecting.

---
schema_version: 1
id: "iss-2609180608280915"
slug: "appmodel-ask-returns-an-empty-reply-as-success-when-nothing"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "ask connects first when a server is chosen, throws the client's own reason when nothing can answer, and the model connects at launch when a server answerer is stored."
impact: fix
---

AppModel.ask returns an empty reply as success when nothing can answer: send returns silently on cannotSend, ask awaits the previous reply task and returns empty, leaving an empty chat; and nothing calls connect at launch any more, so a server answerer is 'not connected' until Cmd-R. ask should throw the client's own reason and connect first.

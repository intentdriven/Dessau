---
schema_version: 1
id: "iss-2610030822065191"
slug: "ask-panics-on-a-null-body"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fix for iss-2610030752128514"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/ask.go"
remedy: "Refuse a body that does not decode to a JSON object with an AskError before touching the map, with a test for null, array and scalar bodies."
---

Ask in internal/gateway/ask.go panics on a JSON null body: the body decodes to a nil map and the later write of payload["model"] assigns into it. The Discord bridge, Ask's only caller, never sends such a body, so nothing reaches it today; a future caller or a bridge change would crash the request goroutine instead of getting an error. Found 2026-10-03 while fixing iss-2610030752128514.

---
schema_version: 1
id: "iss-2609201007355356"
slug: "the-bridge-s-statistics-record-must-stay-free-of-platform-id"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "2026-09-20 interview, from iss-2609190242131524"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/stats/stats.go"
---

The bridge's statistics record must stay free of platform identifiers and the docs must say where they go: the shipped Discord intent's eighth criterion says the statistics record carries the channel and user identifiers, its scope condition cond-2609181018498110 says they go to the log line only, and the delivered code follows the condition. The maintainer decided on 2026-09-20 that the delivered behaviour is the intended one and the shipped intent is not edited; what is owed is an architecture test that holds internal/stats records free of any platform-supplied identifier, and one sentence on docs/discord-bridge.md saying the channel and user identifiers appear on the log line only.

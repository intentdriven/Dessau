---
schema_version: 1
id: "iss-2610032212269524"
slug: "transcript-mark-leaves-debug-armed"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "docs-currency review of the no-transcript docs (iss-2609211218478273)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Disarm a model's debug logging when a save marks it as keeping no transcript, with a test."
---

Ticking a model's transcript box does not take it off the debug-logging list, though docs/transcript.md, the panel's Transcript hint and the 2026-09-20 decision say it does: the settings save never disarms, the mark is consulted only when debug logging is armed, so a model armed first and marked afterwards still launches at debug and writes every prompt and answer to its log.

---
schema_version: 1
id: "iss-2609181116073091"
slug: "the-composer-s-text-file-drop-is-delivered-inline-in-the-com"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The composer's text-file drop is delivered inline in the composer view rather than as the specced tested unit on AppModel, and no automated test covers it at all.

## Triage 2026-09-18

Left open for the maintainer: moving the drop onto AppModel is the spec's shape, but the client has no test target, so the move would land untested either way — it wants deciding alongside how the client gets tested at all (iss-2609181116225273).

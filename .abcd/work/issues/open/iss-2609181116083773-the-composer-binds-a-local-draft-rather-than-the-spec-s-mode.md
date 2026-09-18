---
schema_version: 1
id: "iss-2609181116083773"
slug: "the-composer-binds-a-local-draft-rather-than-the-spec-s-mode"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The composer binds a local draft rather than the spec's model input binding, and the architecture test pins the delivered spelling.

## Triage 2026-09-18

Left open for the maintainer: the local draft and the spec's model binding differ in where a half-typed prompt survives a view's recreation; which the client wants is a design call, and the test pins what ships either way.

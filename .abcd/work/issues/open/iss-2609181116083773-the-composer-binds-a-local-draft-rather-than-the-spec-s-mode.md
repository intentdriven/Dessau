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

## Deferral 2026-09-19

Waits on the maintainer: where a half-typed prompt survives a view's recreation is a design call; the test pins what ships.

## Decision 2026-09-20

Maintainer's decision at interview: keep the delivered behaviour; the shipped promise is corrected by a decision line in `.abcd/work/DECISIONS.md`. Resolved on that grounds.

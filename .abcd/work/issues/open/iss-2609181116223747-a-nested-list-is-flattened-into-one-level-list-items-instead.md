---
schema_version: 1
id: "iss-2609181116223747"
slug: "a-nested-list-is-flattened-into-one-level-list-items-instead"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
---

A nested list is flattened into one-level list items instead of falling back to the model's own text; only tables fall back, against the promise that an undrawable construct is shown as the model wrote it.

## Triage 2026-09-18

Left open for the maintainer: a flattened nested list may well read better than the raw marks, so this is a choice between the mechanism's promise and the delivered rendering. client/README.md now says plainly what is drawn (iss-2609181116224898), so nothing is claimed that is not true while it waits.

## Deferral 2026-09-19

Waits on the maintainer: a flattened nested list versus the model's raw marks is a choice between the mechanism's promise and the delivered rendering; client/README.md says what is drawn.

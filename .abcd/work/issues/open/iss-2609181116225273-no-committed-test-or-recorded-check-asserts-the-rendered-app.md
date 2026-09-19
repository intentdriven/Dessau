---
schema_version: 1
id: "iss-2609181116225273"
slug: "no-committed-test-or-recorded-check-asserts-the-rendered-app"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
---

No committed test or recorded check asserts the rendered appearance of the five markdown constructs: the record evidences only the harness's block split, and nothing holds italic, bold, monospaced, a clickable link or a kept line break as drawn.

## Triage 2026-09-18

Left open for the maintainer: asserting a rendered appearance needs a UI test target the client does not have; standing one up is a decision about how this client is tested, and every other rendering finding here waits on it.

## Deferral 2026-09-19

Waits on the maintainer: asserting rendered appearance needs a UI test target the client does not have; standing one up is a decision about how this client is tested, and the other rendering findings wait on it.

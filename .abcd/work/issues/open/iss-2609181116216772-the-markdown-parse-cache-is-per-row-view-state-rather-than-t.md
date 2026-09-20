---
schema_version: 1
id: "iss-2609181116216772"
slug: "the-markdown-parse-cache-is-per-row-view-state-rather-than-t"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/DessauChat/Markdown.swift"
---

The markdown parse cache is per-row view state rather than the spec's cache keyed by message id and text length, so a row recreated on scroll re-parses a finished reply.

## Triage 2026-09-18

Left open for the maintainer: a cache keyed by message id and text length has to live above the rows and be bounded and evicted; where it lives and how big it gets is a design call, not a move.

## Deferral 2026-09-19

Waits on the maintainer: a bounded, evicted parse cache above the rows is a design call (where it lives, how big).

## Decision 2026-09-20

Decided at the second interview of 2026-09-20; the lane builds it with a test. See the 2026-09-20 line in `.abcd/work/DECISIONS.md` that names this id.

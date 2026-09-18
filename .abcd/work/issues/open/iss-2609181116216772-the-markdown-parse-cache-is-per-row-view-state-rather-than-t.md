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
found_at: "client/GropiusChat/Markdown.swift"
---

The markdown parse cache is per-row view state rather than the spec's cache keyed by message id and text length, so a row recreated on scroll re-parses a finished reply.

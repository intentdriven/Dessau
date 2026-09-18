---
schema_version: 1
id: "iss-2609181213191220"
slug: "the-sidebar-search-matches-the-whole-trimmed-query-as-one-li"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The sidebar search matches the whole trimmed query as one literal substring, so a multi-word query matches only contiguous text and never words found separately, against the press release's promise that the list filters to conversations containing the words.

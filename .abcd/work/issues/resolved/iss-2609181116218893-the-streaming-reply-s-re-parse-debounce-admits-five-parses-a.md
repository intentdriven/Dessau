---
schema_version: 1
id: "iss-2609181116218893"
slug: "the-streaming-reply-s-re-parse-debounce-admits-five-parses-a"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
resolution: "Both re-parse debounces — the reply's and the thoughts' — wait a quarter of a second, which is the spec's four parses a second. TestChatClientReplyReparseStaysWithinItsBudget holds both."
impact: fix
---

The streaming reply's re-parse debounce admits five parses a second where the spec said at most four.

## Grounds

- pursued: the streaming reply re-parses at most four times a second. Wrong if the stream visibly stutters at the wider spacing.

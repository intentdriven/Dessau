---
schema_version: 1
id: "iss-2609181213194439"
slug: "the-sidebar-card-s-date-is-rendered-as-an-explicit-day-month"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The sidebar card's date is rendered as an explicit day, month and year field list rather than the system's short date style the spec named, and an exchange is counted as the assistant replies alone rather than the person's message and its reply.

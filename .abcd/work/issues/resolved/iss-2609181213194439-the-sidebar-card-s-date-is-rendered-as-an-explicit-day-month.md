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
resolution: "The card now draws its date with conversation.createdAt.formatted(date: .numeric, time: .omitted) — the system's own short date style rather than a day/month/year field list of the client's own — and counts exchanges as pairs through a pure exchangeCount(fromPerson:) helper in client/GropiusChat/CardSummary.swift, so a message still waiting for its reply no longer reads as an exchange and a reply that arrived in two parts no longer reads as two."
impact: fix
---

The sidebar card's date is rendered as an explicit day, month and year field list rather than the system's short date style the spec named, and an exchange is counted as the assistant replies alone rather than the person's message and its reply.

## Grounds

- pursued: we expect the card to match what the intent promised — the system's short date and an exchange as a person's message and its reply — because the intent's scope condition names both and the audit recorded both as diverged; wrong if the short numeric date reads as too terse in the sidebar next to Messages, or if a conversation's turns can interleave in a shape where counting pairs reads lower than a person would count.

---
schema_version: 1
id: "iss-2609181213195806"
slug: "the-sidebar-card-s-summary-singularises-at-a-count-of-one-re"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
wontfix_reason: "The card reads '1 exchange' and '1 word' because that is correct English; the spec's fixed plural was a wording of the promise, not a promise to the person reading the card."
---

The sidebar card's summary singularises at a count of one, reading 1 exchange and 1 word, where the spec's wording was a fixed plural form.

## Grounds

- declined: The card reads '1 exchange' and '1 word' because that is correct English; the spec's fixed plural was a wording of the promise, not a promise to the person reading the card.

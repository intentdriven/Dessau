---
schema_version: 1
id: "iss-2609181116224898"
slug: "the-client-readme-lists-only-what-is-rendered-and-names-noth"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/README.md"
resolution: "client/README.md names the two constructs that are not drawn: a table is shown as the model wrote it, and a nested list is drawn as one level of items."
impact: fix
---

The client README lists only what is rendered and names nothing as not rendered, against the mechanism's promise that an undrawable construct - tables, nested lists - is named in the docs as not rendered.

## Grounds

- pursued: the docs name what is not rendered, so nothing a reply cannot draw is a surprise. Wrong if the renderer gains either construct and the sentence is left behind.

---
schema_version: 1
id: "iss-2609181213199302"
slug: "the-thoughts-row-was-expected-to-render-with-no-new-code-by"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "MessageRow now parses both panes through the one scheduleParse/parseNow pass over a single blocks store keyed by the text it parsed; the scheduleReasoningParse twin and the reasoningTask/lastReasoningParse state are gone, with reasoningBlocks kept as a computed read of that store."
impact: internal
---

The Thoughts row was expected to render with no new code by reusing the reply block parser; it landed with roughly thirty lines including three new state properties and a duplicated parse-and-throttle twin rather than reuse of the existing scheduler.

## Grounds

- pursued: we expect one scheduler and one throttle to serve the reply and the Thoughts row because the thoughts are text of the same kind; wrong if the two panes ever need different cadences

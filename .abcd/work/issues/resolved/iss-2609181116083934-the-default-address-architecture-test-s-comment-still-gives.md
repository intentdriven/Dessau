---
schema_version: 1
id: "iss-2609181116083934"
slug: "the-default-address-architecture-test-s-comment-still-gives"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest"
resolution: "The default-address test's comment gives the reason that is true now: the client answers on the Mac out of the box rather than waiting on a server, and the stored default is what a launch reconnects to and what the picker asks for its models."
impact: internal
---

The default-address architecture test's comment still gives the retired reason the composer is disabled until a server answers; the intent's Open Question asked for the comment to say what is now true.

## Grounds

- pursued: the comment says why the assertion exists today. Wrong if the first-run answerer changes again.

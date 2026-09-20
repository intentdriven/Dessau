---
schema_version: 1
id: "iss-2609190312326963"
slug: "a-discord-bridge-that-discord-stopped-cannot-be-started-agai"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial security review of the conversation-store change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/bridge.go"
---

A Discord bridge that Discord stopped cannot be started again by saving the same settings. Apply in internal/bridge/discord/bridge.go returns early when the switch, the token and the session handle are unchanged — 'already running under exactly these settings' — but the handle is a closed channel rather than nil after the run loop exits on a fatal close code, so the bridge is stopped and the early return still fires. For a refused token that is the right outcome, since the same token will be refused again; for a READY over the read limit or a 4010-4013 the operator has nothing to change and must toggle the switch off and on to try at all. Pre-existing, and found while reviewing the conversation store's lifetime rather than by a report.

## Deferral 2026-09-19

Left for the maintainer: the fix it wants is a decision about what a saved
setting means for a bridge Discord has stopped, not a line of code, and it is
older than the lane that found it.

What the lane changed about its weight, recorded so nobody under-rates it:
until iss-2609190312064731 was fixed, an operator who could not restart at
least had a bridge holding nothing of anybody's conversation. The store is now
dropped on that path too, so the no-op restart is the last symptom left — and
the only routes back from a fatal stop are changing the token, switching the
bridge off and on, or restarting the app. That is a correctness wedge rather
than a privacy one, and it is what a reader of the title alone would miss.


## Decision 2026-09-20

Decided at the second interview of 2026-09-20; the lane builds it with a test. See the 2026-09-20 line in `.abcd/work/DECISIONS.md` that names this id.

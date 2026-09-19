---
schema_version: 1
id: "iss-2609181116072455"
slug: "the-built-in-model-s-context-trim-budgets-the-prior-turns-on"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Backends.swift"
deferred_after: "v0.7.1"
deferral_reason: "Left open at the 0.7.1 cut: an observation from the 2026-09-18 fidelity audits that needs the maintainer's decision; recorded here rather than stepped over."
resolution: "The trim now budgets the new prompt's own estimated tokens together with the instructions, the reply's reserve and the prior turns, which are dropped oldest-first into what is left; a prompt that does not fit on its own is not sent at all and the transcript carries the client's own one-sentence explanation instead of a reply. The retry stays for an estimate that was wrong. The arithmetic lives in client/GropiusChat/ContextBudget.swift, which imports nothing, and is run by client/tests/context-budget.sh from an architecture test."
impact: fix
---

The built-in model's context trim budgets the prior turns only and not the new prompt's own tokens, so a prompt that overflows the window on its own is left to the single retry to absorb.

## Triage 2026-09-18

Left open for the maintainer: budgeting the new prompt's own tokens changes what the window holds and when a turn is refused outright rather than retried — a policy call on the built-in model, not a mechanical correction.

## Grounds

- pursued: we expect budgeting the prompt with the prior turns to prevent the overflow rather than catch it, and an outright refusal to be the honest answer for a message no amount of dropped history makes room for, because the retry could only ever raise the same framework error later; wrong if people meet the refusal on messages they consider ordinary, which would mean the reply reserve is too greedy rather than the arithmetic wrong

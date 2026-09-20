---
schema_version: 1
id: "iss-2609161712560916"
slug: "probe-check-sh-s-preflight-does-not-check-that-the-model-can"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual context-probe check 2026-09-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/evidence/2026-09-06-model-bench/"
---

probe-check.sh's preflight does not check that the model can fit the budget before it quits the menu-bar Dessau for hours. The 2026-09-16 run started against a snapshot whose warnings already said the budget was smaller than the smallest model, then polled a queue that could never start until the Mac froze. The preflight should read /api/state's warnings and the model's charge against the budget and refuse to start a model that cannot fit, naming the served window and batched-requests count that would.

## Deferral 2026-09-19

`probe-check.sh` is not in this repository. It has never been committed: no
commit on any branch has ever added a path matching `probe-check`, nothing
untracked or stashed carries that name, and the directory this record names as
`found_at` holds the 2026-09-06 campaign's Python probes
(`context_probe.py`, `sampling_probe.py`, `writer_reviewer.py`) and their JSON,
with no shell script and no README. The script is the maintainer's own, on
their Mac, which is consistent with `found_during: manual context-probe check`.

Nothing can be added to it from here, so this waits on the maintainer. The
decision it needs: either commit the script under
`.abcd/development/research/evidence/`, after which the preflight below is an
ordinary lane item, or close this record and keep the check as a hand tool.

The preflight itself is not blocked by anything missing from the product —
`/api/state` already publishes every term it needs, so whoever writes it does
not have to derive this again:

- the budget in force is `machine.budget`, and what is already charged against
  it is `machine.resident_bytes` (which already includes `machine.exiting_bytes`);
  every byte count on the snapshot is an int64 of bytes, never gigabytes.
- the too-small-budget warning is an element of the top-level `warnings` array,
  worded by `App.tooSmallWarning`: "The memory budget (…) is smaller than the
  smallest model on this Mac (… needs about …), so every request is refused
  until it is raised." The sizes are rendered for people, so a preflight matches
  the fixed prefix rather than the whole string.
- a model's charge is not published for a model that is not resident
  (`resident[].charge_bytes` covers only those that are), so it is computed the
  way `capability.LoadCostOf` computes it: `bytes + bytes/5`, plus
  `kv_charge_per_token × served window × sequences` when the per-token charge is
  positive. The terms are `models[].bytes` (or `size_bytes` while downloading),
  `models[].kv_charge_per_token`, `models[].context_length`,
  `config.models[<repo_id>].served_context` (absent means the declared window),
  and `config.decode_concurrency`.
- the window and batched-requests count that would fit follow from the same
  arithmetic: with `flat = bytes + bytes/5` and
  `headroom = machine.budget − machine.resident_bytes`, any pair satisfying
  `window × sequences ≤ (headroom − flat) ÷ kv_charge_per_token` fits, with the
  window capped at `models[].context_length`.
- `idle_jobs.held_by` carries `no_room` with `idle_jobs.due` naming the model
  when the idle loop has a due model that does not fit (iss-2609161712555136).
  It is `omitempty`, so an absent field means nothing is held, and it reports
  one model at a time — it is a confirmation, not a substitute for the
  arithmetic above.

One caveat that bites any fit check written against the snapshot, including
this one: the concurrency on `/api/state` is the saved figure, not the one the
pool is running with, while the budget beside it is the one in force. That is
captured separately as iss-2609190021445846.

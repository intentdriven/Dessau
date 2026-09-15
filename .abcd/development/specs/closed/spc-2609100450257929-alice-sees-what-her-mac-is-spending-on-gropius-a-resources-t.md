---
id: spc-2609100450257929
slug: alice-sees-what-her-mac-is-spending-on-gropius-a-resources-t
intent: itd-2609091903463596
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The resources roll-up at the head of the Models tab

## Summary

One block at the head of the Models tab that adds up what the panel already
knows: how many models are downloaded and how many loaded, how much disk they
take and how much the volume has left, the memory budget against what is
resident and what is still exiting, and on each card the window the model
declares and the window it is set to serve. No new measurement: every figure
but the volume's free space is already on the state snapshot, and free space
has one reader, `capability.Assess`, which the snapshot's machine block
already carries. Placement was decided at the interview: a block, not a
seventh tab.

## Scope

In scope: the roll-up block (markup in `internal/ui/static/index.html`, a pure
`resourcesSummary(state)` in `app.js` the tests hold, drawn on every state
event); the declared and served windows on each model card, resolved the
fold-aware way the panel already resolves a served window (`servedContext`);
the one addition the snapshot needs, if any (see Approach); a section in
`docs/models-list.md`'s companion page for the panel, or in
`docs/memory-budget.md`, saying what the block shows; the changelog entry.

Out of scope: any figure that needs a directory walk on the render path (the
per-model sizes are recorded at download and rescan and are read from the
registry entry); live usage figures, which are the measurement intent's; a
new endpoint — the block reads the snapshot the panel already polls.

## Approach

**The figures and where each comes from, all already on the snapshot:**

| Figure | Source on the snapshot |
| --- | --- |
| Models downloaded | `models` with `state == "ready"` |
| Models loaded | `resident` entries in state `loaded` (loading counted separately, as "loading") |
| Disk the models take | sum of `models[].bytes` over ready models — recorded sizes, no walk |
| Free space on the models volume | `machine.free_disk`, from `capability.Assess`'s one reader |
| Memory budget | `machine.budget` |
| Resident, charged | `machine.resident_bytes`, which already includes the exiting servers |
| Still exiting | `machine.exiting_bytes`, named as such beside the resident figure |
| Stuck servers | `machine.stuck_servers`, shown only when non-zero |
| Per card: declared window | `models[].context_length` |
| Per card: served window | `servedContext(config, repo_id, context_length)`, the panel's existing fold-aware helper |

So the change is entirely in the panel, plus its tests. If `machine.free_disk`
turns out not to ride the snapshot on every poll (it is read by `Assess` at
start today), the one server-side addition is to refresh it on the snapshot at
the snapshot's cadence — one call to the existing reader, no second one.

**The block** is a `figures` paragraph and a short table at the head of
`#tab-models`, above the cards: "3 models downloaded, 1 loaded · 41.2 GB on
disk, 210 GB free · memory: 52.0 GB of 96 GB budget resident, of which 4.1 GB
still exiting". A model without a declared window shows no window on its card,
never zero. A budget the Mac cannot measure shows no share.

**The cards** gain one line: "context: 131,072 declared · 65,536 served" when
the served window is below the declared one, "context: 131,072" when they are
equal, and nothing when the model declares none. (The context probe's line,
already on the card, sits beneath it.)

**Loopback only.** The block is part of the panel, which the control plane
serves on this Mac alone; nothing here is served to the LAN.

## How each acceptance criterion is tested

| Criterion | Test |
| --- | --- |
| Three ready models, one resident: 3 downloaded, 1 loaded | `internal/ui`: `TestTheRollUpCountsDownloadedAndLoaded` (pure function over a snapshot) |
| Disk total is the sum of recorded sizes, no walk | the same, and `internal/archtest` or `internal/ui` asserting `resourcesSummary` reads `bytes` and never calls an endpoint |
| Free space comes from the capability package's one reader | `internal/gateway`: `TestTheSnapshotCarriesFreeDiskFromTheOneReader` (the machine block's field is `Assess`'s, and no second reader exists — a grep test in archtest for `statfs`/`Statfs` outside capability) |
| Resident memory includes exiting bytes and names them | `internal/ui`: `TestTheRollUpNamesTheExitingPart` |
| Served below declared: both shown, served resolved fold-aware | `internal/ui`: `TestTheCardShowsBothWindowsFoldAware` |
| No declared window: no window shown, never zero | `internal/ui`: `TestACardWithoutADeclaredWindowShowsNoWindow` |
| Off loopback: the block is unreachable | `internal/gateway`: the existing loopback-only test for the panel covers every panel asset; one assertion that the Models tab's markup is served by the same handler |

## Grounds

- pursued: no new measurement is needed, because every figure but free disk is
  already on the snapshot and free disk has one reader; shown wrong if the
  roll-up needs a directory walk or a second reader, as the intent's mechanism
  claim says.

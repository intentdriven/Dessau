---
schema_version: 1
id: "iss-2609111755330533"
slug: "internal-lifecycle-swap-go-claims-no-failure-path-leaves-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "second security pass on the lifecycle branch, 2026-09-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/lifecycle/swap.go"
resolution: "The retired bundle is now set aside in this account's own directory (config.AccountHome, a new single-source helper beside accountDir) instead of a staging directory inside the group-writable applications folder, so the unbounded wait can no longer be ended by a co-resident admin account unlinking the only copy; the incoming bundle still stages in the destination, a home that is not a real directory this account owns and nothing else can write, or that is on another volume from the destination, falls back to the old behaviour with a warning naming the reason, and swap.go's commentary now states the guarantee, the actor it holds against, and the one residual (a compromised admin account on this Mac, which needs no window)."
impact: fix
---

internal/lifecycle/swap.go claims no failure path leaves the Mac with no application. When the destination reappears mid-swap the retired bundle now waits in the kept staging directory (mode 0700, unguessable name) until a person moves it back; but a directory entry is removed by write permission on the PARENT, and the applications directory is group-writable to admin accounts with no sticky bit, so a co-resident admin-group account can delete the staging directory and leave no application for as long as the wait lasts — the same privilege tier the reappearance itself needs, so no new boundary, but the file's claim is not strictly true against that actor and the fix made the window unbounded. Say so in the file's commentary or an ADR note; a stickier home for the set-aside copy (this account's own directory) would close it.

## Grounds

- pursued: we expect the only remaining copy of the application to sit where only this account can unlink it because a directory entry is removed by write permission on the parent and ~/Library is 0700, while the rename that puts it back stays within one filesystem; wrong if the account's own directory is on another volume from the destination (the fallback and its warning), or if a Mac is found where ~/Library is not this account's alone.

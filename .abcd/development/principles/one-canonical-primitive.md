# One canonical primitive

## The rule

> One canonical primitive: before adding a generic-smelling primitive, find its
> canonical home and extend it; never add a third copy — see
> .abcd/development/principles/one-canonical-primitive.md.

## Why

The repository carries an open record of the rule being broken by exactly one
copy, and two records of it being honoured.

`iss-2609091714393599` is the standing case. Two size-rotating file writers
exist — `internal/stats/store.go`'s `storeWriter` and `internal/applog`'s
rotator — and the record is explicit both about why they were not unified when
the second arrived and about which of them is the canonical home: the store's
rotation is not a size-rotating writer with a store on top, because its file
names carry a UTC day and a counter, its pruning enforces two bounds, and what
it drops is folded into a summary that itself counts toward the ceiling.
`internal/applog`'s rotator is the plain primitive; the store's writer is the
specialisation. The record also names its own falsifier: the two drifting on
the discipline they share, one gaining a check on the opened handle, or a mode,
or a symlink refusal that the other lacks. That falsifier is the cost the rule
exists to avoid, written down before it is paid.

`iss-2609081427104462` shows the extraction done: one shared walker in
`internal/archtest` owns the skip rule and all six full-tree scans go through
it, with the walker itself tested against a file planted under a dot-directory,
in place of tests hand-rolling their own `WalkDir`. Its grounds name the
regression to watch for — a scan hand-rolling a walk again.

The `.abcd/work/DECISIONS.md` line of 2026-09-07 on the summary fold records the
same move at the store's read path: `FileStore.Read(ctx, ReadOptions, fn)` is
the store's one bounded read primitive, and `Summaries` takes the same shape
rather than a shape of its own.

Records drawn on: `iss-2609091714393599`, `iss-2609081427104462`, the
`.abcd/work/DECISIONS.md` line of 2026-09-07 on `itd-2609061602043757`.

## How to apply here

- Before writing a rotating writer, a tree walker, a bounded read or a capped
  file read, look for the existing one. `internal/applog`, `internal/archtest`,
  `internal/stats` and `internal/config` each already own one of these.
- Where unification is too large or too risky to do in the change that
  discovers it, the record says so, names the canonical home, and names what
  would show the deferral wrong — the shape `iss-2609091714393599` uses.
- Two copies are a captured observation; a third is a defect.

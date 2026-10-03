---
id: adr-2610030906462776
slug: dessau-serves-from-one-macos-account-the-machine-wide-shared
status: accepted; superseded in part by adr-2610031016411351 (the control-panel clause of Alternative 3)
date: 2026-10-03
supersedes: [adr-2609201008470380, adr-2609201008476813, adr-2609061610107154]
superseded_by: null
related_intents: [itd-2609081259532589, itd-2609091707499248, itd-2609061521102742, itd-2609062346072707, itd-2609100457007827]
related_rfcs: []
related_adrs: [adr-2609201008470380, adr-2609201008476813, adr-2609061610107154, adr-2609201008477513, adr-2609061503319212]
---

# ADR-2610030906462776: Dessau serves from one macOS account; the machine-wide shared model cache is retired

**Supersedes in part:**
[adr-2609201008470380](2609201008470380-the-gateway-may-retain-both-sides-of-a-conversation-in-a-tra.md),
on decision 7 (refused under the shared-cache install);
[adr-2609201008476813](2609201008476813-local-telemetry-may-record-prompt-text-and-completions-only.md),
on the words "refused outright under the shared-cache install" in its first
exception; and
[adr-2609061610107154](2609061610107154-statistics-store-format-json-lines-size-rotated-per-account.md),
on the sentence placing the store under the shared-cache install mode.

**Reverses:** the "Cross-account sharing by singleton election" entry of
`.abcd/development/decisions/DECISIONS.md` (the shared model cache in a
machine-wide folder), and acceptance criteria ac-18 and ac-19 of the shipped
intent itd-2609081259532589 (uninstall under a shared-cache installation, and
`--purge` removing only what this account owns there).

## Context

Since its first release Dessau has offered a shared-cache mode: `make
install-shared` created a machine-wide data root under `/Users/Shared`, mode
`3775` (setgid and sticky), so that several macOS accounts on one Mac kept one
copy of each model. Whoever launched first served, and the others deferred to
it. Each later change had to work around that root: per-account settings,
registry, logs, statistics and runtime moved into each account's own folder; a
settings adoption path carried old settings out of the shared folder; directory
widening carried the setgid and sticky bits into new directories; uninstall
learned to purge file by file by owner; and every state file's read was
hardened against a co-tenant planting a FIFO or a link under its name.

An adversarial security review on 2026-10-03 found that the mode could not be
made safe for model loading: with a model's files, and the folders above them,
owned by whichever account created them, a check made by the serving account
could not guarantee what the model server would go on to read. The fix that
landed the same day (refusing a model whose directory or `config.json` another
account owns) narrowed this but could not close it from inside one account, and
the maintainer chose to remove the mode rather than carry an open race in a
trust boundary.

Pre-1.0, the project carries no migration code: testers re-create state.

## Decision

We will serve from one macOS account. Dessau's data root is that account's own
`~/Library/Application Support/Dessau`, or wherever `DESSAU_ROOT` points, and
everything Dessau keeps — models, download cache, settings, registry, logs,
statistics, self-test results and the private runtime — is under it. Every other
account, on the same Mac or elsewhere, reaches the server over the network
(`localhost` from the same Mac) and never needs the model files.

The machine-wide root is removed: `config.SharedRoot` and its detection, the
per-account split of the layout, the settings adoption, the setgid and sticky
widening, uninstall's shared-root branch, the Makefile target, and the
documentation of the mode. An architecture test fails on any reintroduction in
code, tests, scripts, the Makefile, the docs or the README.

The ownership checks of 2026-10-03 stay as defence in depth: a model is loaded
only when its directory (the link and its target, for a link) and its
`config.json` belong to the serving account, and a model with no `config.json`
is refused.

The decisions this supersedes in part fall away with the mode they were about:
decision 7 of adr-2609201008470380 and the matching words of
adr-2609201008476813 refused the transcript under a shared-cache install,
which no longer exists; every other decision in both records stands. The
sentence of adr-2609061610107154 placing the statistics store under that mode
is moot; the store is in the serving account's own data root, as the rest of
that record (as since superseded) says.

## Alternatives Considered

1. **Keep the mode and close the race.** Rejected: what is at issue belongs to
   another account, and nothing inside this account's process can make it this
   account's to vouch for. Copying a model into the serving account before each
   load would double the disk the mode existed to save.
2. **Keep the mode, documented as unsafe for loading.** Rejected: a trust
   boundary with a known open race, documented, is still open; and the mode
   bought only disk space, which the network stance buys without a shared
   folder.
3. **Retire the mode; other accounts use the server over the network.**
   Chosen. One account owns every file Dessau reads, so the race cannot arise,
   and the other accounts lose nothing they need: the server, its models and its
   control panel are reachable from them over `localhost`.

## Consequences

- Easier: the layout is one folder again; a whole class of co-tenant defences
  becomes defence in depth rather than the boundary; uninstall acts on one
  account's files only.
- Harder: a Mac where two accounts each ran their own server keeps two copies of
  a model. The answer is to run one server and connect from the other account.
- Testers on a shared cache re-download their models into the serving account;
  the old folder under `/Users/Shared` is theirs to remove. `impact: breaking`.
- itd-2609091707499248's criterion "Refused under the shared-cache install, with
  the reason on the panel" and row 10 of its spec spc-2609201007360569 are
  withdrawn by this record; whoever builds that intent drops them, and any
  mention of `config.IsSharedRoot`, rather than building a refusal for a mode
  that does not exist.
- The shipped intents whose scope conditions describe shared-cache behaviour
  (among them itd-2609061521102742, itd-2609062346072707, itd-2609100457007827,
  itd-2609091712141073, itd-2609091903463596, itd-2609061441261073,
  itd-2609091412177263 and itd-2609061602043757) stay as the record of what
  shipped; those conditions describe a mode that is gone and constrain nothing
  further.
- The absolute-path pre-commit hook still exempts `/Users/Shared/`, because the
  changelog's history and these records name it.
- adr-2609201008470380, adr-2609201008476813 and adr-2609061610107154 change
  only their status fields, naming this record.

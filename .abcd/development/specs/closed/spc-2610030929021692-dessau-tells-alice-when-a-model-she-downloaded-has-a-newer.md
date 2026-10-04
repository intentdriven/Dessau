---
id: spc-2610030929021692
slug: dessau-tells-alice-when-a-model-she-downloaded-has-a-newer
intent: itd-2610030857275099
origin: researcher-authored
production_mode: hand-written
---
# Dessau checks downloaded models for newer versions and updates them safely

## Summary

Delivers itd-2610030857275099: an opt-in, scheduled check of each downloaded
model against HuggingFace, a mark in the panel when a file Dessau uses has
changed, and a one-click update that stages the new version, checks every file
and swaps it in atomically. Designed from the two adversarial reviews and the
planning interview of 2026-10-03; the rule for the outbound request is
adr-2610030857208746 (accepted).

## Approach

### Recording the version at download
Every download first resolves the repository's current commit and then fetches
every file at that commit (`resolve/<commit>/…`), never `main`, so a commit
landing mid-download cannot mix files. The registry records the commit and, per
downloaded file, its hash from the Hub's tree listing (the LFS sha256 for large
files, the git blob id for small ones). The field is shared with
spc-2610030846273729, which records the commit of a reviewed decision build.
Models with no recorded commit (everything downloaded before this ships) are
"version unknown". No backfill: pre-1.0.

### The check
- **Request.** `GET /api/models/<repo>?expand[]=sha` (about 100 bytes), sent
  without a token. On 401 or 403 with a token set, once more with it; still
  refused means "can't check" (a missing repository answers 401 anonymously,
  so it is never reported as gated; see iss-2610030913177383). Only when the
  commit differs from the recorded one, the tree at the new commit is listed
  and its hashes are compared with the recorded ones for the files Dessau
  downloads, ignoring documentation (`README.md` and other Markdown, licences,
  images the downloader already skips).
- **Mark.** A model is marked when any compared file differs or is added or
  removed. If the new `config.json` changed, it is fetched (it is small) and
  read with the same rule as the model-code refusal: a non-null `model_file`
  marks the version as one Dessau will not run, with no Update.
- **Pacing.** One repository at a time with a pause between, sharing the Hub
  client and the category job's pacing; the `ratelimit` headers are honoured and
  the round stops early when the remaining budget is low.
- **Scheduling.** A job beside the category job (`jobsCtx`/`jobsWG`), not idle
  work: it never takes the pool lock or the download lock. The last-check time
  is persisted; at startup the round runs, after a short delay with jitter,
  only if the last check is older than the interval, and the schedule compares
  wall-clock time with the stored time, so sleep and wake do not skew it.
- **Offline or failing.** Marks stay as they were; one info line per round.
- **Decision models.** Checked like any model; a changed version is marked "will
  be offered once reviewed", and Update appears only when the available commit
  matches one in the embedded reviewed manifest that a Dessau release ships.

### The setting
`update_check_enabled` (off by default) and `update_check_interval_hours`
(1 to 720, default 24) in `config.json`, repaired on load like the idle
threshold; a save is never refused over either field when it was not touched.
The panel's Settings carries the switch and the interval, and beside the switch
one sentence states what a check sends and to whom, held by a test the way the
bridge's egress sentence is. The three-surfaces tests cover both fields.

### The update
Update stages the new version in a sibling directory at the target commit,
after checking free disk space for the whole download. Every file is
hash-checked against the tree listing; the staged directory passes the same
Precheck as any launch (`model_file`, ownership, `config.json`). Only then is
the swap made: the model is drained from the pool, the old directory renamed
aside, the staged one renamed in, the old one removed after the new one passes
readiness. Any failure removes the staged directory and leaves the old version
serving. Per-model facts tied to the files (the measured context, the
tool-call verdict, a recorded load failure) are cleared as a re-download clears
them today; settings keyed by repository (pin, served-context override) stay.
The ordinary re-download moves onto the same staged path, which resolves
iss-2610030913179523.

### The panel
The model card shows the mark, "version unknown", "will be offered once
reviewed", or "ships code Dessau will not run", and the Update button where
allowed. Every name that comes from HuggingFace (repository, file) is rendered
as text through the panel's escaping helper, with a test.

## Acceptance criteria and what holds each

1. Off sends nothing: an app test with a recording Hub fake over a simulated
   week; a save test that an unrelated save leaves the switch off.
2. On, at start and interval: scheduler tests on stored last-check times.
3. Only real changes: a check test where only `README.md` changed.
4. Version unknown: a registry and panel test for a model with no commit.
5. Token only if needed: a hub test asserting no Authorization header on the
   first request, and the token only after a refusal.
6. Offline: a check test with the Hub unreachable, marks unchanged, and a
   serving-latency test under a stalled Hub.
7. Safe one-click update: download tests for staging, a hash mismatch, a
   failure mid-download and the atomic swap, with the old version serving
   throughout.
8. Update with own code: a check test where the new `config.json` names
   `model_file`.
9. Decision models wait: a test against the reviewed manifest.
10. Setting everywhere: the three-surfaces tests, repair on load, the untouched-
    field save test.
11. Names shown safely: a UI test with markup in a repository and file name.

## Steps

1. Recording the version at download
   - packages: internal/hub, internal/registry, internal/app
   - tests: every file fetched at one resolved commit, never main; the commit and per-file hashes recorded; models without them read "version unknown"
   - landed: #149
2. The check and its setting
   - packages: internal/hub, internal/app, internal/config, internal/ui
   - tests: off sends nothing (a recording Hub fake over a simulated week); no token unless refused; only changes to files Dessau reads mark; a new model_file marks "will not run"; pacing and rate-limit headers; offline leaves marks; the setting on all three surfaces with repair on load and the untouched-field save; the egress sentence beside the switch
   - landed: #152
3. The staged update
   - packages: internal/hub, internal/app, internal/runtime
   - tests: staging at the target commit, disk-space check, every file hash-checked, Precheck on the staged directory, drain and atomic swap, failure leaves the old version serving; the ordinary re-download moved onto the same path (resolves iss-2610030913179523)
   - landed: #162
4. The panel
   - packages: internal/ui, internal/gateway
   - tests: the mark, "version unknown", "will be offered once reviewed", "ships code Dessau will not run", the Update button where allowed; names from HuggingFace rendered as text
   - landed: #165

## Footprint

- packages: internal/hub, internal/registry, internal/app, internal/config, internal/runtime, internal/ui, internal/gateway
- tests: the eleven acceptance criteria as listed above

## Out of scope

Updating automatically; checking at the operator's actions only (rejected in
adr-2610030857208746); models that did not come from HuggingFace.

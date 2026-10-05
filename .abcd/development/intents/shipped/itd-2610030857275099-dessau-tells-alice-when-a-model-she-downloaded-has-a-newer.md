---
id: itd-2610030857275099
slug: dessau-tells-alice-when-a-model-she-downloaded-has-a-newer
spec_id: spc-2610030929021692
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030656210408, itd-2609081259493890, itd-2609100519003748]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Dessau checks downloaded models for newer versions, and updates them with one click

## Press Release

Dessau tells Alice when a model she downloaded has a newer version, and updates
it with one click.

Alice turns on update checks in Settings and picks how often, once a day to
start with. From then on, at every start and every interval, Dessau asks
HuggingFace whether the files of each model she downloaded have changed, and
the control panel marks the ones that have. One click on Update fetches the new
version beside the old one and swaps it in only once every file checks out, so
Carol's chat never meets a half-updated model. Nothing is asked until Alice
turns checks on, and a decision model is never marked.

## Why This Matters

Downloads take a repository's latest files once and are never revisited. Fixes
to chat templates, configs and quantisations land on HuggingFace silently, so
Alice serves a stale build and finds out only when a client misbehaves. And the
one way to update today, downloading again, can leave a model that is half the
old version and half the new (iss-2610030913179523). Seeded 2026-10-03 from the
maintainer's request.

## Mechanism

- We expect Alice to update models she would otherwise leave stale, because the
  mark appears in the panel she already uses. Shown wrong if marked models stay
  un-updated release after release.
- We expect Alice to act on marks because they appear only when a file Dessau
  actually uses has changed, never for a README edit. Shown wrong if a mark
  appears for an upstream change that touches no file Dessau reads.

## Scope Conditions

- Only when Alice has turned checks on; a Mac whose operator never does sends <!-- cond: cond-2610030929026824 -->
  nothing.
- A model is marked only if Dessau recorded which version it downloaded; older <!-- cond: cond-2610030929020636 -->
  downloads show "version unknown" until updated once.
- Models from HuggingFace; public ones are checked without the token, private <!-- cond: cond-2610030929022838 -->
  ones with it only when an anonymous request is refused.
- The Mac is online (offline, marks stay as they were); a decision model is <!-- cond: cond-2610030929025733 -->
  checked, but offered an update only once a Dessau release has reviewed that
  version.

## Acceptance Criteria

- **Given** checks are off, **when** Dessau runs, **then** no check leaves the
  Mac, and saving any other setting never turns checks on.
- **Given** checks are on, **when** Dessau starts (if the last check is older
  than the interval) and each interval passes, **then** every eligible model is
  checked and changed ones are marked.
- **Given** an upstream change that touches only files Dessau does not use (a
  README edit), **when** a check runs, **then** no mark appears.
- **Given** a model whose version Dessau never recorded, **when** Alice opens
  its card, **then** it says "version unknown", shows no mark, and offers
  Update.
- **Given** a token in Settings and a public model, **when** a check runs,
  **then** the request carries no token; a private model gets it only after an
  anonymous request is refused.
- **Given** HuggingFace cannot be reached, **when** a check runs, **then** marks
  stay as they were, one line is logged, and serving is not slowed.
- **Given** a marked model, **when** Alice clicks Update, **then** the new
  version downloads beside the old one at that exact upstream version, every
  file is checked, and it is swapped in only when complete; if anything fails
  the old one keeps serving, so Carol's chat never meets a half-updated model.
- **Given** a newer version that ships its own code, **when** it is checked,
  **then** the mark says Dessau will not run that version and offers no Update.
- **Given** a decision model whose upstream version changed, **when** Alice
  opens its card, **then** it says a newer version exists and will be offered
  once reviewed, with no Update; Update appears when a Dessau release ships the
  reviewed version.
- **Given** the setting (on or off; an interval from 1 hour to 30 days, daily by
  default), **when** it is read from Go, the panel or `config.json`, **then** all
  three agree; an out-of-range value in `config.json` is repaired on load, and
  saving an unrelated setting is never refused over it.
- **Given** a model or file name from HuggingFace containing markup, **when** the
  panel shows a mark, **then** the name appears as plain text.

## Decisions at the planning interview (2026-10-03)

1. On an update: a mark plus a one-click update. The safe update (staged at the
   exact upstream version, every file checked, atomic swap, the old copy kept
   until the new one is in) is part of this intent; the reviews had proposed it
   as a separate one.
2. When: off until the operator turns checks on. Under the 2026-09-08
   precedent this needs no supersession of adr-2609201008476813;
   adr-2610030857208746 records the rule for this check.
3. The existing startup category lookup counts as part of fetching models
   (recorded in adr-2610030857208746); it stays always on.
4. The token: sent only when an anonymous request is refused.
5. Models with no recorded version: "version unknown" plus Update.
6. Interval: daily by default, 1 hour to 30 days.
7. Decision models: checked, with no Update until a Dessau release has
   reviewed the version (the maintainer asked why they should differ; the
   answer is itd-2610030656210408's "reviewed version only").

Typed links: builds on itd-2610030656210408 (one recorded-revision field;
"reviewed version only"), itd-2609081259493890 (three surfaces) and
itd-2609100519003748 (where the panel shows a model's state). Resolves, when
built, iss-2610030913179523 (a re-download can leave a mixed model), since the
update path replaces it.

## Open Questions

_None open._ Settled in spc-2610030929021692: the ordinary re-download moves
onto the same staged path as Update (its step 3), which resolves
iss-2610030913179523.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-c60839654fc3 -->
Fidelity review — receipt rcp-c60839654fc3 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:8830a4a65ccddbb174917335c967ed5442132dcfa29d8d8b0659894d567e1196
Input attestations: pr:intentdriven/Dessau#149 merge 8e4d2cc4806865dbfb371b11911225c40b649caa (step 1: record the downloaded version)@-; pr:intentdriven/Dessau#152 merge 864042b98223b6f33b1e632b925c9100612a79f4 (step 2: the check and its setting)@-; pr:intentdriven/Dessau#162 merge 8c088a4f3ebed7a2fdfdafb584965a1b04108f88 (step 3: the staged update)@-; pr:intentdriven/Dessau#165 merge 63790e21038bdd36bfb4d32b471fb7a3f0fb4b9f (step 4: the panel)@-; tree:integrate/0.10.0 at dd929d8a7dbe0446792423b43594d8b84583561e; go test -count=1 over internal/{app,hub,config,ui,archtest,registry,gateway} all ok@-;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 3 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The schedule returns before anything is due when the switch is off, and the round re-reads the switch before every model. A week of ticks with checks off sends no request. The switch is off by default, and an unrelated settings save leaves it off. The only callers of Latest, FilesAt and SmallFileAt are in the check.
  evidence: internal/app/updatecheck.go:88 — "if !a.Config().UpdateCheck {"
  evidence: internal/app/updatecheck.go:160 — "if !a.Config().UpdateCheck {"
  evidence: internal/app/updatecheck_test.go:162 — "func TestUpdateChecksOffSendNothingOverAWeek(t *testing.T) {"
  evidence: internal/config/updatecheck_test.go:12 — "func TestUpdateChecksAreOffByDefaultAndDailyWhenOn(t *testing.T) {"
  evidence: internal/gateway/version_test.go:46 — "func TestAnUnrelatedSaveLeavesTheUpdateCheckAsItWas(t *testing.T) {"
- ac-2 — MET: main starts the schedule. The first tick comes after a 30-60 s jittered delay, then one tick a minute compares wall-clock time with each model's persisted CheckedAt. Every ready model with a known version that is past the interval is checked, and a changed one is recorded as available. Tests cover a check at the first tick, 7-8 rounds over a week of daily checks, and no request at a start within the interval.
  evidence: cmd/dessau/main.go:363 — "a.StartCheckingForUpdates()"
  evidence: internal/app/updatecheck.go:74 — "if m.Update != nil && now.Sub(m.Update.CheckedAt) < interval {"
  evidence: internal/app/updatecheck_test.go:180 — "func TestUpdateChecksOnRunAtStartAndEachInterval(t *testing.T) {"
  evidence: internal/app/updatecheck_test.go:204 — "func TestAStartWithinTheIntervalAsksNothing(t *testing.T) {"
  evidence: internal/app/updatecheck_test.go:248 — "func TestAChangedWeightMarksTheModel(t *testing.T) {"
- ac-3 — MET: compareVersions skips documentation (Markdown and licence or notice files) on both sides of the comparison. A test where only README.md and LICENSE changed upstream records the model as current and never reads config.json.
  evidence: internal/app/updatecheck.go:293 — "if isDocumentation(f.Path) {"
  evidence: internal/app/updatecheck.go:332 — "func isDocumentation(p string) bool {"
  evidence: internal/app/updatecheck_test.go:230 — "func TestAReadmeOnlyChangeMarksNothing(t *testing.T) {"
- ac-4 — MET: A model with no recorded commit has VersionKnown() false and is never checked, so it carries no mark. The card says 'Version unknown'. updateMark returns no pill without a commit, and updateOffered offers Update when there is no commit. Update then fetches the current version. Registry, app and panel tests hold each part.
  evidence: internal/registry/registry.go:245 — "func (m Model) VersionKnown() bool { return m.Commit != "" }"
  evidence: internal/ui/static/app.js:577 — "if (!m.commit) return 'Version unknown: Dessau did not record which version it downloaded. Update fetches the current one.';"
  evidence: internal/ui/static/app.js:606 — "return !m.commit || (m.update || {}).status === 'available';"
  evidence: internal/ui/update_test.go:27 — "{`{"state":"ready"}`, `Version unknown`, \`\`, true},"
  evidence: internal/app/updatecheck_test.go:216 — "func TestAModelOfUnknownVersionIsNotChecked(t *testing.T) {"
  evidence: internal/registry/version_test.go:58 — "func TestAModelWithNoRecordedCommitIsVersionUnknown(t *testing.T) {"
- ac-5 — MET: Latest makes the commit lookup with an empty token, so newTokenRequest sets no Authorization header. It retries with the token only on a 401 or 403 when a token is set, and the listing and config.json then use the same choice. Hub tests check that a public repository gets no Authorization on any request, that the sequence for a private one is anonymous then Bearer, and that a refusal with no token makes no second request.
  evidence: internal/hub/hub.go:355 — "func (c *Client) Latest(ctx context.Context, repoID string) (Upstream, error) {"
  evidence: internal/hub/hub.go:360 — "if !IsAuthRequired(err) || token == "" {"
  evidence: internal/hub/check_test.go:72 — "func TestACheckOfAPublicRepositorySendsNoToken(t *testing.T) {"
  evidence: internal/hub/check_test.go:98 — "func TestACheckSendsTheTokenOnlyAfterARefusal(t *testing.T) {"
- ac-6 — MET: A model the Hub does not answer for keeps its record, and a round that reaches no model logs exactly one info line. The tests check that a closed Hub leaves an existing mark identical and writes one log line, and that while the Hub stalls, registry, pool and download reads finish in under 200 ms because the round holds no serving lock and each request is bounded.
  evidence: internal/app/updatecheck.go:198 — "a.Log.Info("could not reach HuggingFace to check models for newer versions; marks stay as they were","
  evidence: internal/app/updatecheck_test.go:304 — "func TestAnUnreachableHubLeavesMarksAsTheyWere(t *testing.T) {"
  evidence: internal/app/updatecheck_test.go:331 — "func TestAStalledHubDoesNotSlowServing(t *testing.T) {"
- ac-7 — MET_WITH_CONCERNS: Update fetches the marked commit into a staging folder, hash-checks every file and runs validateModelDir and Precheck before swapping. Tests check the staged swap, the exact marked commit, a hash mismatch, a mid-download failure, a version that ships code and no disk space, all of which leave the old version serving. Concern 1: spec step 3 says the old copy is removed only after the new one passes readiness. DECISIONS 2026-10-03 reads readiness as the directory checks, so the old copy is removed as soon as the new record is published (app.go:1759). A version that passes Precheck but fails at its first load has no old copy to fall back to, which narrows 'if anything fails the old one keeps serving'. Concern 2: while the swap drains the model, new requests for it are refused with 503 (ErrUpdating) rather than served by the old version.
  evidence: internal/app/staged.go:254 — "func (a *App) Update(repoID string) error {"
  evidence: internal/app/staged_test.go:211 — "func TestAnUpdateStagesTheNewVersionAndSwapsItInWhole(t *testing.T) {"
  evidence: internal/app/staged_test.go:242 — "func TestAnUpdateFetchesTheMarkedCommit(t *testing.T) {"
  evidence: internal/app/staged_test.go:263 — "func TestAFailedUpdateLeavesTheOldVersionServing(t *testing.T) {"
  evidence: internal/app/staged_test.go:323 — "func TestTheSwapDrainsTheModelFirst(t *testing.T) {"
  evidence: internal/app/app.go:1759 — "a.removeAside(repoID, st.aside)"
  evidence: .abcd/work/DECISIONS.md:407 — "The spec's "the old one removed after the new one passes readiness" is read as the directory checks a launch makes"
  evidence: .abcd/work/DECISIONS.md:410 — "an Acquire for a drained model, loaded or not, is refused with ErrUpdating (503, counted as busy)"
- ac-8 — MET_WITH_CONCERNS: When the newer config.json changed and names a non-null model_file, the check records runs_own_code. The card says the version 'ships its own code, which Dessau will not run, so it is not offered', Update is not offered, and Update() refuses that status. Concern: when the repository keeps config.json in LFS (served from the content CDN), the check cannot read it and records the version as available, with Update offered. Only the update's own Precheck refuses it later, so the mark does not say 'will not run' in that case.
  evidence: internal/app/updatecheck.go:251 — "return registry.UpdateCheck{Status: registry.UpdateRunsOwnCode, Commit: up.Commit}, nil"
  evidence: internal/ui/static/app.js:582 — "case 'runs_own_code': return `A newer version (${short}) ships its own code, which Dessau will not run, so it is not offered.`;"
  evidence: internal/app/updatecheck_test.go:264 — "func TestANewerVersionThatShipsCodeIsMarkedAsOneDessauWillNotRun(t *testing.T) {"
  evidence: internal/app/updatecheck.go:238 — "case errors.Is(err, hub.ErrCrossOrigin):"
  evidence: internal/app/updatecheck_test.go:519 — "func TestAConfigOnTheContentCDNDoesNotStopTheCheck(t *testing.T) {"
- ac-9 — NOT_MET: Promised: a decision model whose upstream changed shows 'will be offered once reviewed' with no Update, and Update appears when a Dessau release ships the reviewed version. Delivered: the gate is a ReviewedBuilds seam (Options.Reviewed), and 'Nil means no model is a decision model'. The shipped binary calls app.New without Reviewed and ships no reviewed manifest; that manifest is itd-2610030656210408 step 2, which has not landed. In the delivered build, no model is ever marked awaiting_review. A changed decision model would be marked 'available' with Update offered, and no release can make Update appear by shipping a reviewed version. The awaiting_review path exists only in tests that inject a.reviewed by hand.
  evidence: cmd/dessau/main.go:338 — "a, err := app.New(app.Options{Paths: paths, Config: cfg, Log: log, Bind: plan, LogLevel: appLog.Level})"
  evidence: internal/app/app.go:242 — "// Nil means no model is a decision model."
  evidence: internal/app/updatecheck.go:255 — "if a.reviewed != nil {"
  evidence: internal/app/updatecheck_test.go:395 — "a.reviewed = func(id string) (bool, []string) {"
  evidence: .abcd/work/CONTEXT.md:97 — "Criterion 9 waits on itd-2610030656210408 step 2."
- ac-10 — MET: Go holds update_check_enabled (off by default) and update_check_interval_hours, bounded 1 to 720 with 24 as the default. Load repairs an out-of-range interval and names the repair. The panel fills from and posts both keys, and its interval control is held to the configuration's bounds. An unrelated save keeps the setting and is not refused over it. Tests cover the config, gateway, ui and archtest parts.
  evidence: internal/config/config.go:1223 — "DefaultUpdateCheckIntervalHours = 24"
  evidence: internal/config/config.go:1244 — "func (c *Config) sanitizeUpdateCheck() []string {"
  evidence: internal/config/updatecheck_test.go:49 — "func TestAnOutOfRangeIntervalInTheFileIsRepairedOnLoad(t *testing.T) {"
  evidence: internal/ui/updatecheck_test.go:11 — "func TestThePaneIsWiredToTheUpdateCheckSettings(t *testing.T) {"
  evidence: internal/ui/controls_test.go:138 — "{"setUpdateCheckInterval", true, func(c *config.Config, v float64) { c.UpdateCheckIntervalHours = int(v) }},"
  evidence: internal/gateway/version_test.go:46 — "func TestAnUnrelatedSaveLeavesTheUpdateCheckAsItWas(t *testing.T) {"
- ac-11 — MET_WITH_CONCERNS: The card puts the repository name, the mark and the version line into innerHTML through escapeHtml, which goes through textContent, and only in text positions. The commit shown is held to a commit id by the registry. Concern: the test only pattern-matches the source for escapeHtml calls. It is not the rendered test with markup in a repository and a file name that the spec planned. The card shows no file names at all, so the file-name half is never exercised.
  evidence: internal/ui/static/app.js:475 — "if (mark) pill += `<span class="pill update">${escapeHtml(mark)}</span>`;"
  evidence: internal/ui/static/app.js:484 — "${update ? `<div class="info update">${escapeHtml(update)}</div>` : ''}"
  evidence: internal/ui/static/app.js:2840 — "function escapeHtml(s) {"
  evidence: internal/ui/update_test.go:52 — "func TestTheVersionLinesAreEscaped(t *testing.T) {"

Gap audit:
- honoured:
  - Nothing is asked until Alice turns checks on
    evidence: internal/app/updatecheck.go:88 — "if !a.Config().UpdateCheck {"
    evidence: internal/app/updatecheck_test.go:162 — "func TestUpdateChecksOffSendNothingOverAWeek(t *testing.T) {"
  - At every start and every interval Dessau asks whether files of each downloaded model changed, and marks only changes to files Dessau uses
    evidence: internal/app/updatecheck.go:278 — "func (a *App) compareVersions(m registry.Model, files []hub.File) (changed, configChanged bool) {"
    evidence: internal/app/updatecheck_test.go:230 — "func TestAReadmeOnlyChangeMarksNothing(t *testing.T) {"
  - The token is sent only when an anonymous request is refused
    evidence: internal/hub/check_test.go:98 — "func TestACheckSendsTheTokenOnlyAfterARefusal(t *testing.T) {"
  - One click fetches the new version beside the old one and swaps it in only once every file checks out; the ordinary re-download takes the same path (resolves iss-2610030913179523)
    evidence: internal/app/staged_test.go:211 — "func TestAnUpdateStagesTheNewVersionAndSwapsItInWhole(t *testing.T) {"
    evidence: internal/app/staged_test.go:361 — "func TestARedownloadNeverLeavesAMixedModel(t *testing.T) {"
  - Setting held on Go, panel and config.json, with repair on load
    evidence: internal/config/updatecheck_test.go:49 — "func TestAnOutOfRangeIntervalInTheFileIsRepairedOnLoad(t *testing.T) {"
    evidence: internal/archtest/update_check_egress_test.go:17 — "func TestTheUpdateCheckSwitchCarriesTheEgressSentence(t *testing.T) {"
- diverged:
  - Old copy kept until the new one passes readiness: the delivery reads 'readiness' as the directory checks and removes the old copy once the new record is published, so a load failure after the swap has no fallback
    evidence: .abcd/work/DECISIONS.md:407 — "is read as the directory checks a launch makes (validateModelDir and the launcher's Precheck, again in place after the rename), not as starting a model server"
    evidence: internal/app/app.go:1759 — "a.removeAside(repoID, st.aside)"
  - Carol's chat is served by the old version until the swap: during the drain and swap, requests for the model are refused with 503 instead
    evidence: .abcd/work/DECISIONS.md:410 — "A swap refuses new requests for the model while it drains"
  - A version that ships its own code is marked 'will not run': when its config.json is stored in LFS, it is marked available and offered instead
    evidence: internal/app/updatecheck.go:238 — "case errors.Is(err, hub.ErrCrossOrigin):"
  - Press release says 'a decision model is never marked'; the acceptance criteria and delivery give it a mark ('newer version awaiting review')
    evidence: internal/ui/static/app.js:596 — "case 'awaiting_review': return 'newer version awaiting review';"
- missing:
  - Decision models wait for a reviewed version: no reviewed manifest is shipped and app.New is called without Options.Reviewed, so the gate never fires in the delivered binary (depends on itd-2610030656210408 step 2)
    evidence: cmd/dessau/main.go:338 — "a, err := app.New(app.Options{Paths: paths, Config: cfg, Log: log, Bind: plan, LogLevel: appLog.Level})"
    evidence: internal/app/app.go:242 — "// Nil means no model is a decision model."
  - The spec's rendered UI test with markup in a repository name and a file name: delivered as a pattern match on the source instead
    evidence: internal/ui/update_test.go:52 — "func TestTheVersionLinesAreEscaped(t *testing.T) {"

Scope-condition dispositions:
- cond-2610030929026824 — survived: Every request the check makes is gated on the switch, which is off by default. A week of ticks with it off reaches the Hub zero times, and turning it off mid-round stops the round.
  evidence: internal/app/updatecheck.go:88 — "if !a.Config().UpdateCheck {"
  evidence: internal/app/updatecheck_test.go:162 — "func TestUpdateChecksOffSendNothingOverAWeek(t *testing.T) {"
  evidence: internal/app/updatecheck_test.go:428 — "func TestTurningChecksOffStopsARoundInFlight(t *testing.T) {"
- cond-2610030929020636 — survived: Only models with a recorded commit are due a check. A model without one reads 'Version unknown' and is offered Update. Every download records the commit and file hashes it fetched, so one update makes the version known.
  evidence: internal/app/updatecheck.go:71 — "if !m.VersionKnown() {"
  evidence: internal/ui/static/app.js:577 — "if (!m.commit) return 'Version unknown: Dessau did not record which version it downloaded. Update fetches the current one.';"
  evidence: internal/app/version_test.go:45 — "func TestDownloadRecordsTheVersionItFetched(t *testing.T) {"
- cond-2610030929022838 — survived: The check asks the HuggingFace API anonymously and adds the token only after a 401 or 403. Hub tests hold both the public and the private sequence.
  evidence: internal/hub/hub.go:355 — "func (c *Client) Latest(ctx context.Context, repoID string) (Upstream, error) {"
  evidence: internal/hub/check_test.go:72 — "func TestACheckOfAPublicRepositorySendsNoToken(t *testing.T) {"
  evidence: internal/hub/check_test.go:98 — "func TestACheckSendsTheTokenOnlyAfterARefusal(t *testing.T) {"
- cond-2610030929025733 — narrowed: The offline clause holds: an unreachable Hub leaves marks as they were. The decision-model clause holds only behind the ReviewedBuilds seam. The shipped binary passes no Reviewed function and no reviewed manifest exists, so no model is treated as a decision model, and gating to a reviewed version is exercised only by tests that inject one.
  narrowing: The offline clause holds as stated. 'A decision model is offered an update only once a Dessau release has reviewed that version' holds only when a ReviewedBuilds manifest is supplied to app.New. The delivered binary supplies none (pending itd-2610030656210408 step 2), so in the shipped build no model is distinguished as a decision model.
  evidence: internal/app/updatecheck_test.go:304 — "func TestAnUnreachableHubLeavesMarksAsTheyWere(t *testing.T) {"
  evidence: internal/app/updatecheck_test.go:386 — "func TestADecisionModelWaitsForAReviewedVersion(t *testing.T) {"
  evidence: cmd/dessau/main.go:338 — "a, err := app.New(app.Options{Paths: paths, Config: cfg, Log: log, Bind: plan, LogLevel: appLog.Level})"
<!-- abcd-review-end receipt=rcp-c60839654fc3 -->

## Grounds

- pursued: fixes to chat templates, settings and quantisations land on HuggingFace silently, so Alice serves stale builds without knowing; shown wrong if, over a month of daily checks, no model she uses changes in a file Dessau reads. And the only way to update today, downloading again, can leave a half-old, half-new model, which a safe one-click update fixes; shown wrong if no one ever re-downloads a changed model.

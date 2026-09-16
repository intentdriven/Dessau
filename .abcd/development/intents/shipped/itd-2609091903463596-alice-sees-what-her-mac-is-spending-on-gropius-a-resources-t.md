---
id: itd-2609091903463596
slug: alice-sees-what-her-mac-is-spending-on-gropius-a-resources-t
spec_id: spc-2609100450257929
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609061431481936, itd-2609061441261073, itd-2609061441228998]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Alice sees what her Mac is spending on Gropius. A Resources tab in the control panel shows how many models are downloaded and how many are loaded, how much disk the models take and how much the volume has left, the memory budget against what is resident and what is still exiting, and each model's declared and served context window. The figures are the ones Gropius already knows, shown in one place; nothing in it is a request or a prompt.

## Press Release

Alice's Mac is nearly full and she cannot tell what Gropius is holding. She
opens the Models tab and the first thing it shows is the roll-up: how many
models are downloaded and how many are loaded, how much disk they take and how
much the volume has left, and the memory budget against what is resident and
what is still on its way out. Each model's card says the window it declares
and the window it is set to serve.

## Why This Matters

The figures exist but are scattered: per-model sizes and windows on the cards,
the budget against resident bytes in Settings, RAM and free disk on the search
tab. Nothing adds them up, and the question a full Mac asks first has no
single answer. The posture page (itd-2609081718534201) says what is on; this
says what it costs; the measurement intent (itd-2609091712141073) says what
requests did. Placement decided at interview on 2026-09-09: a block at the
head of the Models tab, not a seventh tab.

## Mechanism

We expect no new measurement to be needed, because every figure but the
volume's free space is already on the state snapshot the panel polls, and free
space has one reader in the capability package. Shown wrong if the roll-up
needs a directory walk or a second reader.

## Scope Conditions

- Apple Silicon macOS. <!-- cond: cond-2609100450250935 -->
- The panel is loopback-only, so the figures are the operator's. <!-- cond: cond-2609100450255120 -->
- Shared-cache mode reports every local account's models. <!-- cond: cond-2609100450250643 -->
- Figures are read at the snapshot's cadence and are not live usage; that is the measurement intent. <!-- cond: cond-2609100450254408 -->

## Acceptance Criteria

- Given three ready models and one resident, when the Models tab renders, then the roll-up shows 3 downloaded and 1 loaded.
- Given models with recorded sizes, when the disk total is shown, then it equals the sum of those sizes with no directory walk on the render path.
- Given the models volume has space free, when the roll-up renders, then the figure comes from the capability package's one reader and is shown.
- Given a server still exiting, when resident memory is shown, then it includes the exiting bytes and names that part.
- Given a model with a served window below its declared one, when its card renders, then both windows are shown and the served one is resolved the fold-aware way.
- Given a model without a declared window, when its card renders, then no window is shown, never zero.
- Given a request from off loopback, when it asks for the panel, then the block is unreachable.

## Open Questions

- Resolved 2026-09-09 at interview: a block at the head of the Models tab rather than a new tab; activity stays with the usage dashboard and the measurement intent.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-81fed0a81798 -->
Fidelity review — receipt rcp-81fed0a81798 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:d9c252bfb13a9a0c8ee87ee89bdf81972a25b0866178fccf34acb0968ac14bff · prompt_hash sha256:df493748cef37b75cf333e84525b36bb82e0438fbf2a0c90eb76a7661025ad30
Input attestations: tree:main at acacd07b9fb29a2dfc94a8aa58da22d67e8d8b36, range 6b1f600..acacd07 (PR 52 carried four intents; only itd-2609091903463596's criteria were judged, with `go test ./internal/ui/ ./internal/gateway/ ./internal/archtest/` passing on the tree)@sha256:unknown; request:.abcd/.work.local/reviews/rcp-81fed0a81798.request.md@sha256:d9c252bfb13a9a0c8ee87ee89bdf81972a25b0866178fccf34acb0968ac14bff; intent:.abcd/development/intents/shipped/itd-2609091903463596-alice-sees-what-her-mac-is-spending-on-gropius-a-resources-t.md@sha256:df493748cef37b75cf333e84525b36bb82e0438fbf2a0c90eb76a7661025ad30;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: resourcesSummary counts the snapshot's ready models as downloaded and its loaded residents as loaded, renderModels writes its text into the #resources paragraph at the head of #tab-models on every state event, and a ui test over three ready models and one resident asserts downloaded 3, loaded 1 and the text '3 models downloaded, 1 loaded'.
  evidence: internal/ui/static/app.js:92 — "const ready = models.filter((m) => m.state === 'ready');"
  evidence: internal/ui/static/app.js:93 — "const loaded = resident.filter((r) => r.state === 'loaded').length;"
  evidence: internal/ui/static/app.js:268 — "$('resources').textContent = resourcesSummary(state).text;"
  evidence: internal/ui/static/index.html:60 — "< p id="resources" class="figures">< /p>"
  evidence: internal/ui/resources_test.go:15 — "func TestTheRollUpCountsDownloadedAndLoaded(t *testing.T) {"
  evidence: internal/ui/resources_test.go:78 — "func TestTheRollUpIsDrawnAtTheHeadOfTheModelsTab(t *testing.T) {"
- ac-2 — MET: The disk figure is a reduce over the ready models' recorded bytes on the snapshot; the snapshot's models come from Registry.List, which copies the in-memory records under a read lock, and the only directory read in the registry is Rescan's, which is not on the state path. The ui test holds disk to the sum of the three recorded sizes by evaluating the pure function with no DOM and no fetch.
  evidence: internal/ui/static/app.js:95 — "const disk = ready.reduce((sum, m) => sum + (m.bytes || 0), 0);"
  evidence: internal/registry/registry.go:320 — "func (r *Registry) List() []Model {"
  evidence: internal/registry/registry.go:327 — "out := make([]Model, 0, len(r.models))"
  evidence: internal/registry/registry.go:627 — "func (r *Registry) Rescan(modelsDir string) error {"
  evidence: internal/gateway/control.go:397 — "Models: c.App.Registry.List(),"
  evidence: internal/ui/resources_test.go:27 — "if got["disk"] != float64(6*1073741824) {"
- ac-3 — MET_WITH_CONCERNS: handleState sets Machine.FreeDisk from capability.Assess at the snapshot's cadence, Assess's freeDisk is the only Statfs in internal/ and an archtest walks every non-test .go file outside internal/capability to keep it so; a gateway test reads /api/state and holds machine.free_disk to Assess's figure, and the roll-up prints it as ', N free'. Concern: the panel appends the free figure only inside the `downloaded > 0` branch, so with a volume that has space free but no downloaded model the figure is on the snapshot yet not shown — the criterion's given (space free) is narrower in delivery than written; and the gateway test t.Skips when the volume reports no free space, so on such a machine the criterion has no test.
  evidence: internal/gateway/control.go:461 — "st.Machine.FreeDisk = capability.Assess(c.App.Paths.Models, c.App.MachineRAM(), budget).FreeDisk"
  evidence: internal/capability/capability_darwin.go:54 — "FreeDisk: freeDisk(modelsDir),"
  evidence: internal/capability/capability_darwin.go:86 — "if err := syscall.Statfs(dir, &st); err != nil {"
  evidence: internal/archtest/disk_reader_test.go:14 — "func TestFreeDiskHasOneReader(t *testing.T) {"
  evidence: internal/gateway/control_resources_test.go:13 — "func TestTheSnapshotCarriesFreeDiskFromTheOneReader(t *testing.T) {"
  evidence: internal/gateway/control_resources_test.go:20 — "t.Skip("the models volume reports no free space to measure against")"
  evidence: internal/ui/static/app.js:107 — "if (out.downloaded > 0) {"
  evidence: internal/ui/static/app.js:108 — "parts.push(`${bytes(out.disk)} on disk` + (out.freeDisk ? `, ${bytes(out.freeDisk)} free` : ''));"
- ac-4 — MET: The snapshot's resident_bytes is residentCharge plus the pool's ExitingBytes, taken from one Residency reading, and exiting_bytes rides beside it; the roll-up prints 'memory: R of B budget resident, of which E still exiting' and a ui test with 4 GB resident and 4 GB exiting asserts both phrases and the stuck-server count.
  evidence: internal/gateway/control.go:415 — "resident := residentCharge(st.Resident) + exiting"
  evidence: internal/gateway/control.go:423 — "ExitingBytes: exiting,"
  evidence: internal/ui/static/app.js:112 — "if (out.exiting > 0) mem += `, of which ${bytes(out.exiting)} still exiting`;"
  evidence: internal/ui/resources_test.go:39 — "func TestTheRollUpNamesTheExitingPart(t *testing.T) {"
  evidence: internal/ui/resources_test.go:43 — ""4.0 GB of 100.0 GB budget resident", "of which 4.0 GB still exiting", "1 server stuck""
- ac-5 — MET: contextLabel resolves the served window through servedContext, which matches config keys by foldRepoID and falls back to the declared figure when none is set or the set one exceeds it, and prints 'context < declared> declared · < served> served' when served is below declared; modelInfoLine places it on the card. The ui test uses a config keyed 'Org/M' against model 'org/m' and expects 'context 128K declared · 64K served', and a settings test holds the JS resolver to config.Config.ServedContext case by case.
  evidence: internal/ui/static/app.js:60 — "const served = servedContext(config, m.repo_id, n);"
  evidence: internal/ui/static/app.js:62 — "return `context ${tokensLabel(n)} declared · ${tokensLabel(served)} served`;"
  evidence: internal/ui/static/app.js:1121 — "if (foldRepoID(id) === foldRepoID(repoID)) set = models[id].served_context || 0;"
  evidence: internal/ui/static/app.js:79 — "return ctx ? `${bytes(m.bytes)} · ${ctx}` : bytes(m.bytes);"
  evidence: internal/ui/resources_test.go:62 — "`{"models":{"Org/M":{"served_context":65536}}}`); got != "context 128K declared · 64K served""
  evidence: internal/ui/settings_test.go:217 — "func TestThePanelResolvesTheServedWindowAsGoDoes(t *testing.T) {"
- ac-6 — MET: contextLabel returns the empty string for a context_length that is absent, not a number, or not positive, before any served-window lookup, and modelInfoLine then shows the size alone; the ui test asserts '' for context_length 0 (even with a served_context set) and for a model with no context field at all.
  evidence: internal/ui/static/app.js:59 — "if (typeof n !== 'number' || !Number.isFinite(n) || n <= 0) return '';"
  evidence: internal/ui/static/app.js:79 — "return ctx ? `${bytes(m.bytes)} · ${ctx}` : bytes(m.bytes);"
  evidence: internal/ui/resources_test.go:68 — "if got := label(`{"repo_id":"org/m","context_length":0}`, `{"models":{"org/m":{"served_context":65536}}}`); got != "" {"
  evidence: internal/ui/resources_test.go:71 — "if got := label(`{"repo_id":"org/m"}`, `{}`); got != "" {"
- ac-7 — MET_WITH_CONCERNS: The panel's static assets are mounted at '/' on the same mux that Control.Handler wraps whole in loopbackOnly, whose first refusal clause is a non-loopback RemoteAddr, so the markup that holds the block and the /api/state and /api/events it is drawn from are all refused off loopback by construction; the LAN-refusal test proves it for /api/state from [a LAN address]. Concern: the spec promised one assertion that the Models tab's markup is served by the same handler, and no test requests '/' from an off-loopback address — the only '/' cases in the security tests come from 127.0.0.1 — so the criterion's own path rests on the mux structure rather than on a test that would fail if the UI were ever mounted outside the guard.
  evidence: internal/gateway/control.go:109 — "return loopbackOnly(mux)"
  evidence: internal/gateway/control.go:134 — "mux.Handle("/", c.UI)"
  evidence: internal/gateway/control.go:190 — "func loopbackOnly(next http.Handler) http.Handler {"
  evidence: internal/gateway/control.go:248 — "case !isLoopback(r.RemoteAddr):"
  evidence: internal/gateway/control_security_test.go:32 — "func TestControlAPIRejectsNonLoopback(t *testing.T) {"
  evidence: internal/gateway/control_security_test.go:38 — "{"GET", "/api/state", ""},"
  evidence: internal/gateway/control_security_test.go:215 — "req.RemoteAddr = "127.0.0.1:5555""

Gap audit:
- honoured:
  - A block at the head of the Models tab, not a seventh tab, as decided at the 2026-09-09 interview.
    evidence: internal/ui/static/index.html:59 — "< section id="tab-models" class="panel active">"
    evidence: internal/ui/static/index.html:60 — "< p id="resources" class="figures">< /p>"
    evidence: internal/ui/static/app.js:268 — "$('resources').textContent = resourcesSummary(state).text;"
  - No new measurement: every figure is read from the snapshot the panel already polls, and free disk rides that snapshot from the capability package's one reader.
    evidence: internal/ui/static/app.js:87 — "function resourcesSummary(state) {"
    evidence: internal/gateway/control.go:461 — "st.Machine.FreeDisk = capability.Assess(c.App.Paths.Models, c.App.MachineRAM(), budget).FreeDisk"
    evidence: internal/archtest/disk_reader_test.go:31 — "if strings.Contains(string(b), "Statfs") {"
  - The mechanism's own falsifier — shown wrong if the roll-up needs a directory walk or a second reader — is armed as tests rather than left as prose.
    evidence: internal/archtest/disk_reader_test.go:14 — "func TestFreeDiskHasOneReader(t *testing.T) {"
    evidence: internal/ui/resources_test.go:27 — "if got["disk"] != float64(6*1073741824) {"
    evidence: internal/gateway/control_resources_test.go:13 — "func TestTheSnapshotCarriesFreeDiskFromTheOneReader(t *testing.T) {"
  - Each model's card says the window it declares and the window it is set to serve, resolved the fold-aware way the panel already resolves a served window.
    evidence: internal/ui/static/app.js:57 — "function contextLabel(m, config) {"
    evidence: internal/ui/static/app.js:1117 — "function servedContext(config, repoID, declared) {"
    evidence: internal/ui/resources_test.go:56 — "func TestTheCardShowsBothWindowsFoldAwareAndNeverZero(t *testing.T) {"
  - The docs and the changelog say what the block shows, in the memory-budget page as the spec allowed.
    evidence: docs/memory-budget.md:60 — "## What the Models tab adds up"
    evidence: docs/memory-budget.md:62 — "The line at the head of the **Models** tab is the roll-up: how many models"
    evidence: CHANGELOG.md:29 — "- **The Models tab adds up what your Mac is spending on Gropius.** A line at"
- diverged:
  - The spec described the block as 'a figures paragraph and a short table'; delivered is one figures paragraph joined with ' · ' and no table. The same figures are present, in one line.
    evidence: internal/ui/static/index.html:60 — "< p id="resources" class="figures">< /p>"
    evidence: internal/ui/static/app.js:116 — "out.text = parts.join(' · ');"
  - The spec's card line reads 'context: 131,072 declared · 65,536 served' with exact figures; delivered is 'context 128K declared · 64K served' in the card's existing round-down kibitoken abbreviation, and 'max context 128K' rather than 'context: 131,072' when the two windows are equal. Consistent with the card convention the docs describe, but not the spec's wording.
    evidence: internal/ui/static/app.js:49 — "function tokensLabel(n) {"
    evidence: internal/ui/static/app.js:62 — "return `context ${tokensLabel(n)} declared · ${tokensLabel(served)} served`;"
    evidence: internal/ui/static/app.js:64 — "return `max context ${tokensLabel(n)}`;"
  - The free-disk figure is shown only when at least one model is downloaded: the disk part of the line, free space included, sits inside the `downloaded > 0` branch, so an empty install shows no free figure although the snapshot carries one. ac-3's given (the volume has space free) is delivered under the extra condition that a model exists.
    evidence: internal/ui/static/app.js:107 — "if (out.downloaded > 0) {"
    evidence: internal/ui/static/app.js:108 — "parts.push(`${bytes(out.disk)} on disk` + (out.freeDisk ? `, ${bytes(out.freeDisk)} free` : ''));"
  - The spec named two card tests, TestTheCardShowsBothWindowsFoldAware and TestACardWithoutADeclaredWindowShowsNoWindow; delivered as one test covering both cases. Coverage is the same; the record's test names are not.
    evidence: internal/ui/resources_test.go:56 — "func TestTheCardShowsBothWindowsFoldAwareAndNeverZero(t *testing.T) {"
- missing:
  - The spec promised 'one assertion that the Models tab's markup is served by the same handler' for the off-loopback criterion. No test requests '/' (the page that holds the block) from an off-loopback address: the LAN-refusal test enumerates API routes only, and the cross-site navigation test's '/' cases all set RemoteAddr to 127.0.0.1. The guard holds by construction, but the promised test is absent.
    evidence: internal/gateway/control_security_test.go:35 — "dangerous := []struct {"
    evidence: internal/gateway/control_security_test.go:46 — "req.RemoteAddr = "[a LAN address]:5555" // a LAN host"
    evidence: internal/gateway/control_security_test.go:199 — "{"a link to the panel from another site", "GET", "/","
    evidence: internal/gateway/control_security_test.go:215 — "req.RemoteAddr = "127.0.0.1:5555""

Scope-condition dispositions:
- cond-2609100450250935 — survived: The one figure the delivery added to the snapshot is read by a Darwin-only syscall.Statfs inside capability_darwin.go, whose filename constrains it to macOS, and nothing in the delivery introduced a reading that would answer differently elsewhere; the Apple-Silicon half of the condition is consistent with the delivery but was not separately exercised.
  evidence: internal/capability/capability_darwin.go:44 — "func Assess(modelsDir string, totalRAM, budget int64) Machine {"
  evidence: internal/capability/capability_darwin.go:86 — "if err := syscall.Statfs(dir, &st); err != nil {"
- cond-2609100450255120 — narrowed: Loopback-only holds — the UI is mounted on the mux that Handler wraps whole in loopbackOnly, and the LAN-refusal test proves it for the state the block is drawn from — but the inference 'so the figures are the operator's' is wider than the code claims: Handler's own comment says loopback includes this Mac's other user accounts, which reach the panel over 127.0.0.1, so the figures are visible to anyone with an account on the machine, not to the operator alone.
  narrowing: Holds under 'anyone with an account on this Mac', which loopback admits, rather than 'the operator' alone; the figures are this machine's and never the LAN's.
  evidence: internal/gateway/control.go:103 — "loopback means only this machine — including its other user accounts, which"
  evidence: internal/gateway/control.go:109 — "return loopbackOnly(mux)"
  evidence: internal/gateway/control.go:134 — "mux.Handle("/", c.UI)"
  evidence: internal/gateway/control_security_test.go:32 — "func TestControlAPIRejectsNonLoopback(t *testing.T) {"
- cond-2609100450250643 — untested: The roll-up counts and sums whatever the registry lists, and Rescan's comment says it adopts a second account's models, but nothing in the delivered range exercises a shared cache — no test in internal/ui, internal/gateway or internal/archtest constructs models owned by another account or asserts they are counted — so the condition was neither shown to hold nor contradicted.
- cond-2609100450254408 — survived: The panel redraws the block from each /api/events message, which the control plane sends on registry change and on a two-second tick, and free disk is read once per snapshot inside handleState by a comment that says so; no figure in the block is a live counter, and the measured window and usage figures stayed with the probe and the statistics surfaces.
  evidence: internal/ui/static/app.js:153 — "const es = new EventSource('/api/events');"
  evidence: internal/ui/static/app.js:177 — "renderModels();"
  evidence: internal/gateway/control.go:1700 — "tick := time.NewTicker(2 * time.Second)"
  evidence: internal/gateway/control.go:459 — "// Read at the snapshot's cadence through the one reader the search tab"
## Grounds

- pursued: we expect the roll-up to answer the question people ask first when a Mac fills up, without a new tab; shown wrong if operators still open Finder to find out what Gropius holds.

---
id: itd-2610031024247803
slug: a-program-that-finished-with-a-model-can-tell-dessau-to
spec_id: spc-2610031153301961
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890, itd-2609061441241254, itd-2609061441285238, itd-2609211335097114]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610031018046897]
impact: additive
---

# A program can ask Dessau to unload a model it has finished with

## Press Release

A program that has finished with a model can give its memory back.

Bob's test script runs a round of requests against one of Alice's models, then
tells Dessau it is done with it, and the memory is free for the next round
without anyone opening the control panel. A model Alice has pinned, or one
someone has just used, stays where it is, and a device Dessau does not trust is
refused. Alice can turn this off in Settings.

## Why This Matters

Today the only ways to free a model's memory are the panel's Unload button and
an undocumented control-panel route that answers only on this Mac. A program
that tests several models in turn waits for someone to click Unload (it
happened on 2026-10-03, when a test tool had to ask the maintainer to), and on
one Mac a model left loaded after a test blocks the next. Seeded 2026-10-03
from the maintainer's request.

## Mechanism

- We expect models to sit loaded for less time after a test round, because the
  program that used a model can say it is done the moment it is. Shown wrong if
  models stay loaded after batches as long as before.
- We expect nobody else to lose a model they are using, because pinned, busy
  and just-used models are refused. Shown wrong if Carol's chat ever has to wait
  for a reload because a program unloaded her model.

## Scope Conditions

- Trusted callers only: programs on this Mac, holders of the API key, and <!-- cond: cond-2610031153302285 -->
  paired clients; other devices on a network without a key, never.
- Only a loaded model that no one is using, that is not pinned and is not in <!-- cond: cond-2610031153304419 -->
  its protected time after use.
- A trusted program may unload any such model, not only ones it used; Dessau <!-- cond: cond-2610031153300110 -->
  does not track who used what.
- Unloading only: programs cannot load, delete or change models this way. <!-- cond: cond-2610031153308005 -->

## Acceptance Criteria

- **Given** an install without a key, **when** a program on this Mac asks to
  unload a loaded model that is idle and not pinned, **then** it is unloaded and
  the answer says so.
- **Given** an install without a key, **when** another device on the network asks
  to unload a model, **then** it is refused whatever the switch says, the model
  stays loaded, and the answer reveals nothing about what is loaded.
- **Given** an install with an API key, **when** a program on another machine asks
  with the key, **then** it may unload; without the key it is refused. A paired
  client may unload.
- **Given** a web page open in a browser on this Mac, **when** it tries to unload a
  model, **then** it is refused and nothing changes.
- **Given** a model Alice has pinned, **when** a trusted program asks to unload it,
  **then** it is refused and the model stays.
- **Given** a model that is answering a request, loading, or in its protected
  time after use, **when** a trusted program asks to unload it, **then** it is
  refused at once and the request finishes.
- **Given** the switch is off, **when** any program asks to unload through the API,
  **then** it is refused, and the Unload button in the control panel still works.
- **Given** the switch, **when** it is read from Go, the panel or `config.json`,
  **then** all three agree; a `config.json` that does not mention it means on, and
  saving an unrelated setting is never refused over it.
- **Given** a request to delete a model through the API (the way OpenAI's API
  deletes models), **when** any client sends it, **then** no file is deleted.
- **Given** a program unloaded a model, **when** Alice reads the statistics or the
  log, **then** they say a program unloaded it and what kind of caller it was
  (this Mac, an API key, a paired client), and never show the key.
- **Given** the documentation, **when** a script author reads it, **then** it names
  the API unload and the control panel's own unload, who can use each, and when
  each is refused.

## Decisions at the planning interview (2026-10-03)

1. No switch on the control panel's own unload route (it is the panel's Unload
   button too, and a script is indistinguishable from it); the route is
   documented, and who can use it follows the panel-access setting of
   itd-2610031004535845. Only the API route has a switch, on by default.
2. Trusted callers only: exactly the callers the gateway already entitles.
3. Pinned models are refused (the pin promise of itd-2609061441241254 holds).
4. Models in their eviction-grace window are refused.
5. Under itd-2610031004535845's "this account only", the API unload keeps the
   API's own rule; other accounts' trusted programs can still unload.
6. Recorded without a question (one defensible answer): the first
   state-changing verb on `/v1` is settled by adr-2610031153127219:
   its route, access rule and refusals.

Typed links: refines itd-2609061441241254 (pins hold), itd-2609061441285238
(grace holds), itd-2609211335097114 (unload behaviour) and itd-2610031004535845
(the API unload stays under API rules); builds on itd-2609081259493890 (three
surfaces); resolves the documentation half of iss-2610031018046897.

## Open Questions

_None open._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-92f55f59b36b -->
Fidelity review — receipt rcp-92f55f59b36b (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:da5db8a92bc53a92f1b6154442389fd05929e569d5cf05ac4d16b3698e8f9809
Input attestations: diff:8e4d2cc4806865dbfb371b11911225c40b649caa..66069c7172bf51f44d799c7b22a40e81e34da3e0 (PR #151, feat/api-unload)@sha256:2732c316c97a92cb22e40dbd6cb605d380ce8616ddcd5170109d57da09f12690; diff:f13e5f627fe8707ed8dc562e89159f195eddd706..fd608850e9c214d99203442590330d4365088823 (PR #158, feat/api-unload-setting)@sha256:15bdedc67d777a00c0ee93fd5db5d9a8c9de762cd3535d3b1d5d1cc45fd84b3b; repo:integrate/0.10.0 at dd929d8a7dbe0446792423b43594d8b84583561e@-;

Acceptance rollup: MET 9 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: On a keyless install a loopback caller is entitled via fromThisMachine, Release unloads an idle unpinned model, and the 200 answer carries status unloaded; gateway and pool tests both exercise it and pass.
  evidence: internal/gateway/gateway.go:370 — "return g.admittedKeyed(r) || fromThisMachine(r)"
  evidence: internal/gateway/gateway.go:1727 — "writeJSON(w, http.StatusOK, map[string]any{"status": "unloaded", "model": id})"
  evidence: internal/runtime/release_test.go:42 — "if err := p.Release("Org/M", Caller{Kind: "this_mac"}); err != nil {"
  evidence: internal/gateway/unload_test.go:95 — "func TestAProgramOnThisMacMayUnload(t *testing.T) {"
- ac-2 — MET: The entitlement check runs first and returns one fixed 403 before the switch or the body is read; the test shows identical bodies for a loaded and an absent model, no release, and the same refusal with the switch off.
  evidence: internal/gateway/gateway.go:1687 — "if !g.entitled(r) || (isLoopback(r.RemoteAddr) && !sameOriginFetch(r)) {"
  evidence: internal/gateway/gateway.go:1688 — "writeError(w, http.StatusForbidden, "unloading a model is not available to this client")"
  evidence: internal/gateway/unload_test.go:116 — "if b1 != b2 {"
  evidence: internal/gateway/unload_setting_test.go:28 — "if code != http.StatusForbidden || strings.Contains(body, "turned off") {"
- ac-3 — MET: withAuth refuses a missing key with 401 and tags a verified key; pairedOnly admits a paired client as keyed and tags it; tests cover keyless refusal, key-holder success, and a real paired client through TLSHandler.
  evidence: internal/gateway/gateway.go:263 — "r = withCaller(r, runtime.Caller{Kind: stats.CallerAPIKey})"
  evidence: internal/gateway/pairing.go:396 — "r = withCaller(r, runtime.Caller{Kind: stats.CallerPairedClient, Client: client.Name, Fingerprint: shortFingerprint(spki)})"
  evidence: internal/gateway/unload_test.go:131 — "code != http.StatusUnauthorized"
  evidence: internal/gateway/unload_setting_test.go:104 — "gp.TLSHandler(srv.registry).ServeHTTP(w, r)"
- ac-4 — MET: Cross-site and same-site Sec-Fetch-Site, foreign Origin, rebound Host, form posts and text/plain JSON are all refused on keyless and keyed installs, with zero releases asserted; the route also requires application/json.
  evidence: internal/gateway/unload_test.go:152 — "func TestAWebPageOnThisMacCannotUnload(t *testing.T) {"
  evidence: internal/gateway/unload_test.go:177 — "if n := len(pool.releasedIDs()); n != 0 {"
  evidence: internal/gateway/gateway.go:1702 — "writeError(w, http.StatusUnsupportedMediaType, `the body must be JSON, sent as "Content-Type: application/json"`)"
- ac-5 — MET: Release checks the pin under the pool lock and refuses with ErrPinned, which the route maps to 409; the pool test shows a pinned model refused and still resident.
  evidence: internal/runtime/pool.go:2515 — "if p.isPinnedLocked(e.repoID) {"
  evidence: internal/runtime/release_test.go:34 — "if err := p.Release("org/m", Caller{Kind: "this_mac"}); !errors.Is(err, ErrPinned) {"
  evidence: internal/runtime/release_test.go:38 — "if len(p.Resident()) != 1 {"
- ac-6 — MET: Release refuses loading, in-flight (preemptible idle holds included, as they increment inFlight) and in-grace models without stopping anything; tests assert each refusal, a sub-500ms answer for loading, and that the model stays resident.
  evidence: internal/runtime/pool.go:2521 — "return fmt.Errorf("%s is still loading: %w", repoID, ErrLoading)"
  evidence: internal/runtime/pool.go:2523 — "if e.inFlight > 0 {"
  evidence: internal/runtime/pool.go:2526 — "if !p.graceElapsedLocked(e, 0) {"
  evidence: internal/runtime/release_test.go:69 — "if time.Since(start) > 500*time.Millisecond {"
  evidence: internal/runtime/release_test.go:88 — "!errors.Is(err, ErrInGrace)"
  evidence: internal/gateway/gateway.go:1721 — "writeError(w, http.StatusConflict, err.Error())"
- ac-7 — MET: With APIUnloadOff set the route answers 403 to every entitled caller (others get the generic refusal), and a wired control-panel test shows /api/models/unload still unloading with the switch off.
  evidence: internal/gateway/gateway.go:1697 — "if g.cfg().APIUnloadOff {"
  evidence: internal/gateway/unload_setting_test.go:18 — "func TestTheSwitchOffRefusesTheRoute(t *testing.T) {"
  evidence: internal/gateway/unload_setting_test.go:51 — "func TestTheSwitchOffLeavesThePanelsUnloadWorking(t *testing.T) {"
- ac-8 — MET_WITH_CONCERNS: One key api_unload_off (omitempty, zero means on) is read by Go, shown by config show, and filled/posted by the panel; the settings-surface enumeration covers it. Concern: the panel half is held only by a source-regex test, and 'never refused over it' rests on Validate having no rule for the field, with no save-path test naming this key (uneditedFormBody omits it).
  evidence: internal/config/config.go:426 — "APIUnloadOff bool `json:"api_unload_off,omitempty"`"
  evidence: internal/ui/static/app.js:1456 — "$('setAPIUnload').checked = !c.api_unload_off;"
  evidence: internal/ui/static/app.js:2184 — "api_unload_off: !$('setAPIUnload').checked,"
  evidence: internal/gateway/unload_setting_test.go:38 — "func TestAnAbsentSwitchMeansOn(t *testing.T) {"
  evidence: internal/ui/apiunload_test.go:14 — "regexp.MustCompile(`api_unload_off:\s*!\$\('setAPIUnload'\)\.checked`),"
  evidence: internal/lifecycle/testdata/config_show.json:5 — ""api_unload_off": false,"
  evidence: internal/archtest/settings_surface_test.go:99 — "func TestEverySettingHasAPanelControlOrAnExemption(t *testing.T) {"
- ac-9 — MET: No DELETE route exists on /v1 (DELETE requests get 404/405 with nothing released), and an AST archtest holds that the gateway's Pool and Models interfaces carry no removing method and routes() registers no DELETE.
  evidence: internal/gateway/unload_test.go:219 — "func TestNoClientCanDeleteAModelThroughTheAPI(t *testing.T) {"
  evidence: internal/archtest/v1_no_delete_test.go:19 — "func TestNothingReachableFromV1CanDeleteAModel(t *testing.T) {"
  evidence: internal/gateway/gateway.go:205 — "mux.HandleFunc("POST /v1/dessau/unload", g.handleUnload)"
- ac-10 — MET_WITH_CONCERNS: A release is recorded as a removal with reason released and a validated caller kind (unknown kinds dropped, key never carried), and logged as 'model unloaded' reason=released by=< kind>. Concern: it is visible in the statistics store file and the log, not in the control panel's Statistics tab, which summarises evictions only.
  evidence: internal/stats/stats.go:273 — "By string `json:"by,omitempty"`"
  evidence: internal/app/app.go:1106 — "o.rec.Released(repoID, by.Kind)"
  evidence: internal/app/app.go:1112 — "o.log.Info("model unloaded", "model", repoID, "reason", stats.ReasonReleased, "by", by.Kind)"
  evidence: internal/stats/released_test.go:17 — "r.Released("org/a", "sk-a-key-in-the-wrong-place")"
  evidence: internal/gateway/unload_setting_test.go:95 — "if strings.Contains(got[i].Kind+got[i].Client, "bh_secret") {"
  evidence: internal/ui/static/app.js:2430 — "m.evictions ? `evicted ${m.evictions}×` : null,"
- ac-11 — MET: docs/unload-reference.md names both routes in one table, says who can use each, lists every 409/403/401/404/400/415 refusal, the switch and the records; an archtest holds the page to both route names.
  evidence: docs/unload-reference.md:11 — "| `POST /v1/dessau/unload` | A program on this Mac, a program that sends the API key, a paired client — unless the operator turns it off | A model nobody is relying on |"
  evidence: docs/unload-reference.md:12 — "| `POST /api/models/unload` | A program on this Mac, through the control panel's own routes | Any loaded model that is not answering a request |"
  evidence: docs/unload-reference.md:44 — "### When it is refused"
  evidence: internal/archtest/unload_docs_test.go:12 — "func TestTheUnloadPageNamesBothRoutes(t *testing.T) {"

Gap audit:
- honoured:
  - A program can free a model's memory without anyone opening the control panel
    evidence: internal/gateway/gateway.go:1717 — "if err := g.pool.Release(id, callerOf(r)); err != nil {"
  - A pinned or just-used model stays where it is
    evidence: internal/runtime/pool.go:2515 — "if p.isPinnedLocked(e.repoID) {"
    evidence: internal/runtime/pool.go:2526 — "if !p.graceElapsedLocked(e, 0) {"
  - A device Dessau does not trust is refused
    evidence: internal/gateway/gateway.go:1687 — "if !g.entitled(r) || (isLoopback(r.RemoteAddr) && !sameOriginFetch(r)) {"
  - Alice can turn this off in Settings
    evidence: internal/ui/static/index.html:694 — "< legend>Unloading from a program< /legend>"
  - The previously undocumented control-panel unload route is documented
    evidence: docs/unload-reference.md:86 — "## `POST /api/models/unload`"
- diverged:
  - Alice reads who released a model in 'the statistics': delivered in the statistics store file and the server log, not in the control panel's Statistics tab
    evidence: docs/statistics-store-reference.md:106 — "| `by` | On `released` only: the kind of caller that asked the model API to unload the model"
    evidence: internal/ui/static/app.js:2430 — "m.evictions ? `evicted ${m.evictions}×` : null,"
- missing: (none)

Scope-condition dispositions:
- cond-2610031153302285 — survived: The route admits exactly the gateway's existing entitlement (keyed admission, paired client, or fromThisMachine) and refuses a keyless network caller and a browser page.
  evidence: internal/gateway/gateway.go:370 — "return g.admittedKeyed(r) || fromThisMachine(r)"
  evidence: internal/gateway/unload_test.go:109 — "func TestAnotherDeviceOnAKeylessNetworkIsRefusedAndLearnsNothing(t *testing.T) {"
- cond-2610031153304419 — survived: Release refuses not-loaded, pinned, loading, in-flight and in-grace models under the pool lock before stopping anything.
  evidence: internal/runtime/pool.go:2529 — "p.stopEntryByLocked(e, StopReleased, &by)"
  evidence: internal/runtime/release_test.go:15 — "func TestReleaseUnloadsOnlyAModelNobodyIsRelyingOn(t *testing.T) {"
- cond-2610031153300110 — survived: Release decides on the model's state alone; the Caller is passed through only for the record and the pool reads nothing in it, so any trusted caller may release any eligible model.
  evidence: internal/runtime/observer.go:82 — "// passes it on and reads nothing in it."
  evidence: internal/gateway/unload_test.go:97 — "code, body := serveUnload(g.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil))"
- cond-2610031153308005 — survived: The only added /v1 route is the unload POST, whose body is just a model id; no DELETE or load/change verb exists and the archtest holds it.
  evidence: internal/gateway/gateway.go:205 — "mux.HandleFunc("POST /v1/dessau/unload", g.handleUnload)"
  evidence: internal/archtest/v1_no_delete_test.go:19 — "func TestNothingReachableFromV1CanDeleteAModel(t *testing.T) {"
<!-- abcd-review-end receipt=rcp-92f55f59b36b -->

## Grounds

- pursued: programs that test several models in turn have to wait for someone to click Unload, as happened on 2026-10-03; shown wrong if test tools never use it. And on one Mac a model left loaded after a test blocks the next, so freeing it promptly lets more models be tried; shown wrong if test rounds are no faster with it.

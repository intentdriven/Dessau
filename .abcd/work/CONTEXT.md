# Context — orientation for a fresh session

Dessau is a Go menu-bar app that turns one Apple Silicon Mac into a shared
local-inference server: it downloads MLX models from HuggingFace and serves
them over an OpenAI-compatible API to the local network. `cmd/dessau` is the
entry point; the packages under `internal/` are described in `AGENTS.md`,
which also carries the verified build/test commands.

Live status is not recorded here: open work lives in the issue ledger under
`.abcd/work/issues/` (folder membership is the status signal). Durable,
empirically verified design constraints live in
`.abcd/development/decisions/DECISIONS.md`.

## Sharp edges

- **The request's `model` field is a load instruction, not a label.** The
  gateway rewrites the client's friendly model name into the backend's exact
  `--model` path on every proxied request; a name that misses sends mlx-lm to
  HuggingFace to download it. This is the single most important routing rule.
- **Child processes get `HF_HUB_OFFLINE=1` and an existing `HF_HUB_CACHE`.**
  A missing cache directory makes `mlx_lm.server` raise `CacheNotFound` and
  return an empty model list.
- **One account serves.** Dessau keeps everything in the serving account's own
  data root (`~/Library/Application Support/Dessau`, or `DESSAU_ROOT`); other
  accounts use it over the network. There is no machine-wide root
  (adr-2610030906462776, 2026-10-03).
- **Firewall:** a locally built, ad-hoc-signed binary run from a new path is
  silently blocked for LAN traffic (loopback still works) — re-run
  `make allow-firewall`. The Makefile pins a stable codesign identifier for
  exactly this reason.
- **The serving Mac must never sleep** — a sleeping Mac does not wake for
  network traffic, and looks "down" to remote clients.
- **`abcd identity` cannot see the landing page until it is rendered.** The
  `landing-hero` surface names `site/Dessau/index.html`, which is a build
  artefact and untracked, so on a plain checkout the surface reports *absent*
  and the check stays green whatever the page says. Run `make site` first to
  make it a real check. What holds the page to the identity block on every run
  is `internal/sitetest`, which renders the page and applies that surface's own
  patterns to the output.
- `make test` always runs with the race detector; treat `-race` findings as
  failures, not noise.

## Handoff — runs of 2026-10-03 and 2026-10-04

What the two autonomous runs built, what is left, and what the next release
needs. One step or one issue per PR; every trust-boundary PR had an
independent adversarial review in a separate agent, recorded in its body.

### Before the next release: the issues that matter

Chosen from the open ledger on 2026-10-04 as the ones a release should not
ship without. Everything else open is minor, Swift-client, or process.

1. **iss-2610031758029994 — one empty prompt freezes a model for everyone**
   (major, reproduced on the Mac 2026-10-03). Fixed and merged in #196
   (adr-2610040749545010); nothing left to do but ship it.
2. **iss-2610040752568866 — a frozen model server goes unnoticed: resolved
   2026-10-04.** Measured on the Mac: `/health` answers 503 in that state and
   the health watch (1ff92cc9) replaces the server within one tick. Nothing
   to build.
3. **iss-2610040805353721 — an array-of-strings prompt reaches the
   generation thread** (security). Reproduced on the Mac 2026-10-04: `["hi"]`
   and `[""]` freeze the server. **Decided:** the gateway refuses a
   `/v1/completions` prompt that is not a JSON string, with a 400, under a new
   ADR that partly supersedes adr-2610040749545010 (reading the JSON kind and
   nothing else). Build it test-first.
4. **iss-2610032306154631 — a bridged history can start with an assistant
   turn** (minor bug, user-visible). Gemma- and Mistral-style templates raise
   on it, and the channel's next message fails. Small, local fix in
   `internal/bridge/discord/conversation.go`: drop leading assistant turns
   after every trim. The bridge is a trust boundary in practice (network
   input), so give it the review.
5. **iss-2610032241098944 — the self-test unloads a model pinned during its
   run** (minor bug). The pin promise #184 made for the context probe does
   not yet hold for the self-test. Small: mirror the probe adapter's
   `ErrPinned` refusal in `selfTestServer.Unload`.
6. **iss-2610032241096901 — a resumed context probe can save a window above
   the served window** (minor bug). Stored data that is wrong and stamped as
   current. Clamp the bounds on resume, or drop them when the provenance
   differs.

Worth doing if time allows, not blocking: iss-2610032231045098 (a missed
tool-call probe, microsecond window), iss-2609190254516275 (no connection
ceiling on either listener; needs a design), iss-2609211754253878 (the
readiness wait cannot tell a silent child from a slow one; related to item
2, and its answer may fall out of that measurement).

Release housekeeping for the session that cuts it: the `[Unreleased]`
CHANGELOG carries `impact: breaking` entries (the sampling refusals of #173,
the empty-prompt refusal); move itd-2610030857275099 and
itd-2610031024247803 to `shipped/` once their specs' fidelity reviews run
(`abcd spec close` wants them); never call the release "signed".

### Intents

- **itd-2610030857275099 — update checks.** All four steps landed (#149,
  #152, #162, #165). Criterion 9 waits on itd-2610030656210408 step 2. Spec
  not closed: docs-fidelity review owed.
- **itd-2610030932551549 — builds of one model.** Step 1 landed (#161).
  Step 2, the panel's grouping, not started.
- **itd-2610031024247803 — a program unloads a model.** Both steps landed
  (#151, #158). Spec not closed: fidelity review owed.
- **itd-2610030656210408 — typed decisions.** Step 1 (mlx-lm 0.32.0) landed
  (#163). Step 2 needs the reviewed-build manifest recorded on a machine
  that reaches the Hub. Step 3 needs the golden fixtures. Step 4 not started.
- **itd-2610031004535845 — panel access for this account.** Step 1 landed
  (#153). Step 2 waits on the browser spike, and carries two review findings:
  run the lookup only for loopback peers, only when the panel is narrowed,
  once per connection; and note scoped IPv6 link-local addresses.
- **itd-2610030932556747 — Swift client.** Not attempted: no Swift toolchain
  in the build container.

All are planned and reviewed; none was started in the 2026-10-04 run.

### What the 2026-10-04 run landed

Every one an issue fix or a record, merged through the queue: #175 (the
rescan's check for a stale aside), #176 (the maintainer's decisions of
2026-10-03), #177 (unload cancels a measurement), #178 and #179 (the
transcript switch and the docs that describe it), #180 and #181 (records and
small corrections), #182 (the folded merge setting), #183 (a resident-only
acquire never waits on a reload), #184 (the context probe honours pins),
#185 (no platform identifiers in statistics, held by a test), #186 (the
read bound is one figure), #187 to #189 (the Discord bridge: restart after a
re-save, typing at admission, conversation bounded in bytes), #190 (CI runs
on pushes to main only), #191 (tests cannot resolve the real home folder),
#192 (the staging-name test no longer flakes), #194 (the Mac reproduction's
records), #195 (a failed update says why before the download ends — a race
that failed #194's CI), #196 (the empty-prompt refusal).

### Hand steps owed on a Mac

- Done 2026-10-04: the measurement for iss-2610040752568866 (resolved:
  `/health` answers 503 and the health watch catches it), the reproduction
  for iss-2610040805353721 (array prompts freeze the server; the maintainer
  decided the refusal, see DECISIONS), the Clef golden fixtures (in
  `internal/runtime/testdata/clef/`, PR 198), and the Swift client build
  (`client/build.sh` builds against the macOS 27 SDK; the four
  `client/tests` scripts pass, 46 checks).
- The panel's browser spike (itd-2610031004535845) before step 2 starts.
- The Swift client's open bugs (iss-2609200815308397, appearance switching,
  is major) — checked by hand in the built app.
- The six live Discord checks (iss-2609190242198542) before the bridge is
  called done.
- Installing a release in the serving account.

### Said plainly

- **Branches are not deleted on GitHub from a cloud session.** The proxy
  answers a deletion with success and leaves the branch; a local session
  deletes merged branches, each checked against `main` first. All of the
  2026-10-04 run's branches are deleted.
- **PR bodies are clean.** The tooling appends a footer with a session link
  on create; each body was rewritten through the GitHub API afterwards and
  re-read.
- **Commit trailers read `Assisted-by: Claude` with no model name**, because
  this environment forbids a model identifier in anything pushed.
- **A PR GitHub calls "dirty" may be stale.** #191 sat armed for five hours
  marked conflicting against a `main` it already contained; a merge of the
  current `main` (with auto-merge off while pushing) cleared it.

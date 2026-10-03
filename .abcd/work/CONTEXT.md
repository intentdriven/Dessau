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

## Handoff — autonomous run of 2026-10-03

What the run of 2026-10-03 built, what it left, and what it needs from the
maintainer. One step or one issue per PR; every trust-boundary PR had an
independent adversarial review in a separate agent, recorded in its body.

### Intents

- **itd-2610030857275099 — update checks (spc-2610030929021692).** All four
  steps landed: 1 #149, 2 #152, 3 #162, 4 #165. Criterion 9 (decision models
  wait for a reviewed build) rests on the reviewed-build seam that
  itd-2610030656210408 step 2 fills; until then no decision model is offered
  an update. The spec is not closed: `abcd spec close` wants a docs-fidelity
  review, which was not run.
- **itd-2610030932551549 — builds of one model (spc-2610030950480763).** Step
  1 landed in #161 (carried first by #150 and #157, each replaced after a
  CHANGELOG conflict once armed). Step 2, the panel's grouping, was **not
  started**: the run stopped building to stay inside the budget.
- **itd-2610031024247803 — a program unloads a model
  (spc-2610031153301961).** Both steps landed: 1 #151, 2 #158. Departure from
  the spec, recorded in DECISIONS: the statistics keep only the kind of
  caller; a paired client's name goes to the log alone, because the
  statistics pages promise no record says who. Spec not closed (fidelity
  review owed).
- **itd-2610030656210408 — typed decisions (spc-2610030846273729).** Step 1,
  the runtime upgrade to mlx-lm 0.32.0 with the signed-off set, is #163.
  Re-verifying 0.32.0 found that it accepts `max_tokens: 0` and then kills the
  batched generation thread for every client; the gateway now refuses that,
  and a `stop` that is not text, which kills it the same way (0.31.3 too).
  Step 2 is handed back (below). Step 3 needs the golden fixtures. Step 4 not
  started.
- **itd-2610031004535845 — panel access for this account
  (spc-2610031016319710).** Step 1, the kernel lookup of a connection's
  account, landed in #153. Two review findings are carried to step 2: run the
  lookup only for loopback peers, only when the panel is narrowed, once per
  connection; and note scoped IPv6 link-local addresses.
- **itd-2610030932556747 — Swift client.** Not attempted: no Swift toolchain
  in the build container.

### Issues

Resolved: iss-2610030913177383 (#152), iss-2610031239271873 (#152),
iss-2610030822065191 (#154), iss-2610031018231170 (#160),
iss-2610031010360026 (#156), iss-2610030913179523 and iss-2610031239271799
(#162), iss-2610030913170591 (#164). The docs half of iss-2610031018046897
landed in #158; its "this account only" half waits on itd-2610031004535845
step 2.

Captured during the run, open: iss-2610031317470004 (update abandoned under
steady traffic), iss-2610031317475284 (a re-download's Precheck needs the
runtime), iss-2610031324593822 (a staging link re-planted mid-update),
iss-2610031320392078 (a failed update is silent in the panel),
iss-2610031444343397 (the rest of the batched loop's unguarded calls, and a
`/health` 503 not treated as a crash).

### Handed back, with the question each needs answered

- **itd-2610030656210408 step 2:** the reviewed-build manifest cannot be
  recorded here — huggingface.co is unreachable from the build container.
  *For each decision model to support, which repository and exact commit is
  reviewed? Record the commit and per-file hashes on a machine that reaches
  the Hub.*
- **iss-2610031010371709:** *Should Load be refused (or relabelled) and chat
  requests refused for a model the chat rule marks not chat-capable now, or
  only once step 4 of itd-2610030656210408 adds the decision kind? And may a
  Load-button criterion be added to that intent?*
- **iss-2610031317475284:** *Is a staged version held to the whole launch
  Precheck, which needs the runtime installed, or only to its model-code
  half when the runtime is absent?*
- **iss-2610031317470004:** *Refuse new requests for a model while it is
  being swapped (a runtime change), or keep the staged copy after an
  abandoned swap and retry later?*
- **iss-2610030656102623:** needs the Swift client; no question, only a Mac.

### Hand steps owed on a Mac

- The Clef golden fixtures (itd-2610030656210408 step 3), recorded by running
  the reference server outside Dessau.
- The panel's browser spike (itd-2610031004535845) before step 2 starts.
- The Swift client build (`client/build.sh`).
- Installing a release in the serving account.
- New with #163: provision the 0.32.0 runtime and load a model; with a
  batched model, confirm against the child directly that `max_tokens: 0`
  kills its generation thread
  (`.abcd/development/research/notes/2026-10-03-mlx-lm-0.32.0-reverification.md`).

### Said plainly

- **Branches are not deleted on GitHub.** This environment's proxy answers
  a branch deletion with success and leaves the branch. Merged branches to
  delete: feat/record-downloaded-version, feat/update-check-setting,
  feat/api-unload, feat/panel-peer-lookup, fix/ask-null-body,
  fix/download-race-test, fix/hybrid-pattern-list-2, feat/build-groups-3,
  fix/memory-line-reserved, feat/api-unload-setting, feat/staged-update,
  feat/runtime-upgrade, fix/outbound-host-test, feat/update-panel.
  Replaced and closed, also to delete: feat/build-groups,
  feat/build-groups-rebased, fix/hybrid-pattern-list.
- **Every PR body carries a "Generated by Claude Code" footer.** The tooling
  appends it on every write and it could not be removed; the session links it
  first carried were removed. Strip the footer by hand if it matters.
- **Commit trailers read `Assisted-by: Claude` with no model name**, because
  this environment forbids a model identifier in anything pushed.
- **Armed PRs were replaced, never pushed to.** A PR armed for auto-merge
  conflicts on CHANGELOG as soon as a sibling merges; the run replaced two
  (#150→#157→#161, #155→#160) and then armed one CHANGELOG PR at a time.
- **Two flaky tests were real defects in the tests**: the download-race test
  (fixed in #159) and the staged swap's observer (fixed in #162).
- **Cost.** The run cannot read its own spend; it stopped starting new work
  to finish inside the stated budget.

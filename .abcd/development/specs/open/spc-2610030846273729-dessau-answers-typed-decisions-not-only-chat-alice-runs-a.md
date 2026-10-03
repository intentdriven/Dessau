---
id: spc-2610030846273729
slug: dessau-answers-typed-decisions-not-only-chat-alice-runs-a
intent: itd-2610030656210408
origin: researcher-authored
production_mode: hand-written
---
# Dessau serves decision models through POST /v1/systemone

## Summary

Delivers itd-2610030656210408: Dessau serves Clef-family decision models at
`POST /v1/systemone`, answering typed questions (`noul`, `choice`, `score`)
with a probability per option in one forward pass, from a reviewed copy of the
model's reference implementation running in Dessau's one, upgraded runtime.
Decision models are a model kind of their own: listed and shown as such,
refused on chat endpoints, never driven by idle work, pinned to a reviewed
upstream revision, and charged to the memory budget like any other model.
Designed from the two adversarial reviews of 2026-10-03 and the planning
interview the same day.

## Prerequisites (each is its own record, settled before the build starts)

1. **Dependency sign-off. Settled 2026-10-03: the full set** (decision line of
   that date in `.abcd/work/DECISIONS.md`). mlx-vlm 0.7.x requires `mlx>=0.32.2`; the lock pins
   `mlx==0.32.0`, so the upgrade moves the whole runtime (the reference was
   tested on mlx 0.32.3, mlx-lm 0.32.0, mlx-vlm 0.7.4). The maintainer signs
   off the full transitive list `uv pip compile` produces (mlx-vlm brings
   Pillow, opencv-python, mlx-audio and its audio stack, fastapi, uvicorn and
   more), or a narrower `--no-deps` set proved sufficient by the import chain.
2. **ADR on the reference implementation. Settled 2026-10-03:
   adr-2610030929484599** — Dessau serves from its own rewrite, held to the
   reference by golden fixtures, with attribution in `ACKNOWLEDGEMENTS`.
3. **The model server's private socket** (iss-2610030846581757) has landed: the
   decision server listens through the same Dessau-owned launcher.

## Approach

### Runtime
One runtime, upgraded for every model. The lock is regenerated with hashes; the
provisioning readiness marker is keyed on the lock's hash, not on the mlx-lm
version alone (today `internal/runtime/provision.go` keys it on
`mlxLMVersion`, so adding a package would never reprovision an existing
install). Everything verified against mlx-lm 0.31.3 is re-verified against the
new versions and the evidence recorded: the sampling launch flags
(`TestSamplingFlagsWereVerifiedAgainstThePinnedServer` and the research note),
the 512 `--max-tokens` default and the served-window flag, the `model_file`
loader paths in mlx-lm and mlx-vlm (the refusal mirrors their exact condition),
every request field the server reads (the refused-fields list), and the private
socket launcher's reuse of the server's handler.

### Recognising a decision model
A model is a decision model when its directory holds the joint head
(`joint_head.safetensors` and `joint_head_config.json`) and its files match a
reviewed build in Dessau's embedded manifest. Hub tags are informative only
(`pipeline_tag: zero-shot-classification` also covers ordinary classifiers).
The kind lives in the registry beside the chat verdict's single home
(`registry.Model.CanChat`, held by `TestTheChatVerdictHasOneHome`): a decision
model's `CanChat` is false whatever the operator's chat rule says.

### Reviewed builds
An embedded manifest lists each reviewed build: repository id, the upstream
commit, and the sha256 of every file the server reads other than the weight
shards (config, head config and head weights, tokenizer files, processor and
generation config, chat template), plus the weight index. The hub downloads a
manifest repository at its pinned commit (`resolve/<commit>/…`), never `main`,
and records the commit in the registry. At load, every listed file is checked
against the manifest; a mismatch refuses the load with a reason naming the
build and saying it does not match the reviewed version. A newer upstream
revision is ignored until a Dessau release ships its review. The first
reviewed build is `mlx-community/clef-flash-4bit` (commit 140bf7e037f5…, about
6.2 GB), whose golden fixtures the maintainer is producing on a Mac with mlx
0.32.3, mlx-lm 0.32.0 and mlx-vlm 0.7.4; `mlx-community/clef-4bit` (commit
d004817d…) follows once its own fixtures exist. Each build has its own golden
fixture set.

### The decision server
Dessau's own rewrite (adr-2610030929484599), embedded in the binary and launched
through the same private launcher as every model server, re-implementing only:
request encoding, the prompt layout, the joint schema head, loading the
reviewed weights through mlx-vlm, and the answer computation. Nothing from the
reference or the model repository is imported. Its behaviour:
- caps on the number of questions, options per `choice` (2–16), score levels,
  and text sizes; images refused (a later intent); a state longer than the
  window refused with the excess named; generic messages for internal errors;
  one request at a time;
- no URL fetching, no client keyword arguments to any processor, no download
  fallback, no truncation;
- held to the reference by golden fixtures per reviewed build, generated from
  the reference at that build's pinned upstream version (same top answer,
  |Δp| ≤ 0.001); the fixtures are kept in the repository, the reference code is
  not.
It is launched, refused and reaped exactly as a chat child is, through the same
Precheck (`model_file`, ownership, `config.json`) plus the manifest check.

### Gateway
`POST /v1/systemone` is registered inside `routes()`, so `withAuth` and
`pairedOnly` apply unchanged. The handler validates the body (`model`, `state`,
`questions`, each question's `type`, `instructions`, `criteria`), resolves the
model, and refuses a chat model with a 4xx naming `/v1/chat/completions`; the
chat and completions handlers refuse a decision model with a 4xx naming
`/v1/systemone`. Neither refusal loads anything. The `model` field is rewritten
for the child as chat's is, and the response echoes the client's own string,
never the launch path. Errors use the gateway's error shape.

### Response shape (the reference's, published as Dessau's)
`{"model": <client string>, "answers": {<name>: <answer>}, "usage":
{"input_tokens": n, "output_tokens": 0, "latency_ms": n}}`, probabilities
rounded to four places:
- `noul`: `{"type": "noul", "noul": P(true)}`;
- `choice`: `{"type": "choice", "choice": <argmax, ties to the first in
  criteria order>, "confidence": p, "probabilities": {<option>: p}}`, in
  criteria order;
- `score`: `{"type": "score", "score": <expected level index>, "confidence":
  <max p>, "legend": {"0": <text>…}, "probabilities": {"0": p…}}`.

### Pool, memory and idle work
A decision child has its own readiness check (a fixed SystemOne request), runs
at concurrency 1, and is charged its weights plus the measured prefill peak at
its fixed window (the build's `max_length`). The hub's "fits" verdict and the
pool's load decision read that one figure, so they cannot disagree. The
context probe, the tool-call probe and the self-test select chat models only;
the decision kind makes that structural, and a test proves no chat request
reaches a decision child.

### Surfaces, statistics
`/v1/models` carries `kind: "decision"` and `chat: false`; the control panel
shows the kind; `config.json` has nothing that can make a decision model a
chat model. A decision request is counted in the statistics with its input
tokens and latency. Whether the planned transcript store
(itd-2609091707499248) records it is settled when that store lands.

## Acceptance criteria and what holds each

1. Same answers: an `internal/mlxtest` test runs each reviewed build's golden
   fixture through `/v1/systemone` and compares against the reference
   server's recorded output (same keys and types, same top answer, |Δp| ≤
   0.001); skipped loudly where the model is absent.
2. Marked in the list: a gateway test with the chat rule cleared and set.
3. Wrong model refused: gateway tests both ways, asserting nothing acquired.
4. Too long refused: a decision-server test with a state over the window.
5. Same access rules: gateway tests reusing the chat endpoint's auth and
   pairing matrix for the new route.
6. No chat probing: pool and app tests that no idle job selects a decision
   model, and a fake child that fails the test on any chat request.
7. Reviewed version only: hub tests (download at the pinned commit) and
   runtime tests (manifest mismatch refused with its reason).
8. Fits, said once: a test that the hub's verdict and the pool's charge read
   the same figure.
9. No downloaded code: the existing `model_file` refusal plus an archtest that
   the decision launcher imports nothing from a model directory; a server test
   that images and URL fields are refused and nothing is fetched.
10. Runtime re-checked: the re-verification evidence above, the reprovisioning
    test on the lock hash, and the existing flag and refusal tests passing on
    the new lock.
11. Panel shows it too: a UI test held to the same field Go publishes.

## Steps

1. The runtime upgrade
   - packages: internal/runtime
   - tests: the regenerated hash-pinned lock (mlx-lm 0.32.0, mlx 0.32.3, mlx-vlm 0.7.4, the signed-off set); reprovisioning keyed on the lock's hash; the sampling-flag, served-window, model_file-refusal, refused-field and socket-launcher checks re-verified against the new versions and their evidence recorded; no OTEL exporter configuration reaches a child
   - landed: #163
2. Recognising and pinning decision models
   - packages: internal/registry, internal/hub, internal/capability
   - tests: the recorded upstream commit (shared with spc-2610030929021692 step 1; land it here if that step has not); the embedded reviewed-build manifest; the decision kind beside CanChat; a manifest mismatch refused at load; the fits verdict and the pool's charge read one figure
3. The decision server
   - packages: internal/runtime (the embedded decision server), internal/mlxtest
   - tests: the rewrite against each reviewed build's golden fixtures (same top answer, |Δp| ≤ 0.001), loudly skipped where the model is absent; caps; images and over-long state refused; nothing imported from a model directory
   - hand: the golden fixtures are recorded on an Apple Silicon Mac with the model, by running the reference server outside Dessau (the decision of 2026-10-03), before this step starts
4. The gateway and the pool
   - packages: internal/gateway, internal/runtime, internal/app, internal/ui
   - tests: POST /v1/systemone under withAuth and pairedOnly; refusals both ways naming the endpoint that fits, nothing loaded; the response shape; /v1/models kind and chat:false; the panel shows the kind; readiness by a decision request; idle work never sends a decision child a chat request

## Footprint

- packages: internal/runtime, internal/registry, internal/hub, internal/capability, internal/gateway, internal/app, internal/ui, internal/mlxtest
- tests: the eleven acceptance criteria as listed above; the runtime re-verification evidence

## Out of scope

Images in requests; decision models other than reviewed Clef builds; batching
several requests in one forward pass; transcript recording of decision
requests (with itd-2609091707499248).

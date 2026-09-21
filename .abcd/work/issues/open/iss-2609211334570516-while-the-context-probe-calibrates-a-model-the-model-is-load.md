---
schema_version: 1
id: "iss-2609211334570516"
slug: "while-the-context-probe-calibrates-a-model-the-model-is-load"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "live server v0.9.1 on 2026-09-21, 503 for every chat request"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/contextprobe/probe.go"
---

While the context probe calibrates a model, the model is loading/in_flight and charged the whole memory budget (charge_bytes = budget: 82.46 GB for a 2.2 GB model), so it is not evictable and every real request for another model is refused 503 for the length of the step; the step timeout floor is eleven minutes (defaultStepTimeout in internal/contextprobe/probe.go: at least ten minutes plus a minute's margin), and after a failed step the probe can pick the same model again. Idle work starved real requests on the live server for the whole period the maintainer tried several chat clients. An idle job should yield the budget to a real request (abort the step, release the model) rather than the request yielding to the job.

- 2026-09-21 14:35 (observed by the operator, not inferred): the probe re-picked the same model after the step ended — `loaded_at` moved from 14:26:50 to 14:34:35 with the step still "calibrating at 1024 tokens" — so it loops on the unresponsive model and the budget is never released.

## Correction from the live box's logs (2026-09-21 15:55, `dessau.log` and the child's log)

The mechanism is the LOAD path, not the probe's step timeout. The child
(`mlx_lm server`) raised `ValueError: Model type glm_ocr not supported` in its
generate thread on the first request while its httpd kept answering `/health`;
Dessau waited its load-readiness timeout — "did not become ready within 10m0s"
— then logged `model failed to load`, unloaded it (`reason=load_failed`), and
the probe re-queued it ("the measurement is held … it stays queued") and tried
again eleven minutes later: **32 attempts from 09:36 to 15:38**, six hours,
each holding the loading model charged at ≈ the whole budget so no client
request fit (the clients' 503s took 0–1 ms; 32 of the 35 logged 503s are the
probe's own requests after 10 min). Only the maintainer's Unload at 15:49 made
the probe record the failure (`bound=model`) and stop. So the fixes are: (1) a
load fails fast when the child's own log carries a fatal load error (Dessau
already captures that log per model) instead of waiting the readiness timeout;
(2) a `load_failed` model is not re-queued by an idle job until something about
it changes; (3) the probe's step floor is secondary and stays as filed.

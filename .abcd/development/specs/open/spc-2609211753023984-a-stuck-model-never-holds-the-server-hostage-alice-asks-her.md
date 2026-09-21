---
id: spc-2609211753023984
slug: a-stuck-model-never-holds-the-server-hostage-alice-asks-her
intent: itd-2609211335097114
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A stuck model never holds the server hostage: a real request pre-empts idle work

## Summary

Delivers itd-2609211335097114: a real request is served, not refused and
retried, while an idle job (the context probe, the self-test, the tool-call
probe) holds the memory it needs — including while the job's model is still
loading. Two existing seams are extended: the context probe takes its hold
through the pool with the soft-hold tag the self-test already uses, and the
pool's eviction plan may name a *loading* entry whose only holders are soft.
A pinned model is never freed. The park a pre-empted client pays is bounded
by the pool's drain limit, derived from the child stop timeouts and stated
in the docs; past it the client is refused naming the holder. The maintainer's
seven decisions of 2026-09-21 (on the intent) are the design's fixed points.

## Scope

Packages: `internal/runtime` (the pool: eviction plan, soft-only loading
entries, the drain bound as a published figure), `internal/contextprobe` and
`internal/app` (the probe's hold and its yield path; the probe's own unload
respects pins), `internal/gateway` (the refusal after the bound names the
holder — extending `idleHolder`, no new disclosure), `internal/ui` (the card's
"for how long"), `internal/archtest` (no soft-hold tag reachable from a
gateway request), `docs/context-probe.md`, `docs/self-test.md`, and the page
that documents the memory refusal. Changelog under `[Unreleased]`,
`impact: additive`.

## Approach

1. **The probe's hold is a pool soft hold** (decision 2). Before a step, the
   probe acquires the model through the pool under
   `runtime.WithSoftHold(runtime.WithSource(ctx, probeSource), selftest.YieldFrom(ctx))`
   — the shape `internal/app/selftest.go:48` and `toolprobe.go:53` use — and
   only then drives the gateway path for the step itself, so the measurement
   keeps the gateway's bounds (the 2026-09-11 harness shape stands; its
   "runtime untouched" clause is superseded by decision 2). The gateway's
   own acquisition for the probe's request finds the model resident under the
   probe's hold; the probe's hold is the one the pool may pre-empt. The yield
   signal reaches the probe as it reaches the self-test (`YieldFrom`); the
   probe's `stepYielded` path (`probe.go:329-335`) is unchanged: bounds kept,
   no figure published, resume from `b.lo`.
2. **A loading entry with only soft holders is takeable** (decision 3).
   `evictionPlanLocked` (`pool.go` ~1624) today skips `!isReady(e)`; it now
   admits an entry that is loading AND `softOnlyLocked(e)`, and `preemptLocked`
   tears the child down through the ordinary stop path (SIGTERM, SIGKILL,
   reap). A client's load and a pin are never in the plan (decision 4; the pin
   check precedes the soft-hold clause as today). The abandoned load's charge
   is released with the entry; no measurement is written.
3. **The park is the drain bound** (decision 5). A pre-empting caller waits
   `DrainWait` (= `maxDrainWait` = `stopBound + drainMargin`, `pool.go:797-801`)
   for the memory to come back, then is refused. The figure is exported once
   (`runtime.DrainBound()` or the pool's option, one canonical place) and the
   docs state it from that source; the refusal after the bound extends the
   0.9.3 `idleHolder` text with the bound — to entitled clients only, as the
   2026-09-21 disclosure rule already limits it.
4. **The pin wins** (decision 4). The probe's `unloadWaiting` → `Pool.Unload`
   path gains the pin check the eviction plan has (the bug captured beside
   this spec): a probe never unloads a pinned model on a yield; the client is
   refused naming the holder.
5. **The card says for how long** (criterion 7): `residencyLabel` in
   `internal/ui/static/app.js` reads `selftest.Status.Since` (already on the
   snapshot) and the probe's equivalent, and renders the duration beside the
   job.
6. **Docs** (criterion 8): `docs/context-probe.md` and `docs/self-test.md`
   say a real request pre-empts idle work, the park bound and its source, and
   that a pinned model is never freed; the memory-refusal page carries the new
   sentence.

## How each criterion is held

1. Probe's hold is a pool soft hold — a pool test asserting the probe's
   acquisition is `softOnlyLocked`; an archtest walking the gateway's request
   path for any `WithSoftHold` (none reachable).
2. Loading model torn down, request served — a pool test with a fake launcher
   whose load blocks: a soft-held loading entry, a client acquisition, assert
   the child stopped, the client served, no 503.
3. The park bound — a pool test with a fake child that ignores SIGTERM: the
   client is refused after `DrainWait` with the holder and the bound in the
   error; a docs test holding the documented figure to the exported constant.
4. The pin wins — a pool test (pinned soft-held entry: not in the plan, client
   refused naming the holder) and a probe test (`unloadWaiting` on a pinned
   model does not unload).
5. A yield leaves no partial figure — the existing `stepYielded` and self-test
   yield tests, run through the new pre-emption path (loading and ready).
6. A client's hold is never pre-empted — the existing eviction tests stay
   green; one added: a client-held loading entry is not in the plan.
7. The card — a node test over `residencyLabel` with `Since` set.
8. The docs — the docs-currency hand check on the shipping line.

## Security

`internal/runtime` and `internal/gateway` are trust boundaries. The soft-hold
tag stays a context value no HTTP request can set (archtest, criterion 1).
Tearing down a loading child is the existing stop path; nothing new is
executed. The refusal names the holder only to entitled clients, as 0.9.3
does; the bound is a constant, not a secret. Adversarial security review of
the diff before it is presented.

## Verification

- `make test` green, `gofmt -l .` empty, `go vet ./...` clean; every test
  above watched red before the change and green after.
- Hand checks, recorded on the shipping decision line, on the scratch root
  with the probe on and the idle threshold low: send a chat request while the
  probe is loading its model and see it answered (the log shows the yield,
  the park and the load); pin the model under probe and see the refusal name
  the holder with the pin resident; read the park bound off the docs and off
  the refusal and see them agree; the card's duration.
- The open question (whether about twenty seconds is acceptable as the park,
  or the stop timeouts are tightened for a soft-held child) is answered by
  the design review from measurements on the scratch root, and the answer
  recorded as a departure or a decision line.

## Out of scope

The silent-child readiness residual (its own capture); any change to what
the probe charges; a new setting or a config.json change; pre-empting a
client's hold; the served-window proposal (itd-2609091712142715, held).

## Falsifiers

The mechanism claim on the intent: a torn-down load that leaves the
accounting or a measurement inconsistent, or a park that routinely exceeds
the drain bound. Either degrades the promise to refuse-then-retry and is
recorded as a departure, not hidden.

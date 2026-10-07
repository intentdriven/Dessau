---
schema_version: 1
id: "iss-2610041945030758"
slug: "the-idle-clock-appears-not-to-count-client-requests-that-end"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "tracing an unexplained model load after a peer's run, 2026-10-04, v0.9.3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/contextprobe.go"
remedy: "Read the idle check the self-test and context probe share; if it ignores requests that fail before reaching a model, count every client request toward the idle clock, with a test that a request ending unreachable resets it."
resolution: "The idle clock now counts every client request at the gateway (selftest.Clients), in flight and at its end, so a request that ended unreachable on a model since unloaded holds the idle jobs"
impact: fix
---

The idle clock appears not to count client requests that end unreachable. On 2026-10-04 a client's seven requests to a wedged Bonsai each ended as class unreachable after 600 s, the last at 18:44:05. The idle-time jobs (self-test, context probe) share idle_threshold_sec, which was 3600 here, yet the self-test started at 18:45, about a minute later. The last request that reached a model ended at 17:34, so the clock seems to have counted only those. A client that is retrying against a stuck model is not idle, and an idle job loading models around it adds load at the worst moment. Inferred from /api/stats and /api/state timings; the idle check itself was not read.

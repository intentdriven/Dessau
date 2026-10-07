---
schema_version: 1
id: "iss-2609211754253878"
slug: "the-readiness-wait-cannot-tell-an-idle-but-silent-child-from"
severity: "minor"
category: "bug"
source: "plan-review"
found_during: "planning interview itd-2609211335097114"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/launcher.go"
remedy: "Detect a silent child during readiness with one bounded first completion (a single token, a short timeout) after /health answers, leaving the probe's step floor as the 2026-09-21 decision set it; a child that answers /health but not that completion fails readiness at once instead of after ten minutes, with a runtime test using a fake server that answers /health and never a completion."
---

The readiness wait cannot tell an idle-but-silent child from a slow one: a model server that answers /health but never answers a completion and writes no traceback (no fatal line for the LoadLogger of PR 144 to act on) still costs the full ten-minute readiness timeout, once per process (marked transient). Seen only in principle; the 2026-09-21 incident's child DID write a traceback. The fix would detect the silent case (e.g. a bounded first completion during readiness) and, if it lowers the probe's step floor to do so, REVERSES the 2026-09-21 decision that keeps the floor as the gateway's prefill base — flag for the maintainer, never assert. Routed out of itd-2609211335097114 at its planning interview (decision 6).

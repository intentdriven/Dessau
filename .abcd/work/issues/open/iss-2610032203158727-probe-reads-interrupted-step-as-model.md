---
schema_version: 1
id: "iss-2610032203158727"
slug: "probe-reads-interrupted-step-as-model"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "implementing the maintainer's decision on iss-2610031818057157"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/contextprobe/probe.go"
remedy: "Reproduce first: run the panel Unload against a probe mid-step with the probe logging at debug, and find which path ends the run as incomplete and dequeued; fix only what that shows."
---

Once, an operator's Unload of a model the context probe was measuring left the model unqueued and marked incomplete straight away, rather than paused for a resume, which is the path every later run showed ("context probe paused … a request arrived", then a resume). The first reading of it was that the step's request came back as the gateway's upstream failure before the probe saw its session cancelled, and was misread as the model's own limit; a unit test of exactly that ordering (the answer arriving after the cancel) did not reproduce it, because the cancelled request returns the cancellation, so that mechanism is unconfirmed. If some path does read an interrupted step as the model's, a measurement could come out below the model's real window, which is why it stays recorded. Seen while implementing the maintainer's decision on iss-2610031818057157 on 2026-10-03; with Unload now cancelling the measurement, the bounds such a misread would corrupt are dropped on that path.

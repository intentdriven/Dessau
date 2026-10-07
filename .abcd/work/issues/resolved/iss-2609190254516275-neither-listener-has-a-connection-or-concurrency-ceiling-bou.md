---
schema_version: 1
id: "iss-2609190254516275"
slug: "neither-listener-has-a-connection-or-concurrency-ceiling-bou"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the listener read-bound change"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/dessau/listeners.go"
remedy: "Maintainer's decision 2026-10-07: cap per client address plus a global cap on each listener. Wrap both listeners so one remote address may hold at most a fixed number of open connections and the listener at most a fixed total; refuse (close) a connection over either cap before any read, with tests for both caps and for a loopback client; record the figures chosen and why."
resolution: "Every listener (plain and TLS, loopback and the bound address) is wrapped in one connCap in cmd/dessau/listeners.go: 32 open connections per client address (full address, loopback exempt) and 512 per listener, a connection over either closed on accept before any read, its slot freed once on close."
impact: fix
---

Neither listener has a connection or concurrency ceiling. Bounding the request read converts 'one unauthenticated connection held forever' into 'held about fifteen seconds of headers plus thirty of body, then reconnect', which is a real improvement but is not the end of goroutine and connection exhaustion on /pair: a peer that reconnects in a loop still costs a goroutine and a socket per attempt, and nothing caps how many it may hold at once. iss-2609190226050845 should not be read as closing that.

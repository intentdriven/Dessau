---
schema_version: 1
id: "iss-2610071231371995"
slug: "request-bodies-can-hold-a-large-share-of-this-mac-s-memory"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of the connection caps (iss-2609190254516275), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Trace whether a queued request's body is retained while it waits; then bound the gateway's total in-flight body bytes (a byte budget taken before the read and released after forwarding) or lower maxRequestBody to what the largest served window can need, with a test that a flood of large bodies is refused before memory grows."
---

Request bodies can hold a large share of this Mac's memory at once when no API key is set. The gateway reads each body whole, io.ReadAll over http.MaxBytesReader at maxRequestBody (32 MiB, internal/gateway/gateway.go), after withAuth, which lets every client through on a keyless install. With the listener's 512-connection cap, partial bodies are bounded by bandwidth (about 30 s of throughput in flight, some 3.75 GB on gigabit, plus ReadAll's growth), and a complete body waiting in a model's queue may stay in memory for its wait; whether it does was not traced. Assessed by the security review of the connection caps, which bound it where nothing did before; the exposure itself predates that change.

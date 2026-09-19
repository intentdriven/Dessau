---
schema_version: 1
id: "iss-2609190110117982"
slug: "internal-gateway-pairing-go-s-pairhandler-and-handlerevoke-t"
severity: "critical"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "PairHandler and handleRevoke now Clone the configuration before they touch it, so the running paired set is never written from under the registry that reads it on every handshake and every request, or the panel that iterates it on every poll. Held by TestPairingDoesNotWriteTheRunningPairedSet under -race, watched failing first as 'WARNING: DATA RACE ... Write ... pairInto ... pairing.go:147'."
impact: fix
---

internal/gateway/pairing.go's PairHandler and handleRevoke take config.Config by value from App.Config() and then write to cfg.Clients. The struct is a copy but the map header is not: that map IS the running configuration's. Both handlers therefore mutate the live paired set while internal/pairing.Registry.Lookup reads it on every TLS handshake and every paired request, and Control.snapshot iterates it through ClientIDs on every panel poll. A Go map is not concurrency-safe and this is a fatal error rather than a race report: 'fatal error: concurrent map read and map write' in Registry.Lookup, reproduced from fifty unauthenticated POST /pair requests against a reader loop. MaxClients does not bound it, because re-pairing the same fingerprint skips the ceiling check. Config.Clone already deep-copies the map and is simply not called on the only two paths that write it.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would

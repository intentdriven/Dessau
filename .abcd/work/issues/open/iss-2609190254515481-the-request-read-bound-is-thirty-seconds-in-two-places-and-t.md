---
schema_version: 1
id: "iss-2609190254515481"
slug: "the-request-read-bound-is-thirty-seconds-in-two-places-and-t"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial security review of the listener read-bound change"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/gropius/listeners.go"
---

The request read bound is thirty seconds in two places and the two are untied. internal/gateway/gateway.go's bodyReadTimeout and cmd/gropius/listeners.go's requestReadTimeout are both 30s and describe the same promise, but nothing couples them, so they can drift silently and the prose that calls it one number stops being true without a test failing. The gateway's per-request SetReadDeadline also overwrites the listener's deadline with a later one, so on the only large-body route the effective bound is the gateway's rather than the listener's. A test that reads both sources and holds them equal would make the claim enforced rather than asserted.

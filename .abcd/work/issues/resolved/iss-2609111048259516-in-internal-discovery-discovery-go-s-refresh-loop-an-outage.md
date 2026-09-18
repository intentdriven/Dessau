---
schema_version: 1
id: "iss-2609111048259516"
slug: "in-internal-discovery-discovery-go-s-refresh-loop-an-outage"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "review of the discovery recovery fix, 2026-09-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/discovery/discovery.go"
resolution: "The refresh loop stands an advertisement down rather than withdrawing it: a responder that has already given up by itself is cancelled and left unspent, because nothing was withdrawn and the registration is the one dnssd can serve again on the socket pair it already has. Only a responder still up is withdrawn, which is what puts its goodbye on the wire. Narrowed after review (iss-2609181119343938): the reuse arm cannot follow the call that stands the advertisement down, because that call is reached only when the TXT record changed and a changed record needs a registration of its own; what the unspent registration buys is the tick after a publish that failed, where the text is unchanged and it is served again instead of another being built. Reuse across a text change would take a dnssd that can re-text a registration in place."
impact: fix
---

In internal/discovery/discovery.go's refresh loop, an outage during which the TXT record also changes calls ad.withdraw() on a responder that has already died; withdraw marks the advertisement spent, which forces the default arm to build a fresh dnssd.NewResponder while the predecessor's socket stays open, because dnssd's respond closes its conn only on the cancellation path. Each such tick strands a socket pair. Found in review of the recovery-report fix; the new twenty-serves test churns the TXT record every tick and so drives this path repeatedly, which reads as coverage of the reuse arm when it is the opposite.

## Grounds

- pursued: two tests against the fake responder — one that a registration whose responder died by itself is not spent and is re-served on the same registration id, one that a live responder is withdrawn, exits, and is spent. What would show it wrong: a dnssd release that closes its conn on the early-return path too, which would make the reuse arm unnecessary rather than wrong.

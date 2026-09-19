---
schema_version: 1
id: "iss-2609190225574067"
slug: "internal-gateway-pairing-go-s-refusal-log-line-in-pairedonly"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the pairing audit fixes branch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/pairing/registry.go"
resolution: "internal/pairing/tlsconfig_test.go now stands TLSConfig up on a real listener and holds VerifyConnection directly: an unpaired key is refused at the handshake and never reaches a handler, a handshake with no client certificate is refused, a paired key is admitted, and a revoked key is refused at its next handshake. internal/gateway/pairing_test.go holds the consequence the refusal log line rests on — an unpaired client writes no refusal line at all, because it never gets a request in."
impact: internal
---

internal/gateway/pairing.go's refusal log line in pairedOnly is bounded only because internal/pairing/registry.go's TLSConfig VerifyConnection refuses a peer whose fingerprint is not in the paired set, and nothing ties the two together. No test in internal/pairing exercises VerifyConnection at all: TLSConfig is covered only indirectly, through the gateway's own end-to-end pairing tests. If the handshake check were ever relaxed to a plain RequireAnyClientCert, every LAN peer with a self-signed certificate would be a fresh key for logEvery, which returns true on every new key and only evicts entries older than its interval — an unbounded log writer reached without any credential. The invariant needs a test in internal/pairing that stands up TLSConfig and shows an unpaired key refused at the handshake.

## Grounds

- pursued: we expect the refusal line's rate-limit key to stay bounded because the handshake bounds the key space to the paired set, and a test in each package now fails if either half moves; wrong if a future listener serves the paired routes under a TLS config these tests do not build, in which case the tie is asserted about a config nothing uses

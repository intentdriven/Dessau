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
---

internal/gateway/pairing.go's refusal log line in pairedOnly is bounded only because internal/pairing/registry.go's TLSConfig VerifyConnection refuses a peer whose fingerprint is not in the paired set, and nothing ties the two together. No test in internal/pairing exercises VerifyConnection at all: TLSConfig is covered only indirectly, through the gateway's own end-to-end pairing tests. If the handshake check were ever relaxed to a plain RequireAnyClientCert, every LAN peer with a self-signed certificate would be a fresh key for logEvery, which returns true on every new key and only evicts entries older than its interval — an unbounded log writer reached without any credential. The invariant needs a test in internal/pairing that stands up TLSConfig and shows an unpaired key refused at the handshake.

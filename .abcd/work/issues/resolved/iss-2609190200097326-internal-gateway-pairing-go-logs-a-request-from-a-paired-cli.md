---
schema_version: 1
id: "iss-2609190200097326"
slug: "internal-gateway-pairing-go-logs-a-request-from-a-paired-cli"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "The refusal of a request from a client this server has not paired is now an Info line beside 'client paired' and 'client revoked', rate-limited on the fingerprint; the per-request line stays Debug like every other per-request line the gateway writes, recorded as a decision. TestEveryPairingEventIsLoggedAtTheShippedLevelByNameAndNeverByKey holds all three lines at config.Default().SlogLevel(), each naming the client and the short fingerprint and never the whole key."
impact: fix
---

internal/gateway/pairing.go logs a request from a paired client with log.Debug, and only the detailed log level maps to debug in internal/config/config.go, so under the shipped default log level a request from a paired client produces no line at all. The acceptance criterion asks that a request from a paired client be logged by its pairing name; what ships names the client correctly but is invisible where the operator would look. No test holds any of the three pairing log lines (pair, revoke, per-request) to naming the client by its name and never by its key, so the criterion rests on reading the code.

## Grounds

- pursued: we expect the operator's sparse log to carry pairing EVENTS and not per-request traffic, because a line per request is the whole log once a chat client streams; wrong if an operator needs per-request attribution at the shipped level, which belongs in the request statistics rather than the log.

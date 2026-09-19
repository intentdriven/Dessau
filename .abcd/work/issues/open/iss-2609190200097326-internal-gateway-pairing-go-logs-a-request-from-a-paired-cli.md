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
---

internal/gateway/pairing.go logs a request from a paired client with log.Debug, and only the detailed log level maps to debug in internal/config/config.go, so under the shipped default log level a request from a paired client produces no line at all. The acceptance criterion asks that a request from a paired client be logged by its pairing name; what ships names the client correctly but is invisible where the operator would look. No test holds any of the three pairing log lines (pair, revoke, per-request) to naming the client by its name and never by its key, so the criterion rests on reading the code.

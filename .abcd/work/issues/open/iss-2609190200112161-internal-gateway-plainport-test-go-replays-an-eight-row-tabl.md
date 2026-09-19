---
schema_version: 1
id: "iss-2609190200112161"
slug: "internal-gateway-plainport-test-go-replays-an-eight-row-tabl"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/plainport_test.go"
---

internal/gateway/plainport_test.go replays an eight-row table written for this test rather than the gateway's existing request fixtures, which is what the acceptance criterion names; it excludes Date and Content-Length from the header comparison; and no row drives a successful chat completion, only the models list, health, and completions that are refused. The criterion asks that every status, every header the gateway sets and every response body be identical byte for byte with and without a paired client, and the answer path a real client uses is the row that is missing.

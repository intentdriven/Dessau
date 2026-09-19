---
schema_version: 1
id: "iss-2609190200099532"
slug: "internal-gateway-pairing-go-s-readpairrequest-bounds-the-pai"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
---

internal/gateway/pairing.go's readPairRequest bounds the pairing body with json.NewDecoder(io.LimitReader(r.Body, maxPairBodyBytes)).Decode, and a streaming decoder stops at the first complete JSON value. A pairing request whose first 4 KiB is a well-formed object followed by any amount of further data therefore decodes and is accepted, where the acceptance criterion says a body longer than the endpoint accepts is refused and nothing is written. The read itself is bounded, so there is no resource harm; what is missing is the refusal and any test for it — TestThePairingEndpointRefusesWhatItCannotStore drives the name and the key bounds and no body bound at all. The endpoint is unauthenticated and on the plain listener, so its input rules should be the ones stated.

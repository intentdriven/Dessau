---
schema_version: 1
id: "iss-2609190100210971"
slug: "internal-gateway-pairing-go-s-pairinto-writes-the-200-answer"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "adversarial review of the client-pairing diff before landing"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "pairInto now returns the answer instead of writing it, and PairHandler writes it only after App.SetConfig has stored the pairing. A save that fails answers with a refusal and no certificate. Held by TestAPairingThatCouldNotBeSavedIsNotAnsweredWithACertificate, which asserts the validation step writes nothing."
impact: fix
---

internal/gateway/pairing.go's pairInto writes the 200 answer and the minted leaf into the response itself, and PairHandler only then calls App.SetConfig. When that save fails the handler calls writeError on a response whose header is already written, so the client receives 200 with a usable certificate for a pairing the server did not record. The client then believes it is paired and every request it makes is refused as unpaired, with nothing to explain it. pairInto should return the answer and let the caller write it after the save succeeds.

## Grounds

- pursued: we expect separating validation from the answer to stop a client believing it is paired when the server is not, because the certificate in the answer is the client's only evidence; wrong if some later caller wants the answer before the save for a reason this one cannot see

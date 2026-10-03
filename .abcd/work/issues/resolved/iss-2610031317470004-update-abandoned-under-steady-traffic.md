---
schema_version: 1
id: "iss-2610031317470004"
slug: "update-abandoned-under-steady-traffic"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of spc-2610030929021692 step 3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Have the pool refuse new requests for a model while it is being swapped (a runtime change, with its own security review), or keep the staged copy after an abandoned swap and retry the swap alone."
resolution: "The swap drains the model: the pool refuses new requests for it with ErrUpdating while in-flight ones finish, so an update lands under steady traffic."
impact: fix
resolved_by:
  commit: "afba359"
---

An update is abandoned under steady traffic and its verified copy thrown away. The swap's swapping mark refuses new loads only; a model already in memory keeps admitting requests through the pool, so Pool.Remove keeps answering busy, the swap gives up after two minutes, and the whole staged and verified version is deleted, to be fetched again on the next try. Found by the adversarial review of spc-2610030929021692 step 3 on 2026-10-03.

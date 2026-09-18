---
schema_version: 1
id: "iss-2609181119343938"
slug: "standdown-runs-only-from-the-branch-where-the-txt-record-cha"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "adversarial review of fix/open-captures-2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/discovery/discovery.go"
resolution: "Narrowed the prose and the comment to what the change does: a registration whose responder died by itself is no longer marked spent, so a LATER tick with the text unchanged serves it again instead of building another. Reuse across a text change is not available — Register bakes the text in — and the comment at standDown now says so, as does the CHANGELOG bullet."
impact: internal
---

standDown runs only from the branch where the TXT record changed, so the reuse arm can never follow it and the socket pair is stranded once per change anyway; the resolve note and the CHANGELOG claim more than the change does

## Grounds

- pursued: the reviewer is right that the reuse arm cannot follow standDown on the tick that calls it; what the unspent registration buys is the tick after a failed publish. What would show this wrong: a dnssd that can re-text a registration in place, which would make the real fix small after all.

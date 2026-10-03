---
schema_version: 1
id: "iss-2610030913170591"
slug: "adr-cites-a-missing-egress-test"
severity: "minor"
category: "inconsistency"
source: "plan-review"
found_during: "adversarial reviews of itd-2610030857275099"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs"
remedy: "Write the outbound-host architecture test the ADR names (every hostname a non-test Go file dials is on a fixed allow list), or correct the record in the ADR that next supersedes it."
---

adr-2609201008476813's Consequences cite 'the architecture test that refuses any outbound host other than the ones the' runtime needs, but internal/archtest holds no such test; the only egress test is bridge_egress_test.go, which covers the bridge. A record that names a guard that does not exist reads as enforcement nobody has. Found by both adversarial reviews of itd-2610030857275099 on 2026-10-03.

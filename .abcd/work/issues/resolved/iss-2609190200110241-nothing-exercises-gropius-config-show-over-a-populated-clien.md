---
schema_version: 1
id: "iss-2609190200110241"
slug: "nothing-exercises-gropius-config-show-over-a-populated-clien"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/lifecycle/config.go"
resolution: "TestConfigShowReportsThePairedSetWithItsFingerprints drives config show and config show --json over a configuration with two paired clients, holds both names and both whole fingerprints in the output and the config.json spelling of every key, pins the reported field set of a paired client to exactly name/spki/paired_at, and holds the fingerprint to never being the redaction placeholder."
impact: internal
---

Nothing exercises gropius config show over a populated clients block. The block is reported only because internal/lifecycle/config.go's collectSettings happens to recurse into a map of structs, and the only fixture in internal/lifecycle/testdata/config_show.json has no paired client in it. The scope condition for the terminal surface says config show reports the clients block with the fingerprints shown and no verb that writes one; as it stands that claim is derived from a walk rather than pinned by a test.

## Grounds

- pursued: we expect the terminal surface's claim to rest on a test rather than on a reflection walk that happens to recurse into a map of structs; wrong if a field is added to a paired client and the pinned set is loosened without anyone arguing for printing it.

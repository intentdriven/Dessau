---
schema_version: 1
id: "iss-2609181116083969"
slug: "the-app-intents-metadata-test-greps-client-build-sh-for-the"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest"
---

The App Intents metadata test greps client/build.sh for the processor step instead of holding a built bundle to carrying the metadata, as the spec named, so a processor that silently stopped writing would still pass.

## Triage 2026-09-18

Left open for the maintainer: holding a built bundle needs the Swift toolchain inside a Go test, which CI does not have — how the metadata step gets a real check is an infrastructure decision, not a test to tighten in place.

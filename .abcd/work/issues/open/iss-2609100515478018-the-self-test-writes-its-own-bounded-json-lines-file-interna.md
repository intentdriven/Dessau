---
schema_version: 1
id: "iss-2609100515478018"
slug: "the-self-test-writes-its-own-bounded-json-lines-file-interna"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-2609100457007827 build"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/results.go"
---

The self-test writes its own bounded JSON Lines file (internal/selftest/results.go) beside the statistics store instead of into it, because the store's schema is per-request and its writer is in another lane. When iss-2609091714393599 unifies the size-bounded writers, the self-test's file should move with them, and a run should become a record kind the statistics reference page names.

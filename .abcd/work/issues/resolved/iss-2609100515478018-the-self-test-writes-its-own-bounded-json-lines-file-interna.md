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
resolution: "The self-test's results file is now written by internal/applog's Rotator, the repository's one size-rotating writer, configured to keep a single file — which is exactly what 'started again at the cap' is — so the results are held to the same guarded open as the log and the statistics store. The runs stay in selftest/results.jsonl rather than being folded into the store's files: a run has no class, no client and no served window, and folding it in would put lines in stats-YYYYMMDD-NNN.jsonl that the dashboard, the Usage tab and a person's own jq would have to learn to skip. The file's name and shape are unchanged, so nothing a reader holds moves. docs/statistics-store-reference.md now names the run as a record kind, says why it has a file of its own, and points at the self-test reference for its fields."
impact: internal
---

The self-test writes its own bounded JSON Lines file (internal/selftest/results.go) beside the statistics store instead of into it, because the store's schema is per-request and its writer is in another lane. When iss-2609091714393599 unifies the size-bounded writers, the self-test's file should move with them, and a run should become a record kind the statistics reference page names.

## Grounds

- pursued: we expect a run to be a named record kind written through the one writer, in its own file, because the store's readers are readers of a per-request format and a run is not a request; wrong if a reader turns out to want runs and requests interleaved in one pass, which would mean the kind belongs in the store's files after all.

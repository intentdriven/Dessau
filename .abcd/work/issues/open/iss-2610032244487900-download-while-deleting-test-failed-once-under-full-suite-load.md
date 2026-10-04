---
schema_version: 1
id: "iss-2610032244487900"
slug: "download-while-deleting-test-failed-once-under-full-suite-load"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "gates for the probe-pins fix (iss-2609211754251373), 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/concurrency_test.go"
remedy: "Run it with -race -count=200 beside a CPU-bound load until it fails, keep the message, and root-cause it; a failing test is not a flake until shown to be one."
---

TestADownloadStartedWhileAModelIsBeingDeletedDoesNotResurrectIt failed once in a run of ./internal/app and ./internal/gateway together under -race, on a branch that touches neither download nor delete; it then passed 15 of 15 in isolation with the branch's change and 10 of 10 without it, and in three further runs of both packages together. Its message was not kept. The test's own assertions are a Fatalf on Download or Delete erroring, a registry/directory mismatch, and a ready model whose files are missing, so one of the delete-versus-download orderings it exists to hold may still be open under load.

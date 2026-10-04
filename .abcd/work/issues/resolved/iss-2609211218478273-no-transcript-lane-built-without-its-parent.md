---
schema_version: 1
id: "iss-2609211218478273"
slug: "no-transcript-lane-built-without-its-parent"
severity: "major"
category: "process"
source: "impl-review"
found_during: "ruthless review of the integration diff of itd-2609091715089488"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/dessau/main.go"
deferred_after: "v0.9.3"
deferral_reason: "the no-transcript server half is live and every model truthfully reads 'keeps no transcript' until the parent (itd-2609091707499248) lands, which is the next lane"
resolution: "Maintainer's decision 2026-10-03, MAKE THE DOCS TRUE NOW: the four surfaces describe only what exists — Dessau keeps no transcript — and the panel no longer reads config.transcript; the docs-currency reviewer read them CURRENT against main plus fix/no-transcript-disarms-debug. itd-2609091707499248 stays planned for a separate decision; the release cut no longer waits on this record."
impact: fix
resolved_by:
  commit: "715cc6584677cb1efc9ae9651f636148cd3177c1"
---

feat/2609091715 (itd-2609091715089488, no-transcript models) is built on a base without its parent, itd-2609091707499248: config.Config has no transcript switch, cmd/dessau/main.go wires nothing into gateway.Options.TranscriptOn, and app.js's transcriptState reads a config.transcript key the snapshot does not carry. Until the parent lands, /v1/models publishes recording:false for every model, the panel's cards and Dessau Chat's picker read 'keeps no transcript' for every model, and the switch sentences on docs/transcript.md, docs/models-list.md, the panel's Transcript hint and the changelog entry describe a control that is not in the tree. That is the truth of this tree — nothing is recorded — but it is not the feature the record describes, and it is the wrong landing order: the 2026-09-20 decision has the parent land first. Resolved when the parent's integration wires cfg.Transcript into TranscriptOn and the panel snapshot carries it, and the docs-currency reviewer reads the four pages against that tree. Open, this record holds the release cut.

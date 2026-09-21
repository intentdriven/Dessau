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
---

feat/2609091715 (itd-2609091715089488, no-transcript models) is built on a base without its parent, itd-2609091707499248: config.Config has no transcript switch, cmd/dessau/main.go wires nothing into gateway.Options.TranscriptOn, and app.js's transcriptState reads a config.transcript key the snapshot does not carry. Until the parent lands, /v1/models publishes recording:false for every model, the panel's cards and Dessau Chat's picker read 'keeps no transcript' for every model, and the switch sentences on docs/transcript.md, docs/models-list.md, the panel's Transcript hint and the changelog entry describe a control that is not in the tree. That is the truth of this tree — nothing is recorded — but it is not the feature the record describes, and it is the wrong landing order: the 2026-09-20 decision has the parent land first. Resolved when the parent's integration wires cfg.Transcript into TranscriptOn and the panel snapshot carries it, and the docs-currency reviewer reads the four pages against that tree. Open, this record holds the release cut.

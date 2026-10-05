---
schema_version: 1
id: "iss-2610042101435880"
slug: "in-the-shipped-binary-no-decision-model-is-ever-marked"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fidelity review rcp-c60839654fc3 of itd-2610030857275099, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Wire the reviewed-build manifest from itd-2610030656210408 step 2 into app.New in cmd/dessau, so decision models are marked 'awaiting review' with no Update until a release ships their reviewed build; add a test on the binary's own wiring."
---

In the shipped binary no decision model is ever marked 'awaiting review': criterion 9 of itd-2610030857275099 is not met. The update check has a hook for decision models (internal/app/app.go around line 242: 'Nil means no model is a decision model'), but cmd/dessau/main.go builds the app without a list of reviewed builds, and none ships. So a decision model with a newer version is marked 'available', with Update offered, and no Dessau release can make Update appear by shipping a reviewed version. The waiting behaviour exists only in tests that supply the list by hand. It depends on itd-2610030656210408 step 2 (the embedded reviewed-build manifest). Today decision models are not served as decision models, so updating one changes no served behaviour.

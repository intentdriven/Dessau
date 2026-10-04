---
schema_version: 1
id: "iss-2610040756460648"
slug: "update-failure-recorded-after-the-download-settles"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "the linux CI run of PR 194, where TestAnUpdateTheRequestsOutlastSaysSo failed on a records-only change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Record the failure inside finishDownload's callback, under the same hold that clears the download, as the other branches record their state."
resolution: "The failure is written inside finishDownload's hold, so the download never ends before the model says why."
impact: fix
resolved_by:
  commit: "51f00cb52fe01cc9f09c949075ac37514dcc354b"
---

A failed staged update records why after it stops counting as a download. The staged branch of the download job calls finishDownload(dl, nil), which removes the model from Downloading, and only then calls Registry.SetUpdateFailure. A reader that waits for the download to finish can read the model in between and see no failure: TestAnUpdateTheRequestsOutlastSaysSo failed that way on PR 194's linux run (UpdateFailed = "", want "busy"), and the panel can poll into the same gap and draw a card that says nothing about a failed update until its next refresh.

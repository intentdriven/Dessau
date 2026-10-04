---
schema_version: 1
id: "iss-2610042101436222"
slug: "the-test-that-model-and-file-names-are-shown-safely-in-the"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "fidelity review rcp-c60839654fc3 of itd-2610030857275099, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui"
remedy: "Add a UI test that renders an update card for a repository name containing markup and asserts it appears as text; drop the file-name half from the criterion or show file names and test them."
---

The test that model and file names are shown safely in the update panel does not render a name with markup. internal/ui's test only searches the panel source for escapeHtml calls, while the spec planned a UI test with markup in a repository name and a file name. The card also shows no file names, so the file-name half of criterion 11 of itd-2610030857275099 is never exercised.

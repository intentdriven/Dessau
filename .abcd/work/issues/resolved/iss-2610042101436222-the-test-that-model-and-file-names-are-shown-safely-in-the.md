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
remedy: "Maintainer's decision 2026-10-07: show and test file names. The update card lists the files a newer version changes, rendered as text, and a UI test renders a card whose repository name and file names carry markup and asserts both appear as text."
resolution: "The check records the files a newer version changes (first 10 by name plus the count); the card lists them through escapeHtml, held by a rendered UI test with markup in the repository and file names"
impact: additive
---

The test that model and file names are shown safely in the update panel does not render a name with markup. internal/ui's test only searches the panel source for escapeHtml calls, while the spec planned a UI test with markup in a repository name and a file name. The card also shows no file names, so the file-name half of criterion 11 of itd-2610030857275099 is never exercised.

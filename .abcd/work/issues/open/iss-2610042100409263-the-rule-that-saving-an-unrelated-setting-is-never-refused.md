---
schema_version: 1
id: "iss-2610042100409263"
slug: "the-rule-that-saving-an-unrelated-setting-is-never-refused"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "fidelity review rcp-92f55f59b36b of itd-2610031024247803, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/apiunload_test.go"
remedy: "Add api_unload_off to the untouched-save test's form body (uneditedFormBody) and a save test that changes another setting while api_unload_off carries an unusual stored value, so the three-surfaces promise is held by a test."
---

The rule that saving an unrelated setting is never refused over api_unload_off is not held by a test. The fidelity review of itd-2610031024247803 (criterion 8) found that it holds only because the field has no validation rule. No save test names the field, and the save test's form helper uneditedFormBody leaves it out, so a future validation rule on api_unload_off could refuse saves of other settings without any test failing. AGENTS.md requires each cross-surface promise to be armed by a test. The panel side is checked only by a source pattern-match (internal/ui/apiunload_test.go).

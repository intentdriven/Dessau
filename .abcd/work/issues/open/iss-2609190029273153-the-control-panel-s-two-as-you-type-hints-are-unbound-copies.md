---
schema_version: 1
id: "iss-2609190029273153"
slug: "the-control-panel-s-two-as-you-type-hints-are-unbound-copies"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "adversarial review of the control-panel redesign draft itd-2609100519003748"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
---

The control panel's two as-you-type hints are unbound copies of Go rules. internal/ui/static/app.js:1214 (budgetHint) carries the memory-budget warning sentence word-for-word duplicated from internal/app/app.go:715, and app.js:241 (graceWaitHint) restates the grace-versus-max-wait rule and both its defaults (120/300) from internal/config/config.go:1176-1182. Nothing binds either pair: internal/ui/grace_test.go:94 asserts graceWaitHint's behaviour by restating the rule in the test rather than comparing it against config.validateGrace, so both copies can drift together and stay green. TestNoSettingsControlIsNarrowerThanValidate (internal/ui/controls_test.go:39) covers numeric min/max only and reaches neither hint. It should be that each as-you-type hint is derived from, or tested against, the single Go rule it explains, the way controls_test.go binds the markup's bounds to config.Validate -- otherwise every hint added to the panel multiplies the client/server drift AGENTS.md records as having wedged this product three times.

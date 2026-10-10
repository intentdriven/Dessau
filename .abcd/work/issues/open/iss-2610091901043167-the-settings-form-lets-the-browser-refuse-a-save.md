---
schema_version: 1
id: "iss-2610091901043167"
slug: "the-settings-form-lets-the-browser-refuse-a-save"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lab-261009093853-9fed999"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/index.html"
remedy: "Add novalidate to #settingsForm and a test, watched failing at the pin, that the form carries it so the browser never gates Save; the server stays the only judge."
---

The settings form lets the browser refuse a save. #settingsForm carries no novalidate while its number inputs declare min, max and step, so browser constraint validation blocks Save on the page's own judgement: a grace period of 1.5 under step="1" is refused though the server accepts it. TestNoSettingsControlIsNarrowerThanValidate covers min and max but not step or type. PANEL-AC13 and VALIDATE-AC6 forbid page-side gating.

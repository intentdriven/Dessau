---
schema_version: 1
id: "iss-2609181053320233"
slug: "client-readme-md-s-ipad-section-says-an-ipad-of-device-famil"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/README.md"
resolution: "client/README.md's iPad section says it is an iPad app with no iPhone version, in place of the device-family jargon."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

client/README.md's iPad section says 'an iPad of device family 2', build jargon in user-facing prose.

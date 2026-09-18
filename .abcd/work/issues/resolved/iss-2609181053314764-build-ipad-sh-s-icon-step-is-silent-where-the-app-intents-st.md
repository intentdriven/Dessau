---
schema_version: 1
id: "iss-2609181053314764"
slug: "build-ipad-sh-s-icon-step-is-silent-where-the-app-intents-st"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/build-ipad.sh"
resolution: "build-ipad.sh keeps actool's output, fails loudly when actool fails, when Assets.car is absent, or when the icon keys did not merge into Info.plist."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

build-ipad.sh's icon step is silent where the App Intents step is loud: actool's output is discarded and nothing checks that Assets.car and the merged icon keys exist, though the icon is an acceptance criterion.

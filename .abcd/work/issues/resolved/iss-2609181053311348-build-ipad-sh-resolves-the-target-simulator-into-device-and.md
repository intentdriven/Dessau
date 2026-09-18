---
schema_version: 1
id: "iss-2609181053311348"
slug: "build-ipad-sh-resolves-the-target-simulator-into-device-and"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/build-ipad.sh"
resolution: "build-ipad.sh installs, launches, spawns and terminates on the resolved $DEVICE rather than 'booted', and picks the iPad from the iOS 27 runtime section of the listing."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

build-ipad.sh resolves the target simulator into DEVICE and then installs, launches, spawns and terminates on 'booted', so with an iPhone simulator already booted the bundle lands elsewhere and the success line names a device it did not touch; use the resolved device in all four calls, and pick the iPad from the iOS 27 runtime section rather than the first listed.

---
schema_version: 1
id: "iss-2609181213181998"
slug: "the-ipad-build-script-s-install-over-usb-is-conditional-on-i"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/build-ipad.sh"
---

The iPad build script's install over USB is conditional on IPAD_DEVICE and has never been run; without it the script only prints the devicectl command, so the press release's install on Bob's own iPad from the Mac it was built on rests on an unexercised path.

## Deferral 2026-09-19

Waits on the maintainer: the install over USB needs their own iPad and their personal team sign-in; nothing here can exercise it.

---
schema_version: 1
id: "iss-2609190200104841"
slug: "client-gropiuschat-picker-swift-draws-the-pairing-lock-lock"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Picker.swift"
---

client/GropiusChat/Picker.swift draws the pairing lock (lock.shield.fill) only on the stored-server row, and that row is rendered only while model.connected is true; a Bonjour-discovered row for the same paired server carries the API-key glyph and no pairing lock. The press release and the acceptance criterion make the lock beside the server's name the one thing the pairing is observable by, so a paired server that Bob has not connected to yet looks unpaired in the picker.

## Triage 2026-09-19

The fix is a client change an agent can make and pin with an architecture test
over the source, but whether the lock then reads correctly beside a discovered
server is an owed hand check on the maintainer's own Mac with a server on the
network: the client has no test target and nothing in this repository can watch
it draw.

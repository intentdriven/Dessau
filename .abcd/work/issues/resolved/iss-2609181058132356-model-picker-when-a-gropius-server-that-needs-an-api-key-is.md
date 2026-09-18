---
schema_version: 1
id: "iss-2609181058132356"
slug: "model-picker-when-a-gropius-server-that-needs-an-api-key-is"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "the maintainer's manual test of the 0.7.0 client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Picker.swift"
resolution: "The picker presents a key sheet when a picked server answers 401: the explanation names the Keychain, the one-time ask and Settings; Use Key stores the key bound to that server and reconnects. Held by TestChatClientAsksForTheKeyWhereTheServerIsPicked."
impact: fix
---

Model picker: when a Gropius server that needs an API key is found and picked, an overlay (a sheet) should ask for the key right there, explain that it is kept securely in the Mac's Keychain bound to that server, that it is asked only once, and where it can be changed later (Settings); today the picker shows a 401 hint and sends the person to Settings (the maintainer's manual test, 2026-09-18).

## Grounds

- pursued: we expect asking for the key where the server is picked to remove the one detour first-time users hit; wrong if servers that need a key are rarer than the sheet's cost in code

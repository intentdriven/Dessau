---
schema_version: 1
id: "iss-2609190146367987"
slug: "client-gropiuschat-gropiuschat-swift-hard-codes-the-mac-s-ow"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "fixing the same defect in Backends.swift for iss-2609190023129175"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The model picker's filter help now interpolates BuiltInBackend.deviceNoun instead of spelling the Mac out, so the iPad build names the device the person is holding. The device-noun guard test was widened from Backends.swift to GropiusChat.swift, with a named allow-list for the one sentence that means the server's Mac."
impact: fix
---

client/GropiusChat/GropiusChat.swift hard-codes "The Mac's own model is always offered." in the model-picker filter help text, where every other device-naming sentence in the client interpolates BuiltInBackend.deviceNoun. The same source compiles for the iPad, so that sentence names a device the person is not holding. It should read "The \(BuiltInBackend.deviceNoun)'s own model is always offered." The neighbouring sentence at the server-address help ("the Mac's .local name or LAN address") is correct as written -- that Mac is the server, not the device in the person's hands -- so a widened guard has to tell the two apart.

## Grounds

- pursued: we expect the one sentence and the widened guard to close this because the helper already exists and every other device-naming sentence in the client uses it; wrong if a Text() with an interpolation renders differently from a plain literal in SwiftUI, or if the allow-list is widened later to cover a sentence that really is about the person's device.

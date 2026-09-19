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
---

client/GropiusChat/GropiusChat.swift hard-codes "The Mac's own model is always offered." in the model-picker filter help text, where every other device-naming sentence in the client interpolates BuiltInBackend.deviceNoun. The same source compiles for the iPad, so that sentence names a device the person is not holding. It should read "The \(BuiltInBackend.deviceNoun)'s own model is always offered." The neighbouring sentence at the server-address help ("the Mac's .local name or LAN address") is correct as written -- that Mac is the server, not the device in the person's hands -- so a widened guard has to tell the two apart.

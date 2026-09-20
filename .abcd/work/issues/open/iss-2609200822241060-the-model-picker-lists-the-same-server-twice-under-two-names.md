---
schema_version: 1
id: "iss-2609200822241060"
slug: "the-model-picker-lists-the-same-server-twice-under-two-names"
severity: "minor"
category: "bug"
source: "manual-test"
found_during: "maintainer manual test of the chat client, 2026-09-20; decomposition of the model-picker rethink"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Picker.swift"
---

The model picker lists the same server twice under two names: once as the Bonjour row (the advertised instance name with its 'API key required · N models' summary) and once as the stored-server row (the bare .local address the client is already pointed at). The stored-server row is shown whenever the client is connected and no discovered row is expanded, without checking whether the browse has already listed the server it points at, so the picker's two sections offer one server as two choices with different glyphs. The row exists for a server the browse cannot see; when the browse does see it, the discovered row should carry the stored state (connected, paired) and the address row should not appear.

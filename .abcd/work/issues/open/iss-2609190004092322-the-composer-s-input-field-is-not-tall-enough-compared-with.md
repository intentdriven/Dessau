---
schema_version: 1
id: "iss-2609190004092322"
slug: "the-composer-s-input-field-is-not-tall-enough-compared-with"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual-test"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The composer's input field is not tall enough compared with Apple Messages and WhatsApp, and its text starts too far to the left: the placeholder sits almost against the capsule's edge where Messages and WhatsApp leave a clear inset (screenshots 2026-09-18 13:36 WhatsApp; 13:43 the client). Match their field height and horizontal text padding; the send button keeps its circle.

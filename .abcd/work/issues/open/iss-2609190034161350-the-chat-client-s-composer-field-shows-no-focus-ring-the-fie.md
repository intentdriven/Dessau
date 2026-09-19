---
schema_version: 1
id: "iss-2609190034161350"
slug: "the-chat-client-s-composer-field-shows-no-focus-ring-the-fie"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "giving the composer field Messages' height and text inset (iss-2609190004092322)"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Composer.swift"
---

The chat client's composer field shows no focus ring. The field's capsule is now drawn in client/GropiusChat/Composer.swift over a plain TextField, because SwiftUI's bordered capsule offers no way to inset its text; a plain field draws no focus effect, so on macOS there is no longer anything to say the composer has keyboard focus, where the bordered capsule used to show the system's ring. Messages shows a focus indication on its field. It should say it has focus in the system's own way without drawing a ring of the client's own.

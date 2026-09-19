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
resolution: "The composer's field is a capsule of at least ComposerMetrics.fieldMinHeight (34pt) with ComposerMetrics.textInset (12pt) before the first glyph, drawn in the new client/GropiusChat/Composer.swift; it was 24pt tall with about 7pt of inset, both measured offscreen against the shipped code. SwiftUI's bordered capsule offers no way to inset its text — padding, safe-area padding and content margins all move the capsule with the text, and an extra-large control size changes neither — so the capsule is drawn in the system's own material and shape, making Composer.swift the no-styling rule's third named exception, one file wide like the transcript's bubbles. Every metric is a scaled metric, so the text-size setting grows the field; multi-line growth to eight lines is unchanged; the send button keeps its circle, now given the field's height so it reads as centred beside one line and stays at the foot of a field that has grown."
impact: fix
---

The composer's input field is not tall enough compared with Apple Messages and WhatsApp, and its text starts too far to the left: the placeholder sits almost against the capsule's edge where Messages and WhatsApp leave a clear inset (screenshots 2026-09-18 13:36 WhatsApp; 13:43 the client). Match their field height and horizontal text padding; the send button keeps its circle.

## Grounds

- pursued: we expect the field to read like Messages' and WhatsApp's because it now matches their measured height and inset and scales with Dynamic Type; wrong if the drawn capsule diverges from the system's own field in an appearance or a future macOS, or if losing the bordered field's focus ring (iss-2609190034161350) costs more than the inset gains

---
schema_version: 1
id: "iss-2609181200258524"
slug: "the-user-s-bubble-draws-its-text-in-a-fixed-color-white-what"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Bubbles.swift"
resolution: "The bubble's text follows the bubble it sits on: white only where the fill is dark by sRGB luminance, the system's own text colour otherwise — which is what the model's quarter-opacity bubble wants in either appearance. A chosen hex colour still stands, with the Default button as the way back. TestChatClientBubbleTextReadsOnItsBubble holds it."
impact: fix
---

The user's bubble draws its text in a fixed Color.white whatever colour the bubble is, so a light bubble colour - one the person chose, or a light accent - leaves white text on a light fill and the reply is unreadable; the stored hex colours take no account of the appearance either.

## Grounds

- pursued: a light bubble colour no longer hides the text on it, in either appearance. Wrong if a colour near the mid point reads badly whichever text colour is picked.

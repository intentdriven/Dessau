---
schema_version: 1
id: "iss-2609200817465716"
slug: "replace-the-text-size-pop-up-with-the-system-s-own-scale-sel"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "maintainer manual test of the chat client, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
blocked_by: ["iss-2609200817467659"]
---

Replace the Text size pop-up with the system's own scale selector: a row of preview thumbnails with a label under each (Larger Text at one end, Default in the middle, More Space at the other), the chosen one outlined, and a sentence under the row explaining what the whole window does at that setting. This is the control macOS uses for display scaling and for text size in its own preferences, so it is the shape people already know; the current pop-up of five words gives no preview of what each means. Depends on the setting actually taking effect (the bug that the text size does nothing is recorded separately); the control is the second step.

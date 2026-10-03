---
schema_version: 1
id: "iss-2610032210293982"
slug: "unload-cancel-survives-a-409"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031818057157"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/unload-reference.md"
remedy: "Say in docs/unload-reference.md that Unload cancels a measurement of the model even when it answers 409."
---

The panel's Unload cancels a measurement of the model before it tries to unload, so an Unload that ends in 409 (a client's request held the model past the wait, or the run ended between the check and the interrupt) has still cancelled it; docs/unload-reference.md describes only the job being asked to let go.

---
schema_version: 1
id: "iss-2609181200251755"
slug: "the-text-size-and-appearance-architecture-tests-count-dynami"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest/chat_client_composer_test.go"
resolution: "The text-size and appearance tests cut the App body into its two scene bodies — the WindowGroup's up to its commands, and the Settings scene's — and require the modifier in each, through the shared sceneRoots helper, rather than counting it anywhere in the file."
impact: internal
---

The text-size and appearance architecture tests count .dynamicTypeSize( and .preferredColorScheme( anywhere in the client's source, so two of either modifier on one inner view would pass while both scene roots had lost theirs - the promise is that the setting is read at the window's root and at Settings' root.

## Grounds

- pursued: a root that loses its setting fails the test. Wrong if the scene bodies stop being findable by those markers and the helper starts failing on shape rather than on substance.

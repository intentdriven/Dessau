---
schema_version: 1
id: "iss-2609190023129175"
slug: "builtinbackend-s-error-sentences-in-client-gropiuschat-backe"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "reading Backends.swift for the built-in context trim fix"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Backends.swift"
resolution: "Every sentence BuiltInBackend shows now interpolates deviceNoun instead of spelling the Mac out: the eight hard-coded strings in the three words(for:) helpers and in unavailability()'s @unknown default. A new archtest, TestChatClientNamesTheDeviceThePersonIsHolding, reads the string literals out of Backends.swift and fails on any that names the device outside the macOS branch where deviceNoun is declared."
impact: fix
---

BuiltInBackend's error sentences in client/GropiusChat/Backends.swift are split on whether they name the device: unavailability() interpolates deviceNoun ("Mac" or "iPad"), while the three words(for:) helpers hard-code "the Mac's own model" in every sentence. On the iPad build those sentences name a device the person is not holding — the context-overflow, guardrail, refusal, rate-limit, language and busy messages all read "Mac". They should interpolate deviceNoun the way the unavailability sentences do.

## Grounds

- pursued: we expect the iPad build's error sentences to name the iPad because the one source is compiled for both devices and deviceNoun is the only thing that knows which; wrong if a sentence is genuinely about the server Mac rather than the device in the person's hands, which the guard would then reject.

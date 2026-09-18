---
schema_version: 1
id: "iss-2609181116081459"
slug: "the-changelog-states-as-shipped-fact-that-writing-tools-work"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
resolution: "The changelog now says what was verified: the composer and the replies are standard text views, so Writing Tools are in their context menu, and the client's actions are declared as App Intents and written into the bundle at build. Neither sentence claims a system-level check that is still owed."
impact: fix
---

The changelog states as shipped fact that Writing Tools work and that Shortcuts and Spotlight list the client's actions, while both checks are recorded as still owed.

## Grounds

- pursued: the release notes claim only what the source and the build gate prove. Wrong if the owed checks come back and show either surface does not in fact work.

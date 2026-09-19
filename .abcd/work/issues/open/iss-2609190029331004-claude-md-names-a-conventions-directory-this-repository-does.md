---
schema_version: 1
id: "iss-2609190029331004"
slug: "claude-md-names-a-conventions-directory-this-repository-does"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "adversarial review of the control-panel redesign draft itd-2609100519003748"
origin: researcher-authored
production_mode: hand-written
found_at: "CLAUDE.md"
---

CLAUDE.md names a conventions directory this repository does not have. The abcd rule-loader section states that the OPINIONS default domain 'points at the canonical conventions under .abcd/development/principles/ rather than copying them', but no such directory exists anywhere in the repo -- .abcd/development/ holds decisions, intents, specs, procedures, research, release-gate and IDENTITY.md, and nothing else. So the OPINIONS domain, when a prompt recalls it, points a reader at nothing, and any decision to write down a design principle (for example the 'intuitive' stance raised by draft itd-2609100519003748) has nowhere to be written. It should be that either the directory exists with the conventions in it, or CLAUDE.md names the surface that actually carries them.

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
resolution: "The directory .abcd/development/principles/ exists, with one file per OPINIONS rule pointer plus a README index; each file quotes its rule verbatim as abcd rules OPINIONS renders it, gives the reasons this repository's own record already carries for it, and names where it bites here. CLAUDE.md's claim that the domain points at canonical conventions rather than copying them is now true."
impact: internal
---

CLAUDE.md names a conventions directory this repository does not have. The abcd rule-loader section states that the OPINIONS default domain 'points at the canonical conventions under .abcd/development/principles/ rather than copying them', but no such directory exists anywhere in the repo -- .abcd/development/ holds decisions, intents, specs, procedures, research, release-gate and IDENTITY.md, and nothing else. So the OPINIONS domain, when a prompt recalls it, points a reader at nothing, and any decision to write down a design principle (for example the 'intuitive' stance raised by draft itd-2609100519003748) has nowhere to be written. It should be that either the directory exists with the conventions in it, or CLAUDE.md names the surface that actually carries them.

## Grounds

- pursued: we expect every OPINIONS pointer to resolve to a file that explains the rule from this repository's record, because the domain deliberately carries no rationale of its own; wrong if a rule is added or reworded upstream and its pointer dangles again, or if a file here starts inventing policy the record does not carry.

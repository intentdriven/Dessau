---
schema_version: 1
id: "iss-2609190042447070"
slug: "nothing-holds-the-opinions-rule-pointers-and-abcd-developmen"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "creating the principles directory the OPINIONS pointers name (iss-2609190029331004)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/principles/README.md"
---

Nothing holds the OPINIONS rule pointers and .abcd/development/principles/ in sync. Every OPINIONS rule the abcd hook injects ends with 'see .abcd/development/principles/<name>.md', and the directory now carries one file per pointer, but no committed check asserts that each pointer resolves or that each file quotes its rule verbatim. The pointers dangled once already (iss-2609190029331004); an upstream rule added, renamed or reworded in the abcd binary would dangle them again silently, and a file here could drift from the sentence it claims to quote. The repository's own convention is that a cross-surface promise is armed by a test rather than a habit. An architecture test in internal/archtest could walk the directory and compare it against 'abcd rules OPINIONS --json', or, if shelling out to abcd from a test is unwanted, against a committed snapshot of that output.

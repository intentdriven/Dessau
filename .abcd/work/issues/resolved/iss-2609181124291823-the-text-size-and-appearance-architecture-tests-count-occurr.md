---
schema_version: 1
id: "iss-2609181124291823"
slug: "the-text-size-and-appearance-architecture-tests-count-occurr"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest"
resolution: "The text-size and appearance tests now read the WindowGroup's and the Settings scene's bodies through a brace-aware scan (sceneRoots delegates to the package's own balanced helper via a new swiftBlock), so each modifier is held inside the scene it belongs to rather than anywhere in the file; the enum sets are read from the whole enum body by enumCases and compared for exact equality, so a fourth case on a following line fails for TextSize as well as Appearance."
impact: internal
---

The text-size and appearance architecture tests count occurrences of the modifier anywhere in the file rather than holding it at the window and Settings scene roots, so both modifiers inside one scene would still pass; the appearance enum regex also pins only the first case line, so a fourth case on a following line slips past.

## Grounds

- pursued: we expect a modifier moved out of a scene root, or a case added below the first case line, to fail the test because the scan is scoped to the scene body and the case set is compared exactly; wrong if the Swift grows a brace inside a string literal or comment within one of those blocks, which the balanced scan does not skip and would misread.

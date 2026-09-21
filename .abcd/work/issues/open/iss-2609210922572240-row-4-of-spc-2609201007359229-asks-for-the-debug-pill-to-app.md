---
schema_version: 1
id: "iss-2609210922572240"
slug: "row-4-of-spc-2609201007359229-asks-for-the-debug-pill-to-app"
severity: "minor"
category: "process"
source: "review-followup"
found_during: "pilot-2 audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
---

Row 4 of spc-2609201007359229 asks for the debug pill to appear in the web panel without a reload and to move from armed to running on unload; pilot 2 checked it through /api/state on a fresh GET (debug_armed, then resident[].debug_log) and not in the panel's DOM. The served state and the drawn pills are held by tests; the DOM half of the hand check is owed. Found by the fidelity audit (rcp-adb86bbdc558, ac-4 MET_WITH_CONCERNS).

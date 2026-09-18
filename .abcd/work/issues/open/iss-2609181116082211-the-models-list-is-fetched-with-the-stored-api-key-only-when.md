---
schema_version: 1
id: "iss-2609181116082211"
slug: "the-models-list-is-fetched-with-the-stored-api-key-only-when"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The models list is fetched with the stored API key only when the request's origin equals the origin the key was saved for, so a newly found server is asked without it - tighter than the acceptance criterion's with the stored API key if there is one.

## Triage 2026-09-18

Left open for the maintainer: the delivered behaviour is the safer one — a key entered for one server is not sent to a newly found one — so the acceptance criterion is the likelier thing to change, and sending a stored credential more widely is not a call to make in a defect sweep.

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

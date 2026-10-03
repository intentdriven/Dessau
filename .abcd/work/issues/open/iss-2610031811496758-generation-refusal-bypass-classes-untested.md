---
schema_version: 1
id: "iss-2610031811496758"
slug: "generation-refusal-bypass-classes-untested"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/generation_refusal_test.go"
remedy: "Add those cases to the refusal table."
---

The bypass classes the review checked by hand have no tests: a unicode-escaped key (top\\u005fk), a duplicate key, a backslash-escaped backslash before an escape (\\\\\\ud800), a lone high surrogate followed by a plain escape (\\ud800\\u0041), an uppercase escape, and a numeric field written as a string (relayed for the server to refuse).

---
schema_version: 1
id: "iss-2609100515484516"
slug: "the-posture-page-says-what-is-on-and-the-self-test-is-a-thin"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-2609100457007827 build"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
resolution: "the self-test PR adds the Self-test line to postureLines, its table test, and the posture-reference row"
impact: additive
resolved_by:
  intent: "itd-2609100457007827"
---

The posture page says what is on, and the self-test is a thing that can be on: a loop that loads models on its own while the Mac is idle. The page has no line for it. One line reading config.self_test, with the reads list the page's tests hold it to, would make the posture page complete again.

## Grounds

- pursued: we expect the posture page to stay a complete account of what is on, so a loop that loads models on its own is stated there the day it ships; shown wrong if a reader of the page is still surprised by a model loading unasked

---
schema_version: 1
id: "iss-2609190242078205"
slug: "the-bridge-s-acceptance-criterion-ac-1-promises-the-bot-show"
severity: "nitpick"
category: "observation"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
---

The bridge's acceptance criterion ac-1 promises the bot shows typing within two seconds, but the answering path is bounded by two workers and a queue, so a message that arrives while both workers are busy shows nothing at all until one frees. internal/bridge/discord/session.go admits the message to a bounded work queue before anything is posted, and the typing indicator is the answering job's first act rather than the dispatcher's. In a busy channel the two-second promise is therefore not held, and a third asker sees silence rather than typing. Either the typing indicator moves ahead of the queue, so that acceptance is acknowledged immediately, or the criterion is narrowed to say it holds while a worker is free.

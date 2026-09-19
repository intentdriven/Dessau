---
schema_version: 1
id: "iss-2609190242254425"
slug: "nothing-holds-the-bridge-s-promise-that-the-bot-shows-offlin"
severity: "nitpick"
category: "observation"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session_test.go"
---

Nothing holds the bridge's promise that the bot shows offline once the switch is thrown. The seventh acceptance criterion of itd-2609180959397172 says that when the bridge is switched off the connection closes, the bot shows offline, and no message is answered until it is switched on again. internal/bridge/discord/session_test.go establishes the first and the third — the socket closes, the goroutine is waited for, and a message sent afterwards produces no REST call at all — but the middle clause is Discord's own presence, which no fake can answer for and no test here can observe. Either the criterion is narrowed to what this repository can hold, or the presence is checked once by hand against a real bot and recorded.

## Triage 2026-09-19

This one waits on the maintainer's hand: presence is a fact about Discord's own
servers, observable only by looking at a real bot in a real member list after
the switch is thrown. The decision it needs is which way to close it — the look
taken once and recorded, or the criterion narrowed to the socket and the
silence, which is all this repository can hold.

---
schema_version: 1
id: "iss-2609190242198542"
slug: "the-six-live-checks-the-discord-bridge-needs-against-real-di"
severity: "major"
category: "process"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/DECISIONS.md"
deferred_after: "v0.10.0"
deferral_reason: "Re-deferred at the 0.10.0 cut: four of the six live checks passed against v0.9.3 on 2026-10-04 (a real bot connecting, a mention and a direct message answered, a long answer split across messages, the slash commands); the sleep-and-wake resume and the rate-limit check were postponed by the maintainer because the Mac had to stay awake for another release, and run against 0.10.0."
---

The six live checks the Discord bridge needs against real Discord are owed and none has been done: a real bot with a real token connecting at all; a mention in a real channel answered; a long answer cut and continued as Discord renders it; a resume across the Mac actually sleeping and waking; the slash commands appearing in Discord's own UI; and the edit cadence holding against Discord's real rate limits rather than a fake's headers. The shipping decision line at .abcd/work/DECISIONS.md:301 names all six as OWED and never claims them done, which is honest, but nothing in the issue ledger tracks them, so the intent is shipped with six of its acceptance criteria resting on a fake gateway. Six of the eleven criteria are MET_WITH_CONCERNS in the fidelity verdict rcp-91c0608082bd for exactly this reason. Until they are done, no user-facing prose should say the bridge is verified against Discord.

## Triage 2026-09-19

This one waits on the maintainer's hand, wholly. Every item needs something no
agent has: a real Discord application and bot token, a server and a channel the
bot is invited to, a phone or browser to send from, and a Mac actually put to
sleep and woken. No fake answers for any of them, and the repository's own rule
is that Discord is never contacted from a test. What it needs from the
maintainer is the six checks done once, each outcome written into the spec's
Verification and a dated decision line — or a decision that the bridge ships
with them owed, which is what the shipping line already says and which this
record then tracks until it changes.

## Deferral 2026-10-04

Re-deferred at the 0.10.0 cut. On 2026-10-04 the maintainer ran four of the six checks against v0.9.3 with a real bot in a test server: the bot connected; a mention and a direct message were answered; a long answer was cut at paragraphs across several messages; `/model` listed and switched models and `/reset` cleared the channel. The sleep-and-wake resume (a 73-second sleep held the connection, so no resume was exercised) and the rate-limit check were postponed by the maintainer because the Mac had to stay awake for another release's cut. Findings from the session are captured separately. The remaining two checks run against 0.10.0.

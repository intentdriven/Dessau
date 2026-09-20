---
schema_version: 1
id: "iss-2609202200256227"
slug: "a-hard-per-client-rate-limit-off-by-default-as-a-later-switc"
severity: "nitpick"
category: "future-work-seed"
source: "plan-review"
found_during: "planning interview 2026-09-20, the two queue intents"
origin: researcher-authored
production_mode: hand-written
---

A hard per-client rate limit, off by default, as a later switch against a runaway client: a paired client, or the unpaired class, sending more than N requests a minute is refused with 429 rather than queued. Declined for the two queue intents (itd-2609202108185678, itd-2609202108180709): the fair turn already keeps a burst from crowding others out, and a limit is a policy the operator would have to set, not a fairness rule; a seed for a later intent.

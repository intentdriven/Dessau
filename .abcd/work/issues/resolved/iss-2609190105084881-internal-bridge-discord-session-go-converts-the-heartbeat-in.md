---
schema_version: 1
id: "iss-2609190105084881"
slug: "internal-bridge-discord-session-go-converts-the-heartbeat-in"
severity: "critical"
category: "security"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
resolution: "The heartbeat interval is bounded as the integer Hello named, with a floor as well as a ceiling, before it becomes a duration. TestTheHeartbeatIntervalIsBoundedAtBothEnds and TestAHostileHelloDoesNotTakeTheProcessDown hold it; with the ceiling-only bound restored the first reports heartbeatInterval(9223372036854775807) = -1ms and the package panics in Int64N."
impact: fix
---

internal/bridge/discord/session.go converts the heartbeat interval named in the gateway's Hello frame to a duration before bounding it: time.Duration(ms) * time.Millisecond. A figure of about 1e13 or larger overflows int64 nanoseconds and comes out NEGATIVE, which the ceiling check does not catch because it only looks for a value that is too large. The negative duration then reaches math/rand/v2's Int64N in the heartbeat's jitter, which PANICS on a non-positive argument — so a single crafted or corrupted Hello frame takes the whole Gropius process down from a goroutine the operator never started. The mirror of it is a Hello naming 1 millisecond, which passes every check and has the heartbeat goroutine spin on the connection. The figure must be bounded as the integer it arrives as, with a floor as well as a ceiling, before it becomes a duration.

## Grounds

- pursued: we expect the fix to hold because a test watched to fail covers it; wrong if the same class of defect appears at a seam this change did not touch

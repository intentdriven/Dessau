---
schema_version: 1
id: "iss-2609190054030656"
slug: "the-discord-gateway-session-s-last-seen-sequence-number-resu"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "running the full race suite over the new Discord bridge"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
resolution: "The gateway session's last-seen sequence is now held behind a mutex on resumeState, with advance, sequence and clear as its only accessors, so the read loop's write and the heartbeat's read no longer race. go test -race over the package is clean."
impact: internal
---

The Discord gateway session's last-seen sequence number (resumeState.seq in internal/bridge/discord/session.go) is written by the read loop on every dispatch and read by the heartbeat goroutine on every beat, with no synchronisation. go test -race reports a data race between session.readLoop and session.sendHeartbeat. A torn or stale read would send Discord a sequence the session has passed, which on a resume asks for events again or skips them. It needs the sequence held behind a lock or an atomic.

## Grounds

- pursued: we expect a mutex on the one field two goroutines touch to be enough because the sequence is a single int64 read and written on unrelated schedules; wrong if the resume state grows a second field the two goroutines share and the lock is not widened with it

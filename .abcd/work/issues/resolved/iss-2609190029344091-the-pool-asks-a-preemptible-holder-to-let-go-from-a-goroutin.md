---
schema_version: 1
id: "iss-2609190029344091"
slug: "the-pool-asks-a-preemptible-holder-to-let-go-from-a-goroutin"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial read of the soft-hold diff for iss-2609100526194406"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
resolution: "preemptLocked's goroutine now delivers every yield before it writes its log line, so nothing a log sink does stands between the pool marking a hold as asked and the holder being asked."
impact: internal
---

The pool asks a preemptible holder to let go from a goroutine that logs first and calls the holder's yield second (preemptLocked in internal/runtime/pool.go). A log sink that blocks therefore delays, and in the limit prevents, a yield the pool has already marked as asked (entry.yielding), so a client parked for that memory waits out its whole bound and takes the refusal the soft hold exists to avoid. The yields must be delivered before anything else the goroutine does.

## Grounds

- pursued: we expect a holder marked as asked to have been asked, because the two now happen in that order on the same goroutine; wrong if a yield is ever delivered from somewhere that can block first.

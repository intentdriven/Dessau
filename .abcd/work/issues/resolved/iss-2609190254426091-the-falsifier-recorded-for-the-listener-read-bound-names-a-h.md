---
schema_version: 1
id: "iss-2609190254426091"
slug: "the-falsifier-recorded-for-the-listener-read-bound-names-a-h"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "adversarial security review of the listener read-bound change"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/gropius/listeners.go"
resolution: "The falsifier is corrected in both places. The comment on cmd/gropius/listeners.go and the dated line in .abcd/work/DECISIONS.md now name the shape that would bite — a route that reads part of its body, answers for longer than the bound and then reads the rest, so nothing has lifted the deadline — and say plainly that a handler streaming over a simply unread body is not that shape."
impact: internal
---

The falsifier recorded for the listener read bound names a hazard that does not exist. Both the dated line in .abcd/work/DECISIONS.md and the comment on cmd/gropius/listeners.go say the bound would be shown wrong by a handler that streams a long answer while its request body is still unread. That shape is harmless: an armed read deadline reaches the read path only, and with no background read started there is nothing to time out and nothing to cancel the request context. The genuine residual is narrower and is recorded nowhere: a handler that reads PART of its body, streams past the bound, and then reads more. No such handler exists today, but that is the invariant a future change would break, and the record points a reviewer at the wrong thing.

## Grounds

- pursued: we expect the narrower falsifier to be the true one because an armed read deadline reaches the read path only and no background read exists to time out until the body hits EOF; wrong if a future net/http arms something else on the connection while a handler is writing

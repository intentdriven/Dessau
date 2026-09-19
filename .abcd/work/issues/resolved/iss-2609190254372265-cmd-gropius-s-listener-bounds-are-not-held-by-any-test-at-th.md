---
schema_version: 1
id: "iss-2609190254372265"
slug: "cmd-gropius-s-listener-bounds-are-not-held-by-any-test-at-th"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the listener read-bound change"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/gropius/listeners_test.go"
resolution: "cmd/gropius/listeners_test.go now holds the wiring as well as the behaviour: one test reads main.go and asserts both listeners are constructed through listenerServer and that no http.Server literal is left in the file, and another asserts the bounds on the server listenerServer returns — the read bound set to thirty seconds, the header and idle bounds unchanged, and no WriteTimeout. The stall test also asserts os.ErrDeadlineExceeded rather than any error, so a connection dropped for some other reason no longer passes for a bound."
impact: internal
---

cmd/gropius's listener bounds are not held by any test at the point where removing them matters. cmd/gropius/listeners_test.go exercises listenerServer directly and passes its own read bound, so nothing asserts that main.go's two call sites construct through listenerServer, and nothing asserts the value of requestReadTimeout. Reverting both call sites to the pre-change inline http.Server literal leaves go build, go vet, gofmt and the whole test suite green: the security control can be deleted from production without a single test noticing. cmd/gropius/tlsbind_test.go already has the pattern for this class, reading main.go and asserting the shape of the branch it cares about. The stall test also accepts any non-nil read error rather than os.ErrDeadlineExceeded, which is the branch internal/gateway/gateway.go uses to answer 408 rather than 400.

## Grounds

- pursued: we expect a source-shape assertion to be the thing that fails when a future edit rewrites the call site, because that is what the reverted-main.go run showed and what tlsbind_test.go already does for the pairing branch; wrong if the call sites are reformatted rather than reverted, in which case the test fails on a change that kept the bound

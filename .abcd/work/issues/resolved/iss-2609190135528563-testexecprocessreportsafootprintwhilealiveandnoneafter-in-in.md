---
schema_version: 1
id: "iss-2609190135528563"
slug: "testexecprocessreportsafootprintwhilealiveandnoneafter-in-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "the final race suite on the Discord bridge branch, then reproduced on main"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Every reading the test takes now goes through readWithin, which takes it off the test goroutine and holds it to a minute: a reader that is parked rather than slow ends the test with a line naming the wait instead of the package's timeout and a goroutine dump. The park itself was a real defect in the reader and not the test's doing — Footprint waited on an output pipe it could not close — captured and fixed as iss-2609190208243645."
impact: internal
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime hangs until the test binary's deadline rather than failing, under go test -race. It reproduces on unmodified main at 3f539b7, so it is not a branch's doing — but this Mac was running several worktrees' suites at once when it was seen, so heavy load may be the trigger rather than the defect. The dump shows the test goroutine parked while os/exec's watchCtx goroutine is blocked on a channel send, which is the shape of a child process that is not being reaped. It should be given a deadline of its own so a slow machine produces a failure that names the wait instead of a package-wide timeout.

## Deferral 2026-09-19

Left for the maintainer. It reproduces on unmodified `main`, so it belongs to
no branch, and the machine it was seen on was running several worktrees'
suites at once — deciding whether that is the cause or only the trigger needs
a quiet Mac, which this was not.

## Grounds

- pursued: we expect a hang in one test to be reported by that test, because a package-wide timeout names nothing and costs ten minutes; wrong if a minute turns out to be inside the range of ordinary scheduling delay on a loaded Mac, which would make this a flake of its own.

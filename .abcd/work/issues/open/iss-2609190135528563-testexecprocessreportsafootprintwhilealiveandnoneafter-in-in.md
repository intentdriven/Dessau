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
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime hangs until the test binary's deadline rather than failing, under go test -race. It reproduces on unmodified main at 3f539b7, so it is not a branch's doing — but this Mac was running several worktrees' suites at once when it was seen, so heavy load may be the trigger rather than the defect. The dump shows the test goroutine parked while os/exec's watchCtx goroutine is blocked on a channel send, which is the shape of a child process that is not being reaped. It should be given a deadline of its own so a slow machine produces a failure that names the wait instead of a package-wide timeout.

## Deferral 2026-09-19

Left for the maintainer. It reproduces on unmodified `main`, so it belongs to
no branch, and the machine it was seen on was running several worktrees'
suites at once — deciding whether that is the cause or only the trigger needs
a quiet Mac, which this was not.

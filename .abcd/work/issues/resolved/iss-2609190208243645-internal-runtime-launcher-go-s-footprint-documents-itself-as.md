---
schema_version: 1
id: "iss-2609190208243645"
slug: "internal-runtime-launcher-go-s-footprint-documents-itself-as"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "making TestExecProcessReportsAFootprintWhileAliveAndNoneAfter deterministic under load (iss-2609190135528563)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/launcher.go"
resolution: "Footprint's process listing now runs through sampleProcess, which sets cmd.WaitDelay so the wait for the listing's output ends a second after the listing itself has been given up on; an abandoned listing is no reading rather than an unbounded park. TestAProcessSampleDoesNotWaitOnAnOutputPipeItCannotClose holds it to that with a listing whose grandchild keeps the pipe open."
impact: fix
---

internal/runtime/launcher.go's Footprint documents itself as bounded but is not: it runs the process listing with exec.CommandContext(...).Output(), and Output's Wait goes on waiting for the listing's stdout pipe to close after the context has fired and the listing has been killed. The pipe is not this process's to close — a listing the kernel has not finished with on a Mac under memory pressure does not die when it is signalled, and neither does anything else holding the descriptor — so the reader parks for as long as that lasts. The pool's footprint sampler calls this and the pool's close waits for the sampler, so an unbounded wait here is a Gropius that will not quit on exactly the machine the reader exists to survive. Measured: with a listing whose output pipe is held, the call returns after 20s instead of the 3s footprintTimeout. It should set cmd.WaitDelay so the wait for the output has an end of its own.

## Grounds

- pursued: we expect a footprint reader that always returns inside its own bounds because the pool's close waits on it and a Mac under memory pressure is where this reader is asked to work; wrong if WaitDelay truncates a legitimate slow listing into a wrong figure — it does not, because Output returns ErrWaitDelay and Footprint answers zero, which is its documented answer to a listing that did not arrive.

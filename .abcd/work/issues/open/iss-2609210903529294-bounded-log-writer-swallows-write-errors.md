---
schema_version: 1
id: "iss-2609210903529294"
slug: "bounded-log-writer-swallows-write-errors"
severity: "minor"
category: "observation"
source: "impl-review"
found_during: "itd-2609062346072707 security review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/boundedwriter.go"
---

An armed model server's bounded log writer swallows write errors: a disk-full or I/O error during a run at the model server's debug level is discarded by boundedWriter, which by the spec's decision never returns an error and never blocks (a writer that stopped draining would stall the model server). Nothing surfaces the failure, so an operator can believe a diagnostic run kept its evidence when the file stopped short of the bound. The writer's contract is decided (spc-2609201007359229, the size bound); how the failure is surfaced — once, to Dessau's own log, or on the panel — is not, and it is a trust-boundary change in internal/runtime. Security review finding on itd-2609062346072707, severity low.

---
schema_version: 1
id: "iss-2609091714393599"
slug: "two-size-rotating-file-writers-now-exist-internal-stats-stor"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "implementing spc-2609091703459471 (the server's own log)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/applog/applog.go"
resolution: "Extracted the shared primitive into internal/applog, the canonical home the record named: applog.OpenIn is the one guarded open (os.Root, O_NOFOLLOW, O_NONBLOCK, mode 0600 and an fstat on the handle) and applog.Rotator the plain size-rotating writer, with the statistics store's day-named, two-bound, summary-folding writer sitting on OpenIn rather than on Rotator so that no on-disk name, content or retention figure moves. The drift the record predicted was there and is fixed: neither writer refused a file it had not made, and both now refuse one whose mode is not 0600."
impact: internal
---

Two size-rotating file writers now exist: internal/stats/store.go's storeWriter and internal/applog's rotator. Both append lines to a 0600 file opened O_CREATE|O_WRONLY|O_APPEND|O_NONBLOCK|O_NOFOLLOW through an os.Root on their own directory, both start a new file when the next line would take the current one past a byte cap, and both remove the oldest file to stay under a bound. They were not unified when the second arrived because the store's rotation is not a size-rotating file writer with a store on top: its file names carry a UTC day and a counter, its pruning enforces two bounds (a months horizon and a byte ceiling), and what it drops is folded into a summary file that itself counts toward the ceiling. Extracting the shared primitive means separating the rotation from the day-numbering, the fold and the retention arithmetic inside the package that owns this repository's durable data format, which is a larger and riskier change than the log file that prompted it. The canonical home, when it is extracted, is the smaller of the two: internal/applog's rotator is the plain primitive and the store's writer is the specialisation. What would show this wrong is the two drifting on the discipline they share — one gaining a check on the opened handle, or a mode, or a symlink refusal that the other does not.

## Grounds

- pursued: we expect the four rules the two writers share (the open-handle check, the mode, the symlink refusal and the O_NONBLOCK discipline) to exist once and be held by a test over both, because a copied discipline is invisible when it is right and silent when it is wrong; wrong if a third caller needs a directory rule the other two must not have, which would mean the split between the file and its directory was drawn in the wrong place.

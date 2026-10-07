---
schema_version: 1
id: "iss-2610032231045098"
slug: "a-tool-call-probe-requeue-can-be-lost-between-errgone-and-dequeue"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the resident-only fix (iss-2609202010357676)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/toolprobe/probe.go"
remedy: "Keep a requeue-requested flag (or a generation counter) on the queue head so an Enqueue that lands while the head is being served survives the dequeue."
resolution: "the queue head keeps an again flag that an Enqueue after the head's last look sets, so dequeue moves it to the back instead of dropping it"
impact: fix
---

A tool-call probe re-queue can be lost: after a resident-only Acquire returns ErrGone, the other caller's load can finish and modelLoaded can call Enqueue before drain reaches dequeue; Enqueue skips the model because it is still queued, and dequeue then removes it, so the model goes unprobed until its next load. The window is microseconds against a load of seconds and the worst outcome is a missed probe, never a wrong verdict.

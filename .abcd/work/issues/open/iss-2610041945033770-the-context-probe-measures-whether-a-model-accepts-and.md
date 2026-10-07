---
schema_version: 1
id: "iss-2610041945033770"
slug: "the-context-probe-measures-whether-a-model-accepts-and"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "maintainer's request after the 2026-10-04 stall investigation; state-of-the-art check: RULER arXiv:2404.06654, HELMET"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/contextprobe.go"
remedy: "Extend the context probe, rather than adding a new tool, with a usability check at its measured size: RULER-style multi-needle retrieval at several depths (arXiv:2404.06654), scored and recorded beside measured_context, so the window Dessau publishes is one the model retrieves from correctly, not only one it accepts."
related_intents: [itd-2610071208598233]
---

The context probe measures whether a model accepts and answers a prompt of a given size, not whether it can use that much context. Its measured window says a prompt that long loaded, prefilled and got an answer before a bound stopped the probe. It does not show that the model retrieves or reasons over content near the start of that window. Published long-context evaluations show the two differ widely. RULER (arXiv:2404.06654) found models that score near-perfectly on vanilla needle-in-a-haystack retrieval yet degrade sharply as context grows, with few holding up at 32K despite claiming more. HELMET (2024, updated 2025) reaches the same conclusion with broader tasks. A served window set from the probe can therefore advertise context a model cannot actually use.

---
schema_version: 1
id: "iss-2609202048576967"
slug: "a-fresh-v0-8-1-install-on-a-128-gb-mac-refuses-every-downloa"
severity: "major"
category: "bug"
source: "manual-test"
found_during: "maintainer's install of v0.8.1 on a second account of the serving Mac, 2026-09-20; reproduced arithmetically from the charge formula and the model's published configuration"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
---

A fresh v0.8.1 install on a 128 GB Mac refuses every downloaded model at default settings, and the warning tells the operator to turn the wrong knob. The control panel reports that the smallest model on the Mac, mlx-community/GLM-OCR-bf16 (about 2 GB of weights), needs about 162 GB, so every request is refused until the budget is raised. The figure is the served-window charge at its defaults, not a fault in the arithmetic: the model declares max_position_embeddings 131072 with 16 layers, 8 KV heads and head_dim 128, so one token of cache is 65536 bytes at f16 and 327680 bytes after the safety factor of 5; the served window defaults to the declared 131072 and decode concurrency to 4, and 327680 x 131072 x 4 is 160 GiB of cache on top of 1.2 x the weights, against a default budget of about 77 GiB on this Mac. Every model the maintainer serves declares a window of 131072 or more, so none fits, and the six that the previous build served without a per-model setting are refused by this one. Two defects, one record. (1) The default served window is the declared window, which for every current long-context model exceeds what any Mac holds at the default concurrency: a fresh install has nothing to serve until the operator discovers Settings > Served context for each model. The record chose the served window as the remedy on 2026-09-09 (spc-2609091431038544, commit 35a0bba6) and left its default at the declared figure; the default needs a decision — the largest window that fits the budget at the concurrency in force, a fixed default such as 32768, or a first-load prompt — and the decision is the maintainer's. (2) tooSmallWarning (internal/app/app.go) says every request is refused until the budget is raised, but 162 GB exceeds the machine and raising the budget cannot help; the message that would help names the served window and the concurrency, which the pool's own refusal already does (internal/runtime/pool.go, needs about ... of memory but the budget is ...) and the warning does not. Captured, not fixed: the default is a design decision and the message change follows it.

---
schema_version: 1
id: "iss-2610042036091478"
slug: "the-empty-prompt-check-refuses-a-prompt-made-only-of-unicode"
severity: "nitpick"
category: "bug"
source: "impl-review"
found_during: "adversarial security review of fix/refuse-non-string-prompt, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/emptyprompt.go"
remedy: "Measure on the Mac whether a prompt of only Unicode whitespace tokenises to nothing on the pinned runtime; if it does not, trim only ASCII whitespace in the emptiness check, with a test either way."
---

The empty-prompt check refuses a prompt made only of Unicode whitespace that the model server would answer. internal/gateway/emptyprompt.go trims a /v1/completions prompt with strings.TrimSpace, which treats U+00A0 (no-break space), U+3000 (ideographic space) and similar characters as space. So a prompt of only those is refused as empty (probed in the security review), although mlx-lm would tokenise it and answer. The check exists because a prompt that tokenises to nothing froze the model server on the Mac; a no-break-space prompt tokenises to something. It fails safe (a refusal, not a crash) and predates the non-string refusal.

---
schema_version: 1
id: "iss-2609190106563320"
slug: "three-smaller-findings-from-the-bridge-s-security-review-tog"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/bridge.go"
resolution: "The token is trimmed on the way in, its field is a password field like the HuggingFace token beside it, the over-window refusal tells whoever asked that the conversation is too long rather than something untrue about the model, and the editor carries the rune count as the answer arrives instead of re-materialising it on every delta."
impact: fix
---

Three smaller findings from the bridge's security review, together because each is one line. (1) internal/bridge/discord/bridge.go tests the token against the empty string only, so a token pasted with a trailing newline — the ordinary result of copying one from a browser or a terminal — opens a connection, is refused 4004, and puts 'Discord refused the bot token' on the panel: advice that will fail again for the same invisible reason. Trim it on the way in. (2) internal/ui/static/index.html draws the bot token in a text field while the HuggingFace token beside it is a password field; the stored value is always the placeholder, so nothing is disclosed on load, but the token is in the clear while it is being pasted. (3) internal/gateway/ask.go answers an over-window conversation with the generic refusal, so the one refusal the person on Discord could act on tells them something untrue about the model instead of that their message is too long. A fixed public sentence for that case leaks nothing about the Mac.

## Grounds

- pursued: we expect the fix to hold because a test watched to fail covers it; wrong if the same class of defect appears at a seam this change did not touch

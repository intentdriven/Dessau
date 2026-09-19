---
schema_version: 1
id: "iss-2609190106273104"
slug: "internal-bridge-discord-answer-go-logs-askerr-error-under-th"
severity: "major"
category: "security"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/answer.go"
---

internal/bridge/discord/answer.go logs askErr.Error() under the reason key at Info for every refused bridged request. That string is gateway.AskError's detail, which ask.go fills from the pool's own error — including a runtime.LaunchError, whose text is the child process's verbatim and carries absolute local filesystem paths under the serving account's home directory. The HTTP path refuses to do this: handleCompletions logs a LaunchError by model name at Info and its text only at Debug, and rate-limits every other pool refusal to one line per model per minute, on the stated rule that a stranger who can drive a refusal must not be able to drive a detailed description of this Mac into a file at the rate it can send requests. The bridge's caller has neither guard, so anyone who can message the bot can put the operator's home-directory and venv paths into the log at the default level and repeat it at the work queue's rate until the size-bounded log rolls. Ask should classify a refusal for the log the way handleCompletions does, and the bridge should log the class rather than the text.

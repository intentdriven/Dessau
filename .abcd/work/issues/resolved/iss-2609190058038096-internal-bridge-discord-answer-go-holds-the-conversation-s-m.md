---
schema_version: 1
id: "iss-2609190058038096"
slug: "internal-bridge-discord-answer-go-holds-the-conversation-s-m"
severity: "major"
category: "security"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/answer.go"
resolution: "A channel that is already being answered is now recognised rather than waited on: the answer takes the conversation's lock with TryLock and, failing, tells the person the bot is still answering their last message and releases the worker. TestABusyChannelDoesNotStopEveryOtherConversation holds it — with the blocking lock restored the whole package deadlocks and the test times out."
impact: fix
---

internal/bridge/discord/answer.go holds the conversation's mutex for the whole of an answer, including the model's generation, which can run for minutes. A second message in the same channel is queued as a job, taken by one of the two answer workers, and then blocks on that mutex while holding the worker. Anyone who can reach the bot may talk to it, so two messages sent into one channel during a long answer occupy both workers and every other channel and direct message stops being answered until the first answer finishes — a denial of service one person can cause by pressing send twice. A channel that is already being answered should be recognised as busy rather than waited on, so the worker is released and the person is told, or the message is dropped with the line the queue-full case already writes.

## Grounds

- pursued: we expect refusing a concurrent answer in one channel to keep every other channel served because the workers are the scarce resource and nothing else holds one for minutes; wrong if somebody legitimately wants two questions answered at once in one channel, which nothing in the intent asks for

---
schema_version: 1
id: "iss-2609190312182409"
slug: "the-bridge-s-model-pick-test-can-be-answered-by-a-refusal-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial security review of the conversation-store change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/answer_test.go"
resolution: "The wait a test uses to know a message was answered is now 'answered', which waits for the POST, refuses the still-answering reply as an answer, and then waits for the channel's answering lock to be free — so the next message in that channel cannot be met with a refusal the next wait would mistake for an answer. Applied to the model-pick test and to both conversation-lifetime tests. Watched to fail with the old pattern and a 150ms hold past the POST: 'the gateway was asked 1 times, want twice'."
impact: internal
---

The bridge's model-pick test can be answered by a refusal instead of an answer, and flakes about one run in sixty under -race. internal/bridge/discord/answer_test.go waits for a POST to the channel's messages endpoint to know a message has been answered, but the placeholder is posted from inside the streaming callback while the channel's answering lock is still held — so the next message the test sends is met with 'I am still answering your last message here', which is a POST to the same path and satisfies the next wait. The test then asserts against one completion instead of two. The production behaviour is right and the harness is wrong: the wait has to mean the answer is finished, not that a message was posted. TestAChannelKeepsItsConversationAndModelAcrossAReconnect has the same shape and is saved only by the two-second reconnect backoff it sleeps through.

## Grounds

- pursued: we expect a wait that means the answer is over to be the lock rather than the POST, because the placeholder is posted from inside the streaming callback; wrong if a test ever wants to drive the still-answering path deliberately, which must then wait on the POST itself.

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
---

The bridge's model-pick test can be answered by a refusal instead of an answer, and flakes about one run in sixty under -race. internal/bridge/discord/answer_test.go waits for a POST to the channel's messages endpoint to know a message has been answered, but the placeholder is posted from inside the streaming callback while the channel's answering lock is still held — so the next message the test sends is met with 'I am still answering your last message here', which is a POST to the same path and satisfies the next wait. The test then asserts against one completion instead of two. The production behaviour is right and the harness is wrong: the wait has to mean the answer is finished, not that a message was posted. TestAChannelKeepsItsConversationAndModelAcrossAReconnect has the same shape and is saved only by the two-second reconnect backoff it sleeps through.

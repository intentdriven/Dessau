---
schema_version: 1
id: "iss-2609181045444147"
slug: "the-chat-client-s-composer-should-have-the-size-and-form-of"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "the maintainer's first look at the 0.7.0 client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The composer takes the form of Messages' composer with standard shapes only: the field carries textInputBorderShape(.capsule) and the send and stop buttons are borderedProminent with buttonBorderShape(.circle), all at the large control size; TestChatClientComposerHasMessagesForm holds it. The '+' and in-field controls of Messages are not part of this change."
impact: fix
---

The chat client's composer should have the size and form of Apple Messages' composer: a full-width capsule text field with the round '+' button outside it on the left and the dictation and emoji controls inside on the right, and a send button that appears as a filled circle inside the field once there is text (the maintainer's screenshot, 2026-09-18). Today the composer is a plain multi-line TextField with a bordered arrow button beside it. The no-styling rule allows this only through standard modifiers (macOS 27's textInputBorderShape for the capsule; a bordered prominent button style for the circle), never a drawn background, and it applies to both the Mac and the iPad clients.

## Grounds

- pursued: we expect the system's capsule field and circular prominent button to read as Messages' composer without a drawn background; wrong if the maintainer still finds the form off next to Messages

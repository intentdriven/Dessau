---
schema_version: 1
id: "iss-2609202237468921"
slug: "a-model-adopted-from-disk-can-never-be-a-chat-model-so-a-sec"
severity: "major"
category: "bug"
source: "manual-test"
found_during: "maintainer's install of v0.9.0 on a second account of the serving Mac, 2026-09-20; diagnosed from the server's own models list and control plane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/registry.go"
---

A model adopted from disk can never be a chat model, so a second account's install lists every model and Dessau Chat refuses them all. Rescan adopts a complete model directory with its size, context length and KV charge, but not the Hub's pipeline_tag and tags, which are fetched only when the model is downloaded and which nothing in the directory can tell the registry again (registry.go, Rescan's comment). The chat rule requires a pipeline tag of text-generation or image-text-to-text and the tag conversational, so an adopted model is chat: false on the models list, and Dessau Chat's picker says none of this server's models can hold a conversation. Seen on the maintainer's v0.9.0 install: six models, all adopted at the same second, all chat: false with no pipeline_tag and no tags. The shared-cache design promises a second account sees the first account's models; it does not yet let it talk to them. Workaround: an empty chat rule matches every model. Fix shapes for the maintainer's decision: fetch the Hub's facts for adopted models when online; or let the directory answer, since a chat_template in tokenizer_config.json is what the runtime renders; or both, the template as the offline fallback.

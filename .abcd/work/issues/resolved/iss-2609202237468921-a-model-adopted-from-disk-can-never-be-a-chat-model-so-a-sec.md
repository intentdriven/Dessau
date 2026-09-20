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
resolution: "Fixed on fix/adopted-models-chat: the directory answers offline — a chat template in tokenizer_config.json or a chat_template.jinja file makes an adopted model a chat model when the Hub's word is absent, and where the Hub's word exists the operator's chat rule stands; the chat verdict has one home, Model.CanChat, held by an architecture test; the Hub's words are completed in the background for adopted models once per start, one model at a time, never blocking startup, a load or a request, and a repo the Hub has no words for is not re-asked forever. Hand-checked on the built binary against the small model copied into a scratch root as an adopted directory: with the Hub unreachable the list read chat true from the template alone and a request was served; online, the Hub's words were recorded within two seconds of start."
impact: fix
resolved_by:
  commit: "12fca312"
---

A model adopted from disk can never be a chat model, so a second account's install lists every model and Dessau Chat refuses them all. Rescan adopts a complete model directory with its size, context length and KV charge, but not the Hub's pipeline_tag and tags, which are fetched only when the model is downloaded and which nothing in the directory can tell the registry again (registry.go, Rescan's comment). The chat rule requires a pipeline tag of text-generation or image-text-to-text and the tag conversational, so an adopted model is chat: false on the models list, and Dessau Chat's picker says none of this server's models can hold a conversation. Seen on the maintainer's v0.9.0 install: six models, all adopted at the same second, all chat: false with no pipeline_tag and no tags. The shared-cache design promises a second account sees the first account's models; it does not yet let it talk to them. Workaround: an empty chat rule matches every model. Fix shapes for the maintainer's decision: fetch the Hub's facts for adopted models when online; or let the directory answer, since a chat_template in tokenizer_config.json is what the runtime renders; or both, the template as the offline fallback.

## Grounds

- pursued: we expect a model's own chat template to say whether it can hold a conversation at least as well as the Hub's tags do, because the template is what the runtime renders; wrong if a templated model refuses to converse or a tagged conversational model has no template

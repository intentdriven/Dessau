---
id: spc-2610030950480763
slug: dessau-tells-a-client-which-of-its-models-are-versions-of
intent: itd-2610030932551549
origin: researcher-authored
production_mode: hand-written
---
# Dessau's models list groups builds of the same model and gives each one's facts

## Summary

Delivers itd-2610030932551549: each `/v1/models` entry and each control-panel
model card carries the model HuggingFace names as its origin, when its own Hub
label says it is a quantised build of it, and the facts that tell builds apart.
Additive fields only; every build keeps its own name. Designed from the two
adversarial reviews and the planning interview of 2026-10-03.

## Approach

### The group
A build belongs to a group when its recorded Hub tags (captured at download,
`internal/registry`) carry exactly one `base_model:quantized:<origin>` tag; the
origin is folded with `config.FoldRepoID`, the one canonical rule for model
ids. No tag, or more than one, means no group. No name parsing, no config-shape
matching (the interview's decision 1). Tags are recorded once, at download.

### The facts
- `quantization_bits`: from the model's own `config.json` (its quantization
  block), extending the canonical parser behind `hub.Model.Quantization()`
  rather than adding a third one; absent when the model is not quantised.
- `size_bytes`: the registry's recorded size on disk.
- `served_context`: already published.

### The models list (additive)
Each entry gains `build_of` (the folded origin, absent when none),
`quantization_bits` and `size_bytes`. Builds are one group when `build_of` and
the model kind are equal (a decision model's `kind: "decision"`, from
itd-2610030656210408, never matches a chat model), which `docs/models-list.md`
states as the contract. A `build_of` value is never resolved as a model name:
`resolveModel` matches full ids and short names of served models only, and a
test pins that a group name that is not itself a served model is refused as
unknown.

### The control panel
Model cards with the same group are shown together under the origin's name,
each with its precision, size and served context, by the same rule the list
uses (one function). Every Hub-supplied string is rendered through the panel's
escaping helper.

## Steps

1. The group and the facts in the models list
   - packages: internal/registry, internal/hub, internal/gateway
   - tests: two builds tagged with one origin share build_of and keep their ids; no tag or two tags means no group; quantization_bits from config.json; size_bytes; a decision and a chat build with one origin are not one group; a group name is refused as a model name
   - landed: #150
2. The control panel and the docs
   - packages: internal/ui, internal/gateway, docs
   - tests: the panel groups exactly as the list does; markup in Hub strings is shown as text; docs/models-list.md documents the three fields and the grouping rule

## Footprint

- packages: internal/registry, internal/hub, internal/gateway, internal/ui
- tests: the six acceptance criteria, held as listed in the steps

## Out of scope

Grouping by name or model shape; families such as Flash and full; any wording
of descriptions (the client's); refreshing tags after download.

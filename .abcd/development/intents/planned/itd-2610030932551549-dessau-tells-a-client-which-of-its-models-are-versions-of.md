---
id: itd-2610030932551549
slug: dessau-tells-a-client-which-of-its-models-are-versions-of
spec_id: spc-2610030950480763
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091129451578, itd-2609061431463108, itd-2609081259493890, itd-2609100519003748]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Dessau's models list groups builds of the same model and gives each one's facts

## Press Release

Dessau tells a client which of its models are builds of the same model, and the
facts that tell them apart.

When Alice has downloaded a 4-bit and an 8-bit build of one model, Dessau's
models list and her control panel show them as builds of the model HuggingFace
names as their origin, each with its precision, its size and the context it is
served with. A tool Bob writes can pick the build that fits its job without
decoding repository names, and every build is still called by its own name.

## Why This Matters

Servers now hold several builds of one model (two Qwen or two Clef builds, say),
and a client sees near-identical repository names with nothing saying they are
interchangeable or what choosing one trades. Seeded 2026-10-03 from the
maintainer's request; split into this server intent and the Dessau Chat intent
itd-2610030932556747 at filing.

## Mechanism

- We expect tools to pick the right build because each build's facts come from
  the model itself, not from its name. Shown wrong if a published fact
  contradicts the control panel's figures.

## Scope Conditions

- Builds are grouped only when HuggingFace labels them as quantised from the <!-- cond: cond-2610030950485151 -->
  same model; anything else appears on its own.
- Every build stays callable by its own name; a client that ignores the new <!-- cond: cond-2610030950487040 -->
  information sees nothing new.
- A decision model never shares a group with a chat model. <!-- cond: cond-2610030950482963 -->

## Acceptance Criteria

- **Given** two builds that HuggingFace labels as quantised from the same model,
  **when** any client reads the models list, **then** each entry keeps its own
  name and carries the same group (that origin) plus its precision, size and
  served context.
- **Given** a model with no such label, **when** it is listed, **then** it
  appears on its own, with its own facts.
- **Given** a decision model and a chat model labelled with the same origin,
  **when** they are listed, **then** they are not grouped.
- **Given** the control panel, **when** Alice opens her models, **then** it
  groups and describes builds exactly as the models list does.
- **Given** a request that names a group rather than a build, **when** it
  reaches Dessau, **then** it is refused as not a model name; every build stays
  callable by its own name.
- **Given** origin labels or other text from HuggingFace containing markup,
  **when** the panel shows them, **then** they appear as plain text.

## Decisions at the planning interview (2026-10-03)

1. Same model means HuggingFace's own `base_model:quantized:<origin>` label and
   nothing else: no name parsing, so the "no taxonomy of our own" rule of
   itd-2609091129451578 holds. bf16 copies, older repositories and adopted
   models without the label appear on their own.
2. Flash and full builds are different models on the Hub and are left out; no
   operator-declared family.
3. Descriptions are facts (precision, size on disk, served context), published
   as fields for clients to word; no new setting, and nothing claims
   "capability".
4. Delivered separately from the Dessau Chat intent, which waits on this one.
5. The word is "build", not "version", so it never collides with "a newer
   version" in itd-2610030857275099.

Typed links: refines itd-2609091129451578; builds on itd-2609061431463108,
itd-2609081259493890 and itd-2609100519003748.

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: servers now hold several builds of one model, and clients see near-identical names with no way to choose; shown wrong if, once grouped, nobody chooses between builds differently than before. And programs that call Dessau need to pick the build that fits their job without hard-coding repository names; shown wrong if such tools keep hard-coding names anyway.

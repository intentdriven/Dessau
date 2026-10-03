---
id: itd-2610030656210408
slug: dessau-answers-typed-decisions-not-only-chat-alice-runs-a
spec_id: spc-2610030846273729
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091129451578, itd-2609091412177263, itd-2609061441261073, itd-2609061429508050]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# Dessau answers typed questions with odds, not only chat

## Press Release

Dessau answers typed questions with odds, not only chat.

Alice's tool sorts incoming messages: is this urgent, which team should take
it, how soon? She downloads a Clef decision model through Dessau, sends the
message and her questions to Dessau's decision endpoint, and gets back each
answer with the probability of every option it could have chosen, in a single
pass, from the same server and under the same key as her chat models. Carol's
chat client never offers that model for conversation: the models list marks it
as a decision model, and each endpoint turns away the other kind of model with
a pointer to the one that fits.

## Why This Matters

A tool that has to act on a verdict needs the verdict and how sure the model
is, not a paragraph to parse. Decision models such as the Clef family answer
typed questions (`noul` yes/no, `choice`, `score`) in one forward pass with a
probability per option, through an interface their makers call SystemOne. Until
Dessau serves them, the only way to run one on this Mac is the model's own
reference server, run by hand: it has no API key, no pairing and no place in
the memory budget, so the pool's figure stops being true the moment it runs.
Seeded on 2026-10-03 from a request by a local tool that classifies flagged
text; the interim hand-run route is recorded in `.abcd/work/DECISIONS.md` the
same day.

## Mechanism

- We expect Dessau's answers to match the reference server's because it runs
  the reviewed reference code on the reviewed weights. Shown wrong if the
  upgraded engine moves any probability by more than 0.001 on the fixed test
  set.
- We expect tools like Alice's to prefer a decision model over prompting a chat
  model, because one pass with a probability per option is faster and easier
  to act on than parsing chat text. Shown wrong if a chat model prompted for
  the same verdicts is as accurate and as fast.

## Scope Conditions

- An Apple Silicon Mac where Dessau runs under one account and everyone else, <!-- cond: cond-2610030846278067 -->
  including other accounts on the same Mac, reaches it over the network.
- Decision models are Clef-family builds at a version checked against the <!-- cond: cond-2610030846275650 -->
  reference, offered only where the budget holds them; chat models are
  unaffected.
- The text sent to a decision model is text or JSON; images are a later intent. <!-- cond: cond-2610030846279580 -->
- No access rule is added or changed: no key on this Mac, the API key and <!-- cond: cond-2610030846278174 -->
  paired clients across the network.

## Acceptance Criteria

- **Given** a checked Clef build that fits the budget, **when** Alice sends a
  text and `noul`, `choice` and `score` questions to `POST /v1/systemone`,
  **then** every answer has the reference server's shape and the same top
  answer, and every probability is within 0.001 of the reference server's, on
  a fixed test set.
- **Given** a decision model on this Mac, **when** any client reads
  `/v1/models`, **then** it is marked as a decision model and `chat: false`,
  whatever the chat-model rule says.
- **Given** a decision model named on a chat endpoint, or a chat model named on
  `/v1/systemone`, **when** the request arrives, **then** it is refused with a
  message naming the endpoint that fits, and nothing is loaded.
- **Given** more text than the model can read, **when** the request arrives,
  **then** it is refused, saying it is too long and by how much; nothing is
  ever answered from part of the text.
- **Given** the existing access rules, **when** a client calls
  `/v1/systemone`, **then** it is admitted or refused exactly as on chat: no
  key on this Mac; the API key or pairing across the network.
- **Given** a decision model on this Mac, **when** idle work runs (the context
  probe, the tool-call probe, the self-test), **then** none of it ever sends
  the model a chat request, and its readiness is proven by a decision request.
- **Given** a newer upstream version of a Clef build, **when** Alice downloads
  it through Dessau, **then** Dessau fetches the checked version, and refuses
  with a reason to load one whose files do not match the check.
- **Given** a Mac whose budget cannot hold a build, **when** Alice browses or
  loads it, **then** Dessau says so before download as well as at load, and
  the two verdicts always agree.
- **Given** a decision model is loaded, **when** it serves requests, **then** no
  Python from the model's folder runs: the server is Dessau's own reviewed
  copy, it fetches nothing from the network, and a request carrying images is
  refused.
- **Given** an existing install updates to the upgraded runtime, **when** it
  starts, **then** the runtime is reinstalled automatically, and every chat
  behaviour checked against the old engine (the sampling flags, the reply
  limit, the model-code refusal and the refused request fields) is checked
  again and still holds.
- **Given** a decision model, **when** the operator opens the control panel,
  **then** it shows the model as a decision model, as `/v1/models` does: the
  panel and Go say the same thing.

## Decisions at the planning interview (2026-10-03)

1. Split adopted: this intent carries the user-facing capability only. "Dessau
   never runs code that comes with a download" is a standing rule recorded on
   its own (and first enforced by the fix for iss-2610030709283687); adopting
   a trimmed copy of the reference implementation, and its Apache-2.0 licence
   in this project, is an architecture decision; the new Python packages are a
   dependency sign-off recorded on their own; images are a later intent.
2. Runtime: one runtime, upgraded for every model to the engine the reference
   code needs (the reviews found mlx-vlm 0.7.x requires a newer mlx than the
   pinned one). Chat behaviour verified against mlx-lm 0.31.3 is re-verified as
   part of this work. The maintainer first answered "a second runtime", asked
   for the question again, and chose the upgrade.
3. Memory: any Clef build that fits the Mac's budget, each checked against the
   reference separately (asked twice, same answer).
4. Wrong model: refused both ways, naming the endpoint that fits. This reverses
   cond-2609091236502214 of itd-2609091129451578 ("every model stays callable
   by name") for decision models on chat endpoints and chat models on the
   decision endpoint, so the impact is breaking.
5. Too long: refused, never truncated (the reference server truncates by
   default).
6. Updates: the reviewed version only; a newer upstream version is ignored until
   Dessau ships a reviewed update.
7. Match: same top answer, each probability within 0.001.

Typed links: reverses cond-2609091236502214 of itd-2609091129451578 (above);
refines itd-2609091412177263 (each refusal names what Dessau cannot do);
refines itd-2609061441261073 (a decision model is charged to the same budget);
refines itd-2609061429508050 (the pure relay stays true for sampling; the
decision endpoint carries no sampling). The adversarial reviews' findings this
intent answers: the reference server fetches image URLs, passes client keyword
arguments to the image processor and truncates silently; the readiness probe,
the self-test and the tool-call probe all drive chat; the "fits" filter and the
pool's charge disagree for this model; downloads resolve `main` and record no
commit.

## Open Questions

_None open._ Settled in spc-2610030846273729: the response shape is the
reference's (its "Response shape" section); a decision model is recognised by
its joint head files and the reviewed-build manifest, never by Hub tags alone;
a decision request is counted in the statistics, and whether the planned
transcript store records it is settled when that store lands
(itd-2609091707499248).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a local tool needs typed verdicts with odds, and chat answers are too slow and too loose for it; shown wrong if that tool's test on the hand-run reference server shows a chat model does as well. And every model this Mac serves should sit behind Dessau's access rules and memory budget; shown wrong if the hand-run reference server is still needed after this ships.

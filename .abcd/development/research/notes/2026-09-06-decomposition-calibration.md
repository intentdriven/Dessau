# Decomposition calibration (itd-84 hand-run)

One graded entry per proposal put through the routing table before filing.
The corpus gates the automated pre-pass; record whether the initial routing
survived the human's confirmation, not just the final table.

## 2026-09-06 — Gropius landing page, updated on release via Cloudflare Workers

Proposal as stated: "Build a Gropius landing page (intentdriven.sh/Gropius for
now) updated via Cloudflare workers when a new release is cut."

Initial routing proposed: FILE-AS-IS as one intent.

| Part | Type | Home |
| --- | --- | --- |
| Alice opens one link and gets the current release download plus the repository | user-facing capability | intent itd-2609061353254616 |
| The page reflects a new release without anyone editing it | user-facing capability | intent itd-2609061353258535 (builds on the first) |
| Cloudflare Worker; pull from the Releases API vs push from release.yml | plumbing | the second intent's spec |
| Page copy renders from the canonical identity block | existing invariant | IDENTITY.md; the first intent refines it by adding a surface |
| Hosting at intentdriven.sh "for now" | decision | .abcd/work/DECISIONS.md, 2026-09-06 |
| Page and Worker source location, second toolchain, dependency sign-off | scope question | open questions on both drafts, for the planning interview |

Typed links: the first intent `refines` the canonical identity invariant. No
supersedes, reverses, or duplicates. No existing `abcd site` composition.

Verdict adopted by the human: SPLIT. The initial FILE-AS-IS routing did not
survive: the human separated the static page (which already tracks the primary
download through the latest-release asset URL) from release-driven updating,
so the page can ship ahead of the Worker. Grade: routing of parts to homes
held; the verdict did not.

## 2026-09-06 — Expose model parameters (seed, temperature) via Settings

Proposal as stated: "expose model parameters (seed, temperature) via Gropius.
Make this configurable via Settings."

Initial routing proposed: FILE-AS-IS as one intent plus one issue capture.

| Part | Type | Home |
| --- | --- | --- |
| Alice sets default sampling parameters in Settings; requests that omit them get those values | user-facing capability | intent (sampling defaults) |
| Reproducible generation via a seed set in Settings | user-facing capability | intent (seed), gated on the verification issue |
| Per-request values from the client still pass through | existing behaviour | docs; stated as a scope condition on both intents |
| Whether mlx-lm 0.31.3 honours a per-request seed | fact to establish | issue capture |
| Precedence between request and Settings values; global vs per-model | open questions | planning interview |
| Injecting defaults into the buffered request body | plumbing | the spec; gateway trust boundary, security review at PR |

Typed links: none to existing records. The gateway's "rewrite only the model
field" stance is a code comment, not a recorded invariant; the spec must name
that this is the first feature merging fields into request bodies.

Verdict adopted by the human: SPLIT. The initial FILE-AS-IS routing did not
survive: the human separated temperature (and the other accepted sampling
parameters) from seed, because seed depends on an unverified upstream
behaviour and must not block the rest. Grade: parts-to-homes held; the
verdict did not. Second consecutive SPLIT where FILE-AS-IS was proposed.

## 2026-09-06 — Manage context size and tell clients the context size per model

Proposal as stated: "Carefully manage context size and let clients know what
the current context size is for each model (if possible)."

Initial routing proposed: SPLIT into two intents, measurement as an open
question on the second.

| Part | Type | Home |
| --- | --- | --- |
| Alice's client reads each model's context length from the models list | user-facing capability | intent (context window visible) |
| Gropius serves only up to the context it can hold, reports it, rejects over-long prompts clearly | user-facing capability | intent (effective context managed) |
| Architectural cap read from the model's config at rescan | plumbing | first intent's spec |
| Effective cap = min(architectural cap, KV headroom under the budget) | mechanism claim | second intent; refines iss-3 |
| Measuring the effective window per model | fact to establish | issue capture |
| Token counting in the gateway vs relying on the server's rejection | open question | planning interview |

Typed links: the second intent `refines` iss-3 (memory budget ignores KV
cache from decode concurrency). Nothing superseded or reversed.

Verdict adopted by the human: SPLIT, plus an issue capture for the
measurement. The proposed split survived; the human additionally promoted
the measurement from an open question to its own ledger record so it is
tracked. Grade: parts-to-homes held; verdict held with one part re-homed
from "open question" to "issue capture". First run where the proposed
verdict survived, after two FILE-AS-IS proposals that were split.

## 2026-09-06 — Model-bench lab review: five intents and three captures

Proposal as stated: the human adopted, verbatim, the list of lab-derived
recommendations the session had presented ("All of it"). Each item had
already been routed in that presentation, so this run records the routing
as confirmed rather than re-proposed.

| Part | Type | Home |
| --- | --- | --- |
| Client sees which models are loaded, picks the warm one | user-facing capability | intent (residency visible) |
| Models Alice pins stay resident; evicting request refused | user-facing capability | intent (pinned models) |
| Alice sets the resident memory budget in Settings | user-facing capability | intent (configurable budget) |
| Recently used models get a grace interval; requests wait bounded time | user-facing capability | intent (eviction grace) |
| Opt-in per-model merging of system messages | user-facing capability | intent (system-message merging); first gateway feature reading prompt content, named in its open questions |
| Docs omit budget default, eviction rule, preload-does-not-pin | defect | issue capture, docs category |
| KV-cache cost per architecture | evidence | appended to existing iss-3 |
| Lab measurements and co-residency arithmetic | evidence | dated research note 2026-09-06-model-bench-findings |

Typed links: the eviction-grace intent `refines` the pool's LRU rule, which is
a code comment and README sentence rather than a recorded invariant; the
configurable-budget intent `refines` iss-3 and iss-6 (budget accounting);
the docs capture `refines` the README's "LRU memory budget" claim. Nothing
superseded or reversed.

Verdict adopted by the human: FILE-AS-IS for all eight parts. Grade: routing
survived unchanged. Note for calibration: this run's routing was proposed in
prose a turn earlier and adopted as a batch, so it is weaker evidence than a
table confirmed part by part.

## 2026-09-06 — Local telemetry: statistics, on-disk store, telemetry pack, dashboard

Proposal as stated: the local-telemetry research recommendation "plus we also
need it to be stored on disk so that we can later build a dashboard with
insights into token use, latency etc. -- we want to use Gropius to learn
about the use of local models; we also want to enable clients to request a
'telemetry pack' for their session(s)".

Initial routing proposed: SPLIT into three intents, dashboard drafted or held.

| Part | Type | Home |
| --- | --- | --- |
| Opt-in statistics shown per model in the control panel | user-facing capability | intent (local request statistics) |
| Records kept on disk with stated retention and format | user-facing capability | intent (durable store), builds on the first |
| A client fetches a telemetry pack for its own sessions | user-facing capability | intent (telemetry pack), builds on both |
| A dashboard over the store | user-facing capability, later | intent (usage dashboard), drafted |
| "Learn about the use of local models" | conjecture | grounds at each gate, not a record |
| On-disk format and retention (JSON Lines vs SQLite, a new dependency) | decision | ADR when the store's spec decides it |
| What a session is on an API with none; who may fetch whose pack | trust-boundary rule | open question on the pack intent; likely its own ADR |
| No prompt text, completions, keys or client addresses in statistics | existing invariant | adr-2609061503319212; all four refine it |

Typed links: all four `refine` adr-2609061503319212. The statistics intent
`refines` itd-2609061441228998 (residency visible). Context-fill figures
depend on itd-2609061431463108. The research note's advice against SQLite
was scoped to a bounded ring; a durable dataset reopens it, no reversal.

Verdict adopted by the human: SPLIT, with the dashboard drafted as a fourth
intent. Grade: parts-to-homes held; the verdict held, with the human choosing
the "draft it" branch of the one open choice offered.

## 2026-09-06 — Planning interview across the store: re-routing after review

The interactive planning interview (fifteen drafts, two adversarial
reviewers per theme) changed four earlier routings. Recorded here because a
decomposition that survives filing and then falls at planning is the signal
the calibration corpus exists to catch.

| Record | Earlier routing | Outcome at planning | Why |
| --- | --- | --- | --- |
| itd-2609061429516182 (seed in Settings) | intent, gated on a verification issue | superseded by itd-2609061429508050 | The verification found the server ignores seed entirely; the capability cannot exist. A gated intent was the wrong home for an unverified premise; a capture plus a documented fact would have sufficed. |
| itd-2609061521134968 (telemetry pack) | intent, with an access-rule ADR to follow | dropped, superseded by adr-2609061503319212 | Both reviewers found it contradicted the ADR's "for the operator of that Mac". The decomposition should have flagged the reversal at filing rather than routing the conflict to a future ADR. |
| itd-2609061431481936 (effective context) | intent, refines iss-3 | held | Needs an ADR on the budget rule and a measurement now confounded by a newly found gateway timeout; three prerequisites, none in the original table. |
| itd-2609061602043757 (summary before deletion) | not present | new intent, builds on the store | Surfaced by the maintainer while answering the retention question; the store's decomposition missed that deleting detailed records loses the history the dashboard exists for. |

Two ADRs were minted at planning rather than filed as parts at capture time
(store format; the gateway's one permitted prompt rewrite); both had been
named as "ADR when the spec decides" in earlier tables, which held.

Grade for the corpus: of fifteen filed intents, eleven survived the
interview as routed, one was re-homed as a new intent's parent, and three
changed bucket. The two that fell were both cases where a trust-boundary or
upstream-behaviour question had been routed forward instead of resolved.

## 2026-09-08 — Tailscale: does a mesh VPN benefit Gropius, and should features key on it?

Proposal as stated: "I have tailscale installed on my machines. Does that
benefit Gropius? And could we activate certain features when Gropius detects
Tailscale? The other way around: if not, should certain features be unavailable
automatically?"

Initial routing proposed: SPLIT — one intent now, one intent later, one ADR,
docs deferred.

| Part | Type | Home |
| --- | --- | --- |
| The endpoint list names a tailnet address as private and encrypted, under the name the tailnet resolves | user-facing capability | intent (itd-2609081015545349), filed |
| A "tailnet only" bind the local network cannot reach | user-facing capability, later | intent, deferred to a separate filing by the human |
| Detection may inform what the app reports, never what it enforces | trust-boundary rule | adr-2609081118587999 |
| How-to for serving over a tailnet, and the standing warning against public funnelling | docs | follows whichever intent ships; not a record of its own |
| mDNS does not traverse a tailnet | verified fact | stated inside the intent, no record of its own |

Typed links: both intents `refine` adr-2609081118587999; the later bind intent
`builds_on` the filed one. Nothing superseded. No reversal — the filed intent
changes no enforcement, which is the ADR's rule applied to itself.

The reversal that was *not* filed is the interesting part: the obvious feature
("a tailnet is present, so waive the API key") reverses the exposure rule in
`config.validate` and the warning in the control panel's status. It was routed
to the ADR as a rejected alternative rather than to an intent. That is the
routing the earlier telemetry-pack run got wrong in the opposite direction —
there a trust-boundary conflict was routed forward to a future ADR and only
caught at planning.

Verdict adopted by the human: SPLIT, with the later bind intent explicitly
deferred to its own filing and the ADR requested by name. Grade: routing
survived. Calibration caveat, the same one the 2026-09-06 shipped-divergences
run carries — the options were put in prose a turn before the table existed and
the human adopted them as a batch, so this is weaker evidence than a table
confirmed part by part. The acceptance criteria on the filed intent are
agent-seeded and marked as such in its Open Questions; the planning interview
has not run.

## 2026-09-09 — model category from the Hub (itd-2609091129451578)

Proposal: the server lists a model's type (chat, coding, other) when browsing,
records it at download, advertises it via the API; clients pick by category.

| Part | Type | Home |
| --- | --- | --- |
| Search results show each model's category | user-facing capability | this intent |
| The category is recorded in the registry at download | mechanism | this intent (spec) |
| The models list advertises the category | user-facing API contract | this intent, plus the models-list reference page |
| GropiusChat's picker offers only chat models | client half | this intent (client criterion) |
| How a category is derived | mechanism claim | spec's Mechanism section, no ADR (no trust boundary) |
| A coding harness picks coding for implementation, chat plus coding for reviews | illustration of a third-party client | press release only |

Typed links: `refines` iss-2609081743175520 (`promoted_from`); `builds_on`
itd-2609061431463108 (the models-list field convention). Flagged reversal: it
widened the same day's decision to publish a boolean chat capability from the
chat template; the human confirmed the widening ("category supersedes the
boolean").

Verdict adopted: FILE-AS-IS, one intent. Grade: routing survived the human;
the VOCABULARY did not survive the design review — a 500-repository sample
showed "coding" is not derivable from Hub metadata, and the human then chose
"rely on the HuggingFace tag, don't invent your own", with a server-side chat
rule (default text-generation or image-text-to-text plus conversational) an
operator can change and the same default in GropiusChat. Lesson for the
protocol: run the feasibility review BEFORE the routing question when a
proposal names a taxonomy, because the taxonomy was the decision.

## 2026-09-09 — server-side logging (itd-2609091412177263)

Proposal: API users get clear messages on what is not possible but never the
why; the why is logged server-side (sparse default, detailed option), within
each account, never shared.

| Part | Type | Home |
| --- | --- | --- |
| API answers say what, never why | user-facing API contract | this intent |
| The why goes to a server log; sparse by default, detailed on request | user-facing capability (setting on three surfaces) | this intent |
| What each level contains, and what is never logged | mechanism | spec, plus the existing prompt-content invariant |
| Logs live within each account, never shared | trust-boundary rule | already shipped the same day (iss-2609091131311102); linked, not restated |

Typed links: `builds_on` the entitlement rule (iss-2609062210238684) and
per-account state (iss-2609091131311102); NOT a duplicate of the per-model
debug draft itd-2609062346072707 (a shared level control would fail the
statistics-switch guard). Flagged reversal: "never the why" for entitled
clients would reverse the same day's error-text decision; the human kept the
why for entitled clients, so nothing reversed.

Verdict adopted: FILE-AS-IS, one intent. Grade: routing survived. The design
review found the log the intent writes to does not exist (stderr only), which
became the first sentence of the mechanism rather than a routing change.

## 2026-09-09 — recording mode (itd-2609091707499248)

Proposal: a server-side mode recording every prompt and answer, explicitly
activated, announced to every client session as a first message, with no way
to circumvent it.

| Part | Type | Home |
| --- | --- | --- |
| A server-wide recording mode | user-facing capability | this intent |
| Explicit activation on three surfaces | capability | this intent |
| Every session told first, no way round | user-facing API contract | this intent |
| A second reader (and, it turned out, a writer) of prompt content | trust-boundary rule | a superseding ADR for adr-2609061610102325 |
| The store lives per-account | mechanism | the spec, on the per-account rule |

Typed links: `supersedes` itd-2609062346072707 (proposed); `refines` the
logging intent's no-prompts scope condition; `reverses` the never-retain
clause of the prompt-content ADR (flagged; the human confirmed the ADR route).

Verdict adopted: FILE-AS-IS with the ADR as companion. Grade: routing
survived; the PROMISE did not survive the design review — "told first, no way
round" is not a thing a stateless server can do — and the human HELD the
intent rather than adopt the every-answer rule. Lesson, same as the category
run: when a proposal's headline is a guarantee, run the feasibility review
before the press-release question, because the guarantee is the decision.

## 2026-09-09 — usage measurement, and held adaptive settings (itd-2609091712141073, itd-2609091712142715)

Proposal: fully dynamic, customised per-model and context-window settings on
the server; until then, collect the data that says what to configure.

| Part | Type | Home |
| --- | --- | --- |
| Per-model settings an operator can change | exists or in flight | no record; `builds_on` the unified models map and the served-window intent |
| Settings that adapt without a restart | capability, unproven | a held draft (itd-2609091712142715) the data lifts |
| Per-request and per-model usage facts on the dashboard, exportable | capability, buildable now | this intent (itd-2609091712141073) |
| No prompt content in it | invariant | already holds; linked |
| Which record carries it | mechanism | the spec: fields on the existing request line, no new kind |

Typed links: `builds_on` itd-2609061521082551 and its store ADRs; `builds_on`
itd-2609061431481936; `refines` itd-2609091301112705.

Verdict adopted: SPLIT. Grade: routing survived as proposed.

## 2026-09-09 — transcript exceptions per model (stub)

Proposal: an exception list so named models keep no transcripts under the
recording mode; recorded as a stub only, by the maintainer's instruction.

| Part | Type | Home |
| --- | --- | --- |
| Named models keep no transcript while recording is on | user-facing capability | this draft, a stub |
| Where the exception lives | mechanism | the per-model settings map; settled at planning |
| What a client is told for an excepted model | open question | inherits the parent's held notice question |

Typed links: `builds_on` itd-2609091707499248 (held); inherits its hold.
Verdict adopted: FILE-AS-IS as a stub, no interview, at the human's request.
Grade: routing not tested (the human asked for a stub); recorded for the count.

## 2026-09-09 — resources view (itd-2609091903463596)

Proposal: a dashboard for server activity and resource utilisation (number of
models, disk space, context window).

| Part | Type | Home |
| --- | --- | --- |
| Server activity | shipped and planned elsewhere | `duplicates` itd-2609061521082551 and itd-2609091712141073 |
| Resource utilisation: models, disk, budget vs resident, windows | capability | this intent |
| What is on and off | planned elsewhere | `builds_on` itd-2609081718534201 |
| Where it lives | mechanism | the spec; decided at interview: a block on the Models tab |

Verdict adopted: SPLIT. Grade: routing survived; the design review moved the
placement from "a tab" to "a block on the Models tab" and the human took it.

## 2026-09-10 — an intuitive control panel for the server

Proposal: an intuitive, state-of-the-art website to configure the Gropius
server — read as the control panel, not the landing page, which the human
confirmed.

| Part | Type | Home |
| --- | --- | --- |
| A redesigned control panel: settings found by task, state shown first, live validation in the server's words | user-facing capability | this draft (itd-2609100519003748) |
| Whether the panel may take a build step, a framework or an asset fetched from outside the Mac | trust-boundary rule and dependency sign-off | a decision before planning (DECISIONS.md, 2026-09-10); an ADR if adopted |
| "Intuitive" as a standing stance | standing stance | the draft's press release; no principle |
| Plumbing | none | — |

Typed links: `refines` itd-2609081259493890, itd-2609081718534201,
itd-2609091903463596. Reversal flagged against adr-2609061503319212 for an
asset fetched from a CDN; the human did not confirm a reversal, so none is
recorded and the question is held at the decision.

Verdict adopted: SPLIT. Grade: routing survived as proposed.

## 2026-09-10 — idle-time model self-test (itd-2609100457007827)

Proposal: an autonomous state-of-the-art self-test and (later)
self-optimisation environment: when activated it loads and unloads the
available models, runs standard tests, captures all telemetry, and feeds the
results into a self-optimisation setting.

| Part | Type | Home |
| --- | --- | --- |
| Opt-in self-test: load each model when idle, run a standard set, unload, record | capability | this intent |
| Self-optimisation from the results | capability, held | `duplicates` itd-2609091712142715, not filed twice |
| What the self-test may record, and which opt-in gates it | ADR consequence | adr-2609061503319212 already scopes it; a scope condition on the draft |
| Which tests are standard, which use cases | open questions | the draft, settled at planning |
| Where the switch lives, how runs are recorded | mechanism | the spec |

Typed links: `refines` itd-2609091301112705 (the probe is one test of this
harness; the human chose refines over supersedes); `builds_on` the 2026-09-06
model-bench campaign; `duplicates` itd-2609091712142715 for the second half.
Flagged, not classified: "captures all telemetry" against the ADR's no-prompt
rule — taken as a scope condition, not a reversal.
Verdict adopted: SPLIT. Grade: routing survived the human's confirmation
unchanged; the interview itself was delegated to the agent by the maintainer.

## 2026-09-18 — messaging apps as a way into Gropius

Proposal (the maintainer): a way of accessing the Gropius server via other
apps, e.g. Discord, WhatsApp.

| Part | Type | Home |
| --- | --- | --- |
| Bob chats with Gropius from a messaging app he already uses | capability | a Discord bridge intent, in the Go server under the three-surfaces rule |
| WhatsApp | different prerequisites (business account, public webhook) | held as a capture; its own intent later |
| Content leaves the Mac for a third party; a bot token is a new secret | trust-boundary rule, flagged as a reversal of the LAN-only posture | an ADR before code, minted at the interview |

Verdict proposed: SPLIT. Verdict adopted: SPLIT, as proposed. Grade: the
routing survived the human's confirmation unchanged.
## 2026-09-17 — the built-in model and the Gropius offer

Proposal (the maintainer, one sentence): make macOS 27 the client's default,
use the model Apple ships with macOS 27 as the default chat interface, and
offer Gropius-provided models when the client discovers a server.

| Part | Type | Home |
| --- | --- | --- |
| macOS 27 as the client's default floor | capability, exists | `duplicates` itd-2609151701196720 (planned 2026-09-16), not filed twice |
| The Mac's own model is the default chat backend, no server needed | capability | itd-2609170718438919, `builds_on` the trunk |
| A discovered Gropius server is offered, never adopted by itself | capability | itd-2609170718430553, `builds_on` the trunk |
| The client no longer connects at launch | reversal, flagged | the discovery promise held by `internal/archtest`'s chat-client tests; for the human at the offer intent's interview |

Typed links: `builds_on` itd-2609151701196720 for both; distinct from
itd-2609151836193724 (Writing Tools on text in place, not a conversation
partner). Flagged, not classified: the connect-at-launch reversal.
Verdict proposed: FILE-AS-IS as one intent. Verdict adopted: SPLIT into two.
Grade: the routing did not survive unchanged — the human split the one user
moment into two shippable intents so the built-in default can land before the
discovery behaviour is touched.

Addendum, 2026-09-17: "markdown returned from the model is formatted in
chats" (the maintainer, mid-session) — one capability, one intent
(itd-2609170836331240, `builds_on` the trunk); FILE-AS-IS, routing adopted
without change. Interaction flagged for the effects intent (both draw the
reply's `Text`), recorded in the specs rather than as a link.

## 2026-09-18 — the iPad client

Proposal (the maintainer): a native iPad app for the chat client, using an
Apple on-device model if the iPad has one, else a Gropius server on the local
network.

| Part | Type | Home |
| --- | --- | --- |
| A native iPadOS 27 chat client sharing the Mac client's code | capability | itd-2609180943290800, `builds_on` the trunk |
| The iPad's own model when it exists | promised | `refines` itd-2609170718438919 (same framework, same availability check) |
| A Gropius server from the local network when it does not | promised | `refines` itd-2609170718430553 (same picker and browse) |
| Distribution: provisioning, TestFlight or the App Store | open question, flagged | the draft's Open Questions; gates the interview |

Verdict proposed: FILE-AS-IS as one intent with the distribution question
open. Verdict adopted: the same. Grade: the routing survived the human's
confirmation unchanged.

## 2026-09-18 — text size in Settings

Proposal (the maintainer): change font size in Settings.

| Part | Type | Home |
| --- | --- | --- |
| A text-size choice in the client's Settings | capability | one intent, `builds_on` the trunk |
| How it is applied | mechanism | the system's Dynamic Type sizes, set at the window's root |

Verdict proposed: FILE-AS-IS. Verdict adopted: the same; the maintainer chose
the whole window over the conversation alone. Grade: routing survived
unchanged. Reviews scaled to the blast radius: one architecture test, no
adversarial pass.

## 2026-09-18 — appearance in Settings

Proposal (the maintainer): change Light/Dark/System mode in Settings.

| Part | Type | Home |
| --- | --- | --- |
| An appearance choice in the client's Settings | capability | one intent, `builds_on` the trunk |
| How it is applied | mechanism | the preferred colour scheme at each scene's root; System is no preference |

Verdict proposed: FILE-AS-IS. Verdict adopted: the same; fully specified by
the ask. Grade: routing survived unchanged; reviews scaled to the blast
radius (one architecture test).

## 2026-09-18 — four items from the maintainer's manual test

| Part | Type | Home |
| --- | --- | --- |
| Thoughts render markdown | capability | itd-2609181104497297, `refines` itd-2609170836331240 (replies already render) |
| Bubbles too close to the edges | nitpick | a capture, fixed in the same change |
| Sidebar cards with icon, date, summary, highlight | capability | itd-2609181104490133 |
| A search bar over conversations | capability | itd-2609181104498312 |

Verdict proposed: FILE-AS-IS, three intents and one capture. Verdict adopted:
the same. Grade: routing survived; reviews scaled to the blast radius (one
architecture test each).

## 2026-09-19 — per-client identity and encrypted interactions

Proposal (the maintainer): each client instance records a unique hash/id to
register with the server; each interaction is then encrypted via that hash.

| Part | Type | Home |
| --- | --- | --- |
| A client pairs once; Alice sees and revokes paired clients | capability | itd-2609182357325215, `builds_on` the trunk and the Gropius offer; refines the API-key intents |
| Traffic encrypted | trust-boundary rule | adr-2609182357322050: TLS, pinned certificate, a keypair per client; plain HTTP kept for unpaired clients |
| "Encrypted via the id" | unsound as asked | corrected at routing: an identifier is not a key |
| Plain HTTP on the LAN | reversal, flagged | kept for unpaired OpenAI clients by the maintainer's choice |

Verdict proposed: SPLIT (intent + ADR). Verdict adopted: SPLIT, plain HTTP
kept. Grade: routing survived; the mechanism was corrected before filing.

## 2026-09-20 — a new logo of three shapes instead of four

Proposal (the maintainer): a new logo using three shapes instead of four,
used for the website (its current path, later the renamed one), the chat
roles, etc.

| Part | Type | Home |
| --- | --- | --- |
| One three-form mark (yellow triangle, red square, blue circle; the grey square dropped) on the server icon, the chat icon, the site hero and wordmark, the README | capability | itd-2609200827202340 |
| The three forms double as the marks of the chat client's three response styles | capability | a chat-styles intent not yet filed; `refines` adr-2609200729102059 (no person's name on a mode; Square, Circle, Triangle) — this intent ships the forms, that one uses them |
| Website path moves from the current family name to Dessau | plumbing of the rename | adr-2609200729102059 consequences; the rename branch, not this intent |
| Committed .icns art regenerated by hand (`make icon`, librsvg) with its source sha | plumbing | Makefile as it stands; done in the change that lands the mark |
| The site hero's alt text names four forms | docs | corrected in the same change |

Verdict proposed: FILE-AS-IS, one intent, two parts routed away. Verdict
adopted: filed at the maintainer's ask in the same turn; the two routed parts
stand unless the maintainer says otherwise. Grade: pending the maintainer's
confirmation of the routing.

## 2026-09-20 — rethink the selection of server models in the chat client

Proposal (the maintainer): the current picker is not intuitive; a welcome
sheet in the style of the Apple Music splash in which the person must select
the model, the on-device model as an option and as the fallback, preferences
selectable from the models of every server found. A SOTA pass was run first
(`research/notes/2026-09-20-model-picker-sota.md`).

| Part | Type | Home |
| --- | --- | --- |
| Composer pop-up labelled with the model in use, sections per server, on-network offer line, default plus fallback plus favourites, per-chat choice, announced fallback, three availability states, menu-bar mirror | capability | itd-2609200829199959, `builds_on` itd-2609170718438919 and itd-2609170718430553 |
| The same server listed twice | bug | iss-2609200822241060, captured before filing |
| A must-choose welcome sheet before the first message | reversal of the shipped "answers out of the box" promise, flagged | declined at routing on the SOTA evidence; open question on the draft for the maintainer |
| A ranked preference list across servers | narrowed at routing, flagged | one default, one fallback, favourites; open question on the draft |
| A stable server identity in the TXT record for deduplication | server-side plumbing, possibly its own intent | open question on the draft; touches `internal/discovery` |
| Reading the on-device availability reasons | conflict between two shipped promises | iss-2609181116081228, decided at the interview |

Verdict proposed: SPLIT (one intent, one capture, two flagged reversals, one
server-side part). Verdict adopted: filed at the maintainer's ask in the
SOTA shape with the maintainer's original shape recorded as open questions.
Grade: pending the maintainer's confirmation of the routing and of the two
flags.

## 2026-09-20 — three answer styles in the top right of the chat window

Proposal (the maintainer, on the picker mock): the top right corner holds
the answer styles, Square, Circle and Triangle, once the picker moved to
the composer pill.

| Part | Type | Home |
| --- | --- | --- |
| Three styles as a segmented control of glyphs, per-conversation, default in Settings, glyph on each reply's label | capability | itd-2609200850330402, `builds_on` itd-2609200827202340 (the forms) |
| No person's name on a mode; shapes from the mark | rule already recorded | adr-2609200729102059, decision 5; this draft `refines` it |
| The style's prompt travels as the client's system message | plumbing | the shipped merge rule adr-2609061610102325; nothing new on the wire |
| The composer pill and the picker's place | decided | itd-2609200829199959, DECISIONS.md 2026-09-20 |

Verdict proposed: FILE-AS-IS, one intent. Verdict adopted: filed at the
maintainer's "yes" on the mock, and confirmed at the interview the same day
(two adversarial reviews, both: one intent, keep the links). Grade: routing
survived; the rule it rests on was written before the capability, which is
the order the ADR intended. The reviews changed the mechanism (no system
message) and the default (Square), not the routing.

## 2026-09-20 — a background image or pattern for chats

Proposal (the maintainer): the user selects a background image or pattern
for chats; selectable are the macOS system backgrounds.

| Part | Type | Home |
| --- | --- | --- |
| A per-conversation background chosen from built-in backgrounds, colours, gradients or the photo library; a default in Settings; None | capability | itd-2609200854205251, `builds_on` itd-2609181102147562 (appearance is Bob's to choose) |
| Bubbles stay legible on a picture in light and dark | plumbing with a contrast test | the same intent; interacts with iss-2609200818340380 (bubble colours) |
| A background never leaves the Mac | already covered | the client's existing promise that only the conversation goes to a server; restated in the press release, no new ADR |
| Which "system backgrounds" (Desktop wallpapers vs Messages-style backgrounds) | open question | the draft; decided at the interview |
| The iPad's bundled set and photo picker | scope condition | the draft |

Verdict proposed: FILE-AS-IS, one intent with four open questions. Verdict
adopted: filed at the maintainer's ask. Grade: pending the maintainer's
confirmation of the reading of "system backgrounds".

## 2026-09-20 — a fully redesigned server UX

Proposal (the maintainer): a fully redesigned server UX using SOTA
principles, modularity, and SOTA design principles, also in terms of a modern
look and feel across device screens.

| Part | Type | Home |
| --- | --- | --- |
| A modern look and feel across window sizes: tokens, adaptive layout, focus and touch, the tabs contract | capability | itd-2609201315575657, `builds_on` itd-2609100519003748 and the mark |
| "Fully redesigned" information architecture by task | already an intent | itd-2609100519003748, planned the same morning (inventory, re-home, accessibility bars) |
| Modularity of the panel's script | plumbing | the new intent's approach: native ES modules, no build step |
| SOTA principles as the basis | evidence | `2026-09-20-control-panel-ui-stack-sota.md`, run before planning |
| The menu-bar menu and the terminal verbs' output share the panel's vocabulary | capability, widened at the interview | the same intent (the maintainer chose all three surfaces) |
| Reachable from an iPad or a phone | declined | loopback-only stands (adr-2609091123526871) |

Typed links: `refines` itd-2609100519003748 and itd-2609200823520756; a
possible `reverses` of the morning's no-build decision was flagged and, after
the research pass, NOT taken — the decision was extended with a written
trigger for a vendored renderer.

Verdict proposed: SPLIT. Verdict adopted: SPLIT, with one widening (three
surfaces rather than the panel alone) and one refusal (device reach). Grade:
routing survived; the maintainer's clarifying questions ("what is a
framework, why did we decide against it") were answered before the stack
question was put, and the research pass changed nothing in the routing.
## 2026-09-20 — a copy button per chat response

Proposal (the maintainer): a copy button per chat response.

| Part | Type | Home |
| --- | --- | --- |
| A visible copy button on every reply, reusing the pasteboard code the context menu already has | capability | itd-2609201332162031, `builds_on` itd-2609151701196720 (the native client) |
| The context-menu Copy stays as it is | already covered | the shipped client; the draft says so |
| Placement, what is copied (text, never the reasoning; rendered or raw), a "Copied" confirmation | open questions | the draft; decided at the interview |

Verdict proposed: FILE-AS-IS, one intent with three open questions. Verdict
adopted: FILE-AS-IS, confirmed by the maintainer. Grade: the initial routing
survived unchanged.

## 2026-09-20 — key-term links in replies

Proposal (the maintainer): an option in Settings to scan responses for key
terms and turn them into links that ask "Tell me more about <term>." as a
follow-up.

| Part | Type | Home |
| --- | --- | --- |
| Term links in replies that send a follow-up question, behind a Settings switch | capability | a new intent, `builds_on` itd-2609151701196720 |
| How terms are found (on-device tagger vs model-marked), link styling against the model's own links, send-or-prefill, the iPad | open questions | the draft |

Verdict proposed: FILE-AS-IS. Verdict adopted: HOLD — the maintainer
changed their mind: no automatic links, because every link embedded in a
response should come from the model; the shape they want instead is
optional follow-up questions offered for clicking. Grade: the initial
routing did not survive; the proposal itself was withdrawn.

## 2026-09-20 — follow-up questions the model offers

Proposal (the maintainer, in place of the held key-term scan): follow-up
questions are requested from the model when a Settings option is on, and
offered for clicking, so every link in a response comes from the model.

| Part | Type | Home |
| --- | --- | --- |
| Optional follow-up questions asked of the model and offered under each reply, behind a Settings switch off by default; a click sends the question as Bob's turn | capability | itd-2609201338120342, `builds_on` itd-2609151701196720 and itd-2609200850330402 (the wire shape the styles decided) |
| No client-side scanning turns reply text into links | standing stance | the press release states it; a decision line if it recurs |
| Wire marking, the Square style's "no questions back", storage, the bridge and the on-device model | open questions | the draft |

Verdict proposed: FILE-AS-IS. Verdict adopted: filed at the maintainer's
ask, replacing the held proposal above. Grade: routing survived.

## 2026-09-20 — up to three optional follow-up questions to click

Proposal (the maintainer): an optional up to three follow-up questions the
user can click on instead of typing.

| Part | Type | Home |
| --- | --- | --- |
| Up to three optional follow-ups offered for clicking | duplicates | itd-2609201338120342, filed minutes earlier from the same conversation; its seeded criteria already say up to three, sent as the person's own turn |

Verdict proposed: HOLD, a duplicate of the draft just filed. Verdict adopted:
nothing filed; the number three is already in the earlier draft. Grade:
routing survived (duplicate recognised, no second record).

## 2026-09-20 — store/export an entire conversation

Proposal (the maintainer): an option to store/export an entire conversation.

| Part | Type | Home |
| --- | --- | --- |
| Export a whole conversation to a file Bob chooses (save panel on the Mac, share sheet on the iPad) | capability | itd-2609201355512965, `builds_on` itd-2609151701196720 |
| "Store": the client already keeps every conversation and reopens it | already covered | the shipped client; the draft says so |
| Format, what travels with the turns, an all-conversations archive | open questions | the draft |
| Distinct from the server-side transcript recording itd-2609091707499248 | stated in scope | the draft |

Verdict proposed: FILE-AS-IS. Verdict adopted: filed, confirmed by the
maintainer. Grade: routing survived.

## 2026-09-20 — Brave search

Proposal (the maintainer): add Brave search API functionality; research
whether the key belongs on the client or the server.

| Part | Type | Home |
| --- | --- | --- |
| Bob's own Brave key in Dessau Chat's Keychain; the client runs the search loop, shows Brave attribution and sources, stores answers and links, never snippets | capability | itd-2609201407580721, `builds_on` itd-2609151701196720; `refines` the 2026-09-10 ideate verdict with the client-side shape it did not weigh |
| An opt-in MCP search sidecar beside the server with Alice's key, granted per paired client (some users, not all) | capability | itd-2609201407587936, `builds_on` the pairing intent and the client-side draft; carries the 2026-09-10 reframing |
| The gateway holds no key, adds no outbound host, never reads tool messages, relays them untouched | trust-boundary rule | already held by adr-2609061503319212 and adr-2609061610102325; two passthrough tests owed, as a criterion |
| Key custody and who is Brave's Customer | research | 2026-09-20-brave-search-key-custody-sota.md |

Verdict proposed: FILE-AS-IS, one intent (client-held). Verdict adopted:
SPLIT into two — the maintainer wants the sidecar too, shared with some
paired clients rather than all. Grade: the routing widened by one record at
the maintainer's confirmation.

## 2026-09-20 — the number of follow-ups clicked per model over time

Proposal (the maintainer): record the number of follow-ups clicked per model over time.

| Part | Type | Home |
| --- | --- | --- |
| A count of clicked follow-ups per model per day in the statistics store, shown in the Statistics tab | capability | itd-2609201824151758, `builds_on` itd-2609201338120342 and the statistics store itd-2609061521102742 |
| One request mark the gateway reads and counts | trust-boundary read, no ADR reversed | the draft's criteria; security review at build |
| The bridge and the on-device model send no mark | scope condition | the draft |

Verdict proposed: FILE-AS-IS with the location open. Verdict adopted: filed
with the location decided (the server's statistics). Grade: routing survived,
narrowed by the maintainer's choice of location.

## 2026-09-20 — the two queue drafts: a place in line, and a fair turn

Proposal (the maintainer, as two quoted-text seeds): Bob sees his place in
the queue while he waits; every person on the network gets a fair turn at a
busy model.

| Part | Type | Home |
| --- | --- | --- |
| A client tags its request and asks the server where that request stands; Dessau Chat shows the answer after a threshold | capability | itd-2609202108185678, `builds_on` the eviction-grace intent and the pairing intent; spec spc-2609202141358048 |
| A turn queue at a loaded model: paired clients by least debt, the unpaired class from what is left, contention logged and counted | capability | itd-2609202108180709, same `builds_on`; spec spc-2609202154054806 |
| Who counts as one client for scheduling — the identity question both reviews held for a decision record | ADR | adr-2609202200197975, refining the pool's "an address is not a client" rule and iss-2609070252377294 rather than reversing them |
| A hard per-client rate limit as a later switch against a runaway client | future-work seed | iss-2609202200256227, captured after the interview, declined for both intents |

Verdict proposed: the reviewers' routing — the two records plus a decision
record for identity, and the rate limit captured as a seed. Verdict adopted:
FILE-AS-PROPOSED, confirmed by the maintainer at the interview; the seed was
not captured at the interview and is captured now. The interview also chose
the mechanism the design review's most severe finding turned on (the client
asks the server about its own tagged request, not SSE comment lines) and
sequenced the fair turn first, because its ordering rule is what a place
means. Grade: routing survived; one capture landed late.

## 2026-09-21 — itd-2609211335097114, a stuck model never holds the server hostage

Filed as a seed from the live-box incident of the same day; two adversarial
reviews (design/feasibility, record discipline) before the interview, both
REVISE on the same ground: most of the seed was already shipped.

| part | type | home |
|---|---|---|
| A real request pre-empts idle work, including a loading model; the pin wins; a bounded park | capability | itd-2609211335097114 (planned, spc-2609211753023984) |
| A load the child has given up on fails in seconds; a failed model is not re-queued; the probe measures only chat models; the 503 names the holder | already shipped | PR 144, 0.9.3 — struck from the intent, cited in Why This Matters |
| The probe's own unload ignores pins | bug | iss-2609211754251373 |
| A silent child (answers /health, no traceback, no completion) still costs the readiness timeout | bug, reversal-flagged | iss-2609211754253878 |
| Which holders are preemptible; a loading soft-only entry may be torn down; no HTTP client can carry a soft hold | decision (candidate ADR) | the 2026-09-21 decision line (supersedes one clause of 2026-09-11, reverses 2026-08-02 for idle holders); promote to an ADR when the spec's design review confirms the lock order |
| The card's "for how long" | plumbing | criterion 7 of the intent, not its own record |

Verdict proposed by the reviewers: REVISE — strike the shipped claims, scope
to the residual, name the reversals. Verdict adopted: the full residual,
FILE-AS-REVISED, confirmed by the maintainer at the interview; the reversals
were put to the maintainer as choices (pool-side soft hold; tear down a
loading idle-held model; the pin wins) and each was chosen, not assumed. Two
captures landed at the interview, not late. Grade: routing survived; the
reviewers' "candidate ADR" is carried as a decision line pending the design
review, which is a deferral of the ADR, not a disagreement with the routing.

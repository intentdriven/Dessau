# SOTA — choosing the model that answers, in a macOS chat client (2026-09-20)

Research for the model-picker intent filed the same day, run by a research
agent briefed to break the maintainer's suggestion (a must-choose welcome
sheet in the style of the Apple Music splash, the on-device model as option
and fallback, a ranked preference list across servers). Primary sources: the
Human Interface Guidelines (onboarding, launching, sheets, pop-up buttons,
toolbars, generative AI), Nielsen Norman Group studies, the Foundation
Models availability API and WWDC25 session 286, Apple's own model choice in
Shortcuts and Xcode 26, and the pickers of thirteen peer clients. Ranked;
the peer table, the counter-arguments that survived and the one-design
recommendation follow.

## Findings

1. **Do not gate the first message behind a model choice.** NN/g (70 users,
   4 apps, between-subjects): a tutorial at launch gave no better task
   success (91% vs 94%, p=0.443) and made tasks feel harder (SEQ 4.92 vs
   5.49, p=0.047). The default effect (Johnson and Goldstein, Science 2003)
   means most people keep what is preselected, so preselect the option that
   always works. HIG Onboarding: "fast, fun, and optional"; HIG Launching:
   "launch instantly". NN/g's modal criteria: a modal is for information
   critical to continuing; with a working on-device model the choice is not.
   No surveyed client forces a choice when a usable model exists; Msty forces
   a setup path only because it has no model until one is chosen.
   <https://www.nngroup.com/articles/mobile-tutorials/>,
   <https://www.nngroup.com/articles/modal-nonmodal-dialog/>,
   <https://developer.apple.com/design/human-interface-guidelines/onboarding>.
2. **The day-to-day picker lives at the composer.** Claude: the model appears
   next to Send, changes apply from the next reply. Zed: selector on the
   message editor with starred favourites. Xcode 26: chosen per conversation,
   the composer placeholder shows the model in use. ChatGPT: tier picker at
   the top of the conversation. HIG pop-up buttons: a flat list of mutually
   exclusive options with a useful default and a label that predicts the
   options. HIG toolbars: every toolbar item also exists as a menu command.
   The current top-right toolbar popover is the outlier among peers.
   <https://support.claude.com/en/articles/8664678-change-the-model-effort-and-thinking-settings>,
   <https://zed.dev/docs/ai/agent-panel>,
   <https://developer.apple.com/documentation/xcode/writing-code-with-intelligence-in-xcode>,
   <https://developer.apple.com/design/human-interface-guidelines/pop-up-buttons>.
3. **Fallback is announced in the transcript, never silent, never
   mid-stream.** Cursor's silent model swaps produced a large complaint
   corpus ("a violation of users' trust") and were replaced by a prompt.
   VS Code Copilot shows "Routed to <model>" inline. Routers disclose the
   served model in the response. Apple's own precedents: Shortcuts' Use
   Model action offers to switch to the cloud model rather than switching;
   Siri has a "Confirm ChatGPT Requests" toggle. HIG Generative AI: let
   people know when and where the app uses AI. Direction matters: server to
   on-device is privacy-safe and may announce-and-continue; on-device to
   server must ask first.
   <https://forum.cursor.com/t/cursor-is-changing-models-without-warning-and-sometimes-mid-task/159208>,
   <https://docs.github.com/en/copilot/concepts/models/auto-model-selection>,
   <https://developer.apple.com/design/human-interface-guidelines/generative-ai>.
4. **Group by location, on-device first, one row per server.** Shortcuts'
   picker is three rows: On This Mac, Private Cloud Compute, ChatGPT, plus
   Ask Each Time. Xcode 26 splits Locally Hosted from Internet Hosted. The
   Home app keeps unreachable accessories listed as "No Response"; Finder
   keeps locked shares listed and prompts on click. Bonjour resolves a name
   conflict by renaming ("(2)") and a restart without deregistering leaves a
   stale twin, so servers are deduplicated by a stable identity the product
   controls, never by display name. Open WebUI's merge of identical model
   IDs across connections is the warning that a model name needs its server.
   <https://club.macstories.net/posts/automation-academy-apple-intelligence-gpt-5-and-the-use-model-action-in-shortcuts>,
   <https://developer.apple.com/library/archive/qa/qa1311/_index.html>,
   <https://support.apple.com/en-us/102056>.
5. **One default plus one fixed fallback plus favourites, not a ranked
   list.** Apple removed drag-to-reorder for preferred Wi-Fi networks in
   Ventura and replaced it with behaviour scoring. FaceTime's camera choice
   is automatic plus one override. Raycast: a default per surface and
   per-command overrides. Zed: a default model and starred favourites.
   Ranked fallback arrays exist in developer APIs (OpenRouter, LiteLLM),
   never in surveyed consumer UI. A two-level list (preferred server, then
   on-device) is defensible; a general ranking has no consumer precedent.
   <https://support.apple.com/en-qa/102169>,
   <https://manual.raycast.com/ai/ai-commands>,
   <https://openrouter.ai/docs/guides/routing/model-fallbacks>.
6. **Per-conversation choice inherits the global default.** Claude and Xcode
   choose per conversation; Open WebUI's in-chat "Set as default" exists,
   and its issue tracker shows the confusion when a new chat inherits the
   last-viewed chat's model instead of the default. "New chat starts on the
   default" is a tested invariant.
   <https://github.com/open-webui/open-webui/issues/17439>.
7. **Three unavailability reasons, three UI states.** Foundation Models
   reports deviceNotEligible (permanent), appleIntelligenceNotEnabled (a
   setting) and modelNotReady (transient); WWDC25 says to adjust the UI to
   each. Practitioner consensus: not eligible, hide or disable the row with
   one line; not enabled, a row with a link to switch it on; not ready,
   "Preparing…" and a silent retry. Nothing available anywhere is an
   empty-state view in the transcript area, not a modal.
   <https://developer.apple.com/documentation/foundationmodels/systemlanguagemodel/availability-swift.enum>,
   <https://developer.apple.com/videos/play/wwdc2025/286/>.
8. **The Apple Music welcome sheet is a splash, not a decision.** It offers
   Continue, Start Listening and a tour, and asks nothing. Borrowing its
   form for a required choice keeps the look and inverts the purpose.

## Peer clients

| Client | Picker lives | Auto | First-run forced choice | Grouping | Unavailability shown |
|---|---|---|---|---|---|
| ChatGPT | top of conversation | yes, default | no | n/a | hidden picker on the free tier |
| Claude | next to Send | no | no | n/a | change applies to the next reply |
| Ollama app | dropdown in chat | no | no | local and cloud in one list | minimal |
| LM Studio | top bar | no | skippable starter download | flat | "No model loaded" |
| Msty | per-chat dropdown | no | yes, no model exists otherwise | mixed | not documented |
| Jan | chat window | no | no | local and cloud | not documented |
| Open WebUI | header, "Set as default" | no | no | merged IDs, prefix per connection | random load balance |
| Cursor | composer | yes | no | flat, per-model toggles | was silent, now a prompt |
| VS Code Copilot | composer | yes | no | flat | "Routed to X" per reply |
| Zed | message editor, favourites | no | no | per provider | settings |
| Xcode 26 | new-conversation pop-up | no | no | locally or internet hosted | not documented |
| Raycast AI | settings default, per command | no | no | checkbox list | n/a |
| Shortcuts Use Model | action parameter | Ask Each Time | per action | three rows | offers the switch |

## Counter-arguments that survived

- **The client exists to demonstrate the server, and an on-device default
  risks nobody touching it.** Real. The mitigation the HIG prefers over
  onboarding is a contextual tip: when servers first appear, a dismissible
  line above the composer offers one of them in one click.
- **A must-choose sheet guarantees the person knows a choice exists.** The
  same knowledge comes from a labelled composer control plus the per-reply
  model label the client already has, without the measured penalty.
- **Some people want a rank.** The two-level shape, preferred server then
  on-device, covers the case that matters, the server being down.
- **Apple does not say when Siri goes to Private Cloud Compute.** That is
  opacity not to copy; the Cursor record shows what silent switching costs.

## Not worth adopting

A modal must-choose welcome sheet; a drag-to-rank list; an Auto router
across servers (Copilot and Cursor have it for vendor capacity and cost,
which this product does not have, and it generates the trust complaints);
silent or mid-stream fallback; a toolbar-only picker; deduplicating servers
by display name.

## Recommendation

No welcome sheet. A composer-adjacent pop-up button labelled with the model
in use opens a menu: On this Mac first, availability-aware; one section per
discovered server, lock where a key is needed, greyed with "No response"
where unreachable, deduplicated by server identity, each listing its chat
models; a footer with Set as Default and Manage Servers. The same menu under
a Model menu in the menu bar; the toolbar item at most a duplicate. First
launch opens a working chat on the on-device model; when servers first
appear, a dismissible line above the composer offers one in one click.
Settings holds two controls, the default model and "if it is unavailable,
answer on this Mac", on by default; starred favourites pin models to the top
of the menu. A new chat starts on the default; a change in a chat affects
that chat; Set as Default promotes it. Fallback only between turns, written
into the transcript as a row with a retry; server to on-device announces and
continues, on-device to server always asks. The three unavailability reasons
get three states; nothing available anywhere is an empty state, not a modal.

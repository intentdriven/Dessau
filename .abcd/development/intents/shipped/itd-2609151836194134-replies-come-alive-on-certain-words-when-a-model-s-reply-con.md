---
id: itd-2609151836194134
slug: replies-come-alive-on-certain-words-when-a-model-s-reply-con
spec_id: spc-2609170842094071
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170836331240]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Replies come alive on certain words

## Press Release

Replies come alive on certain words. When a model's reply contains one of a
small set of words — congratulations, well done, warning, careful, wow,
amazing — that word animates once, in place, with the system's own text
rendering: a shimmer, a shake, a bounce, the way Messages brings a word to
life, and only on the words that earned it. Bob switches effects off in
Settings. A reply is never altered, only drawn differently: what the model
said is what is copied or saved.

## Why This Matters

A reply is text a person reads, and a native app is allowed a little
delight where the system provides the means. The set is small and fixed so
the effect stays a surprise rather than a wallpaper.

## Mechanism

We expect the effect to be drawn on the words alone because SwiftUI's
`TextRenderer` hands a `Text`'s glyph runs to the app, and a custom
`TextAttribute` on the matched range lets the renderer animate only those
runs. The renderer is applied only while the effect plays; the block then
returns to the plain, selectable `Text` the formatted-replies intent draws.
What would show this wrong: a `TextRenderer` that cannot animate one run of
a `Text` built from an attributed string, in which case the word is drawn as
its own `Text` for the duration of the effect.

## Scope Conditions

- A fixed list of English words, whole-word and case-insensitive, held in <!-- cond: cond-2609170842093005 -->
  one place in the client and named on the Settings page; one on/off switch,
  on by default. The maintainer's decision at the interview, 2026-09-17.
- An effect plays once, when the reply finishes streaming, never on every <!-- cond: cond-2609170842090786 -->
  redraw.
- The 27 client only, building on the formatted-replies intent: the word <!-- cond: cond-2609170842097968 -->
  match runs per rendered block, over the block's plain text.
- The renderer is client-owned drawing, so the no-styling test gains one <!-- cond: cond-2609170842097664 -->
  named exception for the effect renderer file, with its reason in the
  test's own comment.
- "Once" is a fact about the message, not the row: the played flag lives on <!-- cond: cond-2609170842090283 -->
  the model keyed by message id, because the transcript's rows are recreated
  on scroll.

## Acceptance Criteria

- Given effects are on, when a reply containing "congratulations" finishes,
  then that word shimmers once, the rest of the reply is still, and the
  effect does not play again on scroll or redraw.
- Given a reply with an animated word, when Bob copies or saves it, then the
  text is the model's text unchanged.
- Given the switch is off, when a reply with a listed word arrives, then
  nothing animates.
- Given a listed word inside another word ("forewarning"), when the reply
  arrives, then nothing animates; given "Warning:" with a capital and a
  colon, then it does.
- Given the client's source, when the architecture tests run, then the word
  list appears once, and it is the list the Settings page names.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-1db24927e6a0 -->
Fidelity review — receipt rcp-1db24927e6a0 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:886df11932f363fc5fb7b7307197f148f664dd58634ff4b406697a101faddf3c
Input attestations: diff:acacd07..210ce66 (PR 58, merged; audited on origin/main at 400ccd7) -- client internal/archtest install.sh .github README.md docs CHANGELOG.md@sha256:da0c18bfb59c2c67e4e35a363d5d29a445eb86969c738f5c943f26fd3167cbdb;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 3 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — NOT_MET: The renderer does animate only the tagged runs, but the trigger is read at the wrong moment: the row latches `playing` from `model.effectsToPlay` on the reply block's first appearance — which is the first streamed token, since the block is shown as soon as displayText is non-empty — while `queueEffects` inserts the message id only in the stream's `defer`, after that latch is set; so a reply that finishes does not animate on the live row, and the id left behind in the set is consumed by a row recreated on scroll, which is the play-again the criterion forbids. The maintainer's own record lists 'the effect animation as drawn' as a manual check still owed, so nothing observed contradicts this reading.
  evidence: client/GropiusChat/GropiusChat.swift:801 — "queueEffects(for: messageID, in: convoID)"
  evidence: client/GropiusChat/GropiusChat.swift:1269 — "let due = model.effectsToPlay.contains(message.id)"
  evidence: client/GropiusChat/GropiusChat.swift:1175 — "@State private var playing: Bool?"
  evidence: client/GropiusChat/Effects.swift:78 — "guard let effect = run[EffectAttribute.self] else {"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: ... the effect animation as drawn"
- ac-2 — MET_WITH_CONCERNS: Copy puts `message.text` on the pasteboard verbatim and the effect path only draws — `EffectText` takes the rendered AttributedString and never writes back to the message — so what is copied or saved is the model's text; the named concern is that for the effect's ~1.3s the block is drawn as concatenated Texts with no `.textSelection(.enabled)`, so hand-selection is unavailable during the effect, which is the falsifier the intent's own Grounds names ('wrong if ... the renderer costs the reply its selection').
  evidence: client/GropiusChat/GropiusChat.swift:1221 — "NSPasteboard.general.setString(message.text, forType: .string)"
  evidence: client/GropiusChat/Effects.swift:122 — "Text(text).textSelection(.enabled)"
  evidence: client/GropiusChat/Effects.swift:125 — ".textRenderer(EffectRenderer(progress: progress))"
- ac-3 — MET: `queueEffects` is guarded on `effectsEnabled` before any match runs, so with the switch off no id is ever queued, the row's latch resolves to false and every block is drawn as the plain selectable Text; the toggle itself is the one `@AppStorage("effectsEnabled")` setting, on by default.
  evidence: client/GropiusChat/GropiusChat.swift:853 — "guard effectsEnabled,"
  evidence: client/GropiusChat/GropiusChat.swift:1296 — "Text(s).textSelection(.enabled)"
  evidence: client/GropiusChat/GropiusChat.swift:427 — "@AppStorage("effectsEnabled") var effectsEnabled: Bool = true"
- ac-4 — MET_WITH_CONCERNS: The matcher searches case-insensitively and admits a hit only when the characters on both sides are non-letters, so 'forewarning' is rejected (preceding 'e' is a letter) and 'Warning:' is admitted (trailing ':' is not), and the shipping record says a harness on the build Mac matched effect words whole and case-insensitively; the concerns are that no test in this repository exercises the boundary behaviour — the architecture test only greps the source for the `.caseInsensitive` option — and the criterion's second half ('then it does' animate) rests on the trigger path ac-1 finds broken.
  evidence: client/GropiusChat/Effects.swift:46 — "while let r = text.range(of: word, options: [.caseInsensitive], range: search..< text.endIndex) {"
  evidence: client/GropiusChat/Effects.swift:49 — "if before.map({ !$0.isLetter }) ?? true, after.map({ !$0.isLetter }) ?? true {"
  evidence: internal/archtest/chat_client_native_test.go:189 — "if !strings.Contains(effects, `options: [.caseInsensitive]`) {"
  evidence: .abcd/work/DECISIONS.md:277 — "matched effect words whole and case-insensitively"
- ac-5 — MET_WITH_CONCERNS: TestChatClientEffectWordsLiveInOnePlace exists and passes here (go test ./internal/archtest/ -run TestChatClientEffectWordsLiveInOnePlace: PASS), reading EffectWords.list from Effects.swift and asserting the Settings page renders EffectWords.settingsSentence, which is the same constant; the concern is that 'appears once' is enforced by scanning the other Swift files for a single sentinel word ('congratulations') rather than for every word in the list, so a duplicate of 'warning' or 'wow' elsewhere in the client would pass.
  evidence: internal/archtest/chat_client_native_test.go:174 — "func TestChatClientEffectWordsLiveInOnePlace(t *testing.T) {"
  evidence: internal/archtest/chat_client_native_test.go:192 — "if !strings.Contains(all["GropiusChat.swift"], "EffectWords.settingsSentence") {"
  evidence: internal/archtest/chat_client_native_test.go:196 — "if name != "Effects.swift" && strings.Contains(src, `"congratulations"`) {"
  evidence: client/GropiusChat/GropiusChat.swift:1366 — "Text(EffectWords.settingsSentence)"

Gap audit:
- honoured:
  - A small fixed list of six words, each earning one of three effects, held in one place in the client
    evidence: client/GropiusChat/Effects.swift:20 — "static let list: [(word: String, kind: EffectKind)] = ["
    evidence: client/GropiusChat/Effects.swift:13 — "enum EffectKind: String, CaseIterable { case shimmer, shake, bounce }"
  - Bob switches effects off in Settings, and the Settings page names the words from that same constant
    evidence: client/GropiusChat/GropiusChat.swift:1365 — "Toggle("Animate certain words", isOn: $model.effectsEnabled)"
    evidence: client/GropiusChat/GropiusChat.swift:1366 — "Text(EffectWords.settingsSentence)"
  - The effect is drawn on the matched words alone, through a TextRenderer reading a custom TextAttribute on those runs
    evidence: client/GropiusChat/Effects.swift:60 — "struct EffectAttribute: TextAttribute {"
    evidence: client/GropiusChat/Effects.swift:78 — "guard let effect = run[EffectAttribute.self] else {"
  - A reply is never altered, only drawn differently: Copy hands over the model's text unchanged
    evidence: client/GropiusChat/GropiusChat.swift:1221 — "NSPasteboard.general.setString(message.text, forType: .string)"
  - The no-styling architecture test gains exactly one named exception for the effect renderer file, with its reason in the test's own comment
    evidence: internal/archtest/chat_client_native_test.go:49 — ""Effects.swift": "the text-effects renderer draws glyphs itself; that is its purpose","
    evidence: internal/archtest/chat_client_native_test.go:40 — "with its reason: Effects.swift is a TextRenderer, which is client-owned"
  - The user-facing prose describes the shipped behaviour and the switch
    evidence: client/README.md:64 — "A few words (congratulations, well done, warning, careful, wow, amazing) animate once when a reply arrives; switch that off under Replies in Settings."
- diverged:
  - Promised: the word animates once when the reply finishes. Delivered: the row latches whether to animate at the reply block's first appearance (the first streamed token), while the id is queued afterwards in the stream's defer — so the decision is read before the fact it depends on exists, and the stale id can instead fire on a later row recreation.
    evidence: client/GropiusChat/GropiusChat.swift:801 — "queueEffects(for: messageID, in: convoID)"
    evidence: client/GropiusChat/GropiusChat.swift:1269 — "let due = model.effectsToPlay.contains(message.id)"
  - Promised: a played flag on the model keyed by message id. Delivered: an inverted due-to-play set on the model plus a per-row @State latch that is the operative flag, so the played fact is half on the row after all.
    evidence: client/GropiusChat/GropiusChat.swift:467 — "@Published var effectsToPlay: Set< UUID> = []"
    evidence: client/GropiusChat/GropiusChat.swift:1175 — "Latched on first appearance: whether this row plays the effect."
  - Promised: the word match runs per rendered block, over the block's plain text. Delivered: the renderer matches per block, but the gate that decides whether a reply animates at all matches over the whole reply's raw markdown text, so the two matches can disagree.
    evidence: client/GropiusChat/GropiusChat.swift:855 — "!EffectWords.matches(in: text).isEmpty else { return }"
    evidence: client/GropiusChat/GropiusChat.swift:1298 — "EffectText(text: s, matches: EffectWords.matches(in: String(s.characters)))"
  - Promised: the block returns to the plain, selectable Text the formatted-replies intent draws. Delivered: it does after ~1.3s, but while the effect plays the block is concatenated Texts with no textSelection, so selection is lost for the duration — the falsifier the intent's Grounds names.
    evidence: client/GropiusChat/Effects.swift:125 — ".textRenderer(EffectRenderer(progress: progress))"
    evidence: client/GropiusChat/Effects.swift:138 — "private var tagged: Text {"
- missing:
  - Any executed check that the effect plays as drawn — no Swift test target exists in client/, the architecture tests only read the source as text, and the maintainer's manual check of the animation is recorded as still owed.
    evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: ... the effect animation as drawn"
    evidence: internal/archtest/chat_client_native_test.go:21 — "matches, err := filepath.Glob(filepath.Join(root, "client", "GropiusChat", "*.swift"))"
  - An armed check that the effect plays once and never again on scroll or redraw — the once-only promise rests entirely on unexercised code, and nothing in the suite would redden if the latch ordering changed.
    evidence: internal/archtest/chat_client_native_test.go:174 — "func TestChatClientEffectWordsLiveInOnePlace(t *testing.T) {"
    evidence: client/GropiusChat/GropiusChat.swift:860 — "func effectStarted(_ messageID: UUID) {"

Scope-condition dispositions:
- cond-2609170842093005 — survived: The six English words live in one constant matched whole and case-insensitively, the Settings page names them from that same constant, and there is exactly one toggle, on by default.
  evidence: client/GropiusChat/Effects.swift:20 — "static let list: [(word: String, kind: EffectKind)] = ["
  evidence: client/GropiusChat/GropiusChat.swift:427 — "@AppStorage("effectsEnabled") var effectsEnabled: Bool = true"
  evidence: client/GropiusChat/GropiusChat.swift:1366 — "Text(EffectWords.settingsSentence)"
- cond-2609170842090786 — falsified: The assumption was that an effect plays once when the reply finishes streaming; the delivered code reads the play decision at the reply block's first appearance — the first token — and only queues the id afterwards in the stream's defer, so the play is not triggered at the finish and the leftover id is available to a later row recreation instead.
  evidence: client/GropiusChat/GropiusChat.swift:801 — "queueEffects(for: messageID, in: convoID)"
  evidence: client/GropiusChat/GropiusChat.swift:1269 — "let due = model.effectsToPlay.contains(message.id)"
  evidence: client/GropiusChat/GropiusChat.swift:1207 — "} else if !displayText.isEmpty { reply }"
- cond-2609170842097968 — narrowed: The work is confined to the 27 client and builds on the formatted-replies block rendering, and the renderer does match per rendered block over that block's plain text — but a second match, over the whole raw reply, decides whether anything animates at all.
  narrowing: Holds for the drawing path only: the per-block, plain-text match governs which runs are tagged, while the decision that a reply animates is made by matching the whole message's raw markdown text, so a word the blocks do not carry can still arm the effect and vice versa.
  evidence: client/GropiusChat/GropiusChat.swift:1298 — "EffectText(text: s, matches: EffectWords.matches(in: String(s.characters)))"
  evidence: client/GropiusChat/GropiusChat.swift:855 — "!EffectWords.matches(in: text).isEmpty else { return }"
- cond-2609170842097664 — survived: The no-styling test carries exactly one exemption, keyed by the file name, with the reason stated both in the map's value and in the test's own comment, and the test passes on this checkout.
  evidence: internal/archtest/chat_client_native_test.go:48 — "exempt := map[string]string{"
  evidence: internal/archtest/chat_client_native_test.go:49 — ""Effects.swift": "the text-effects renderer draws glyphs itself; that is its purpose","
  evidence: internal/archtest/chat_client_native_test.go:41 — "drawing by definition and is the text-effects intent's whole mechanism."
- cond-2609170842090283 — narrowed: A set keyed by message id does live on AppModel, but it is a due-to-play set consumed at a row's first appearance, and the flag that actually decides the row's branch is the row's own @State latch — so the fact is only about the message up to that first appearance.
  narrowing: Holds only until a row first draws the reply: from that moment the play/don't-play fact lives in the row's @State `playing` latch, and an id queued after that latch was taken stays in the model's set unclaimed, so the message-level fact no longer governs what the row draws.
  evidence: client/GropiusChat/GropiusChat.swift:467 — "@Published var effectsToPlay: Set< UUID> = []"
  evidence: client/GropiusChat/GropiusChat.swift:1175 — "@State private var playing: Bool?"
  evidence: client/GropiusChat/GropiusChat.swift:1271 — "if due { model.effectStarted(message.id) }"
## Grounds

- pursued: we expect a once-only effect on a handful of words to read as delight rather than noise, because the list is fixed and small and the system's own renderer draws it in place; wrong if people switch it off, or if the renderer costs the reply its selection

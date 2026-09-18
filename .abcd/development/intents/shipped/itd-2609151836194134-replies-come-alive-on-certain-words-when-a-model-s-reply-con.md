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

<!-- abcd-review: OWED receipt=rcp-1db24927e6a0 -->
Fidelity review OWED (receipt rcp-1db24927e6a0).

## Grounds

- pursued: we expect a once-only effect on a handful of words to read as delight rather than noise, because the list is fixed and small and the system's own renderer draws it in place; wrong if people switch it off, or if the renderer costs the reply its selection

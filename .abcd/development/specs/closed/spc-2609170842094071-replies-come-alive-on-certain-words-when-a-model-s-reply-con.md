---
id: spc-2609170842094071
slug: replies-come-alive-on-certain-words-when-a-model-s-reply-con
intent: itd-2609151836194134
origin: researcher-authored
production_mode: hand-written
---
# Replies come alive on certain words

## Summary

A `TextRenderer` in a new file `client/GropiusChat/Effects.swift` animates
the glyph runs of a reply block that carry a custom `TextAttribute`
(`EffectAttribute`, carrying the effect kind); the attribute is put on the
matched words of the block's attributed string when the reply finishes, the
renderer is applied for the effect's duration (about a second), and then the
block returns to the plain selectable `Text` the formatted-replies intent
draws. The word list is one constant, `EffectWords`, mapping six words to
three effects (shimmer, shake, bounce); one Settings toggle, `effectsEnabled`,
on by default.

## Scope

In scope: `Effects.swift`; the `played` set on `AppModel` keyed by message id;
the Settings toggle; the exception named in the no-styling test for
`Effects.swift`; a `TestChatClientEffectWordsLiveInOnePlace` architecture test
that reads the constant and the Settings page's wording; the client README's
sentence on effects.

Out of scope: user-editable words (the maintainer's decision, 2026-09-17);
effects on the person's own messages; effects while streaming.

## Approach

When a reply's stream ends, `AppModel` matches `EffectWords` against the
reply's plain text as whole words, case-insensitively (a word boundary on
both sides, so "forewarning" does not match and "Warning:" does). If any
match and the toggle is on and the message id is not in `played`, the id
goes into `played` and the message row is told which ranges to animate. The
row applies `.textRenderer(EffectRenderer(kind:, progress:))` to the block
holding the range, with the range's runs tagged through
`Text.customAttribute`; a `TimelineView`-driven progress runs 0→1 over the
duration, after which the row drops the renderer. Rows are recreated on
scroll; because the flag is on the model, a recreated row draws plain text.

## How the acceptance criteria are met

1. Shimmers once on completion, not on redraw — the `played` set.
2. Copy is the model's text — the effect is drawing only.
3. Off switch — the toggle checked before matching.
4. Whole-word, case-insensitive — the boundary match; checked by hand and in
   the architecture test's sample strings.
5. One list, named on the Settings page — the architecture test.

## Verification

Suite green; `client/build.sh` builds; the manual check recorded.

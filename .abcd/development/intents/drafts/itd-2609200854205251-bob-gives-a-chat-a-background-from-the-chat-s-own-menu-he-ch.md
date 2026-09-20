---
id: itd-2609200854205251
slug: bob-gives-a-chat-a-background-from-the-chat-s-own-menu-he-ch
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609181102147562]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Bob gives a chat a background. From the chat's own menu he chooses a background for the transcript the way Messages on macOS 26 does: the Mac's built-in backgrounds, plain colours and gradients, or a picture from his own library, and the bubbles sit on it legibly whatever he picks, in light and in dark. The choice is per conversation, remembered, and never leaves the Mac: a background is a file the client reads, and nothing about it is sent to any server or model. Settings holds the default for a new chat, and None puts the transcript back on the plain window.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Bob gives a chat a background. From the chat's own menu he chooses a background for the transcript the way Messages on macOS 26 does: the Mac's built-in backgrounds, plain colours and gradients, or a picture from his own library, and the bubbles sit on it legibly whatever he picks, in light and in dark. The choice is per conversation, remembered, and never leaves the Mac: a background is a file the client reads, and nothing about it is sent to any server or model. Settings holds the default for a new chat, and None puts the transcript back on the plain window.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **Which "system backgrounds".** The maintainer's ask names macOS system backgrounds. Two readings: the Desktop wallpapers under the system's Desktop Pictures folder (large, some dynamic, read from a system path the client may not be entitled to under sandboxing), or the conversation backgrounds Messages ships on macOS 26 (built-in colours, gradients and dynamic backgrounds, plus the photo library). The draft is written for the second, the native precedent; the interview confirms.
- **Per conversation or global.** Messages does per conversation; the draft follows. A global-only choice is simpler and may be enough for a demonstrator.
- **Legibility on a picture.** Bubbles on a photo need a material behind them (the system's regular material) or a scrim, and the bubble colours from iss-2609200818340380 (accent-tinted user bubble, lighter tint for the model) must still pass contrast in light and dark. Which of the two mechanisms, and whether the model bubble keeps its tint or goes to a material, is for the interview.
- **The iPad.** The client also runs on the iPad (itd-2609180943290800), which has no Desktop Pictures; the built-in set must be bundled or drawn (gradients), and the photo library needs the limited-access picker.
- **Storage.** A chosen picture is copied into the client's own container and referenced from the conversation record; pre-1.0, a stored background that no longer resolves falls back to None without migration.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

---
id: itd-2609200829199959
slug: bob-picks-the-model-that-answers-him-from-where-he-types-gro
spec_id: spc-2609201011301946
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609170718438919, itd-2609170718430553]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob picks the model that answers him from where he types

## Press Release

Bob installs GropiusChat and opens it. A sheet welcomes him: a line saying
what the app is, a line saying that his Mac can already answer and that any
Gropius server on his network can too, and one button that closes it. It asks
him nothing and selects nothing. He closes it, types his question into the
pill at the foot of the window, and the answer comes back from his own Mac —
before he has chosen a model, found a server, or been told what a model is.
He never sees that sheet again.

At the start of the message pill, which floats over the transcript rather than
sitting in a bar beneath it, is the control that decides who answers. It
carries the name of the model in use and where it runs: **On this Mac**. It is
a hand's width from the words Bob is typing, and it is the same control in
every window and on the iPad. Clicking it opens a menu with **On this Mac**
first, then one section per Gropius server found on the network, each listing
that server's chat models under its own name: a lock where a server wants an
API key, a greyed row saying **No response** where a server has gone quiet,
and a server Bob has already paired with shown as paired. A server appears
once, however many names the network gives it.

The afternoon Alice switches her server on down the hall, one line appears
above Bob's composer: her server is on the network and serving four models,
with **Use it**, **See all**, and a way to dismiss it for good. One click and
his next reply comes from the large model on Alice's Mac. That line is the
whole of the tour, and it arrives at the moment it means something rather than
on the first launch, when it would have meant nothing.

Carol keeps her window narrow beside her editor. There the picker drops its
label and shows a small glyph for where the answer comes from — a chip for a
server, the Apple mark for this Mac — with a chevron, and the model's name in
the tooltip and to VoiceOver; the name is on every reply already. Nothing
moves on hover, so the same control works under a finger on the iPad. The top
right of the window is not the picker's: it carries the three answer styles,
which are their own intent.

Settings holds two choices rather than a list: the default model for a new
chat, and whether a chat falls back to this Mac when its server cannot be
reached, on by default. Stars in the menu pin the models Bob actually uses to
the top. A new chat starts on the default; changing the model inside a chat
changes that chat and nothing else; **Set as Default** promotes it. There is
no ranked list of servers to build and no list to maintain when one is
retired.

When Alice reboots her Mac in the middle of Bob's afternoon, nothing is lost
and nothing is silently swapped. The fallback happens between turns, never
inside a reply: his next message is answered on his own Mac, and a row in the
transcript says so and offers to retry on Alice's server. The reverse never
happens by itself — a conversation held on this Mac is never moved to somebody
else's machine without Bob being asked. Where this Mac cannot answer at all,
its row says which of the three reasons applies — not eligible, not switched
on, or not ready yet — and offers the fix where there is one. Everything the
menu can do, the menu bar's **Model** menu can do too.

## Why This Matters

The control that decides who answers a message is the most consequential
control in the client, and today it is the furthest one from the message. It
sits in the top-right corner of the toolbar, where the surveyed peers do not
put it — Claude puts it next to Send, Zed on the message editor, Cursor and
VS Code Copilot on the composer, Xcode 26 on the conversation — and where
Bob's eye is not when he is typing. A person who has never opened it does not
know the choice exists, which is the same failure as having no picker at all.

The first run is where that failure is decided. The maintainer's instinct was
a welcome sheet on which Bob must choose his model before his first message,
in the style of Apple Music's splash. The measured evidence runs the other
way: a tutorial at launch bought no better task success and made the tasks
feel harder, and the default effect means most people keep whatever is
preselected — so the thing to preselect is the one that always works. Apple
Music's sheet is the wrong precedent for a required choice precisely because
it asks nothing. What survives from the instinct is the greeting, and the
sheet in this release is exactly that: shown once, dismissible, selecting
nothing, gating no message.

That leaves the real problem the sheet was reaching for. This client exists to
show what a Gropius server is for, and a chat that quietly works on the Mac's
own model risks nobody ever reaching for the server. The answer is not to
interrogate Bob before he has a reason to care, but to make the offer at the
moment a reason appears: the first time servers show up on his network, one
dismissible line above the composer puts Alice's models one click away. A tip
in context beats a tour up front, and it is the only moment in the flow where
the server's existence is news.

The rest follows from the same principle — say what is true, at the moment it
is true. A fallback that happens silently is the one thing the record shows
people will not forgive; a fallback written into the transcript with a retry
costs a line and buys the trust. A server listed twice because Bonjour renamed
it is the client blaming the network for its own bookkeeping. And three
distinct reasons for an unavailable on-device model deserve three distinct
sentences, because two of them have a fix and one does not.

## Mechanism

We expect these six things to work, and each is wrong in a way that would
show:

1. **Opening on the on-device model removes the first-run choice people
   abandon**, because the model that ships with the Mac needs no download, no
   address and no key, so there is always a working answerer to preselect and
   the default effect works for us rather than against us. Wrong if testers on
   a fresh install still stall before their first message, or if the on-device
   model is unavailable often enough that the "always works" premise fails.
2. **The offer line is the demonstration moment**, because a contextual tip
   fired when servers first appear reaches Bob at the one point where a server
   is news to him, and a single click is a smaller price than a tour. Wrong if
   testers dismiss the line without reading it and never open the picker, or
   if the line fires so often it becomes noise.
3. **The welcome sheet greets without gating**, because a sheet that selects
   nothing and blocks nothing costs one dismissal and cannot be abandoned at:
   the measured penalty in the evidence attaches to a required choice, not to
   a hello. Wrong if the sheet measurably delays a first message, or if
   testers report expecting to have chosen something on it.
4. **One default plus one fixed fallback plus favourites covers the use a
   ranked list was for, without a list to maintain**, because the only
   ordering that ever matters in practice is "the preferred server, and this
   Mac when it cannot be reached", and stars handle the rest by frequency
   rather than by rank. Wrong if a tester with three servers keeps
   hand-switching in an order the two controls cannot express.
5. **A fallback announced only between turns cannot corrupt a streaming
   reply**, because a switch inside a stream would splice two models' tokens
   into one answer, and the decision point is therefore placed before the
   first token of a turn and nowhere else. Wrong if a reply is ever observed
   carrying text from two answerers, or if a mid-stream failure leaves a turn
   that can neither continue nor be retried.
6. **Deduplication by identity rather than by display name makes the server
   list stable**, because Bonjour itself renames on a conflict and a restart
   without deregistering leaves a stale twin, so a name is not an identity.
   Wrong if two rows still appear for one server on a restart under the
   interim rule, or if the interim rule collapses two genuinely different
   servers into one row.

## Scope Conditions

- **The chat client only, on the Mac and the iPad.** The server, the control <!-- cond: cond-2609201011309818 -->
  panel and `config.json` gain nothing; this is client-only functionality with
  no server equivalent, on the precedent of itd-2609170718438919. The Discord
  bridge is unaffected and `/model` stays its per-channel choice. The iPad
  claim holds to the terms of itd-2609180943290800: the maintainer's own
  device on a seven-day profile, the on-device path in the simulator.
- **The composer is a pill floating over the transcript, and the picker sits <!-- cond: cond-2609201011303676 -->
  at its start.** There is no toolbar picker and no hover-to-expand: the
  collapsed form is a glyph and a chevron, not a dot, because the dot belongs
  to the Circle mark of itd-2609200827202340. The window's title is the
  conversation, never the app name.
- **The welcome sheet is shown once per install, is dismissible, selects <!-- cond: cond-2609201011309189 -->
  nothing and gates nothing.** It is not a tour, has no pages, and carries no
  model control; a person who dismisses it is in a working chat on this Mac's
  model. Re-showing it is not a feature and there is no setting to bring it
  back.
- **The on-device row reports exactly the three availability reasons the <!-- cond: cond-2609201011308581 -->
  framework gives** — not eligible, not switched on, not ready yet — in the
  client's own words, with a fix offered for the two that have one. This is
  the side already settled on iss-2609181116081228; nothing here reopens it,
  and nothing here makes the client show the framework's own error text.
- **A stored API key never reaches a newly found server.** The key is bound to <!-- cond: cond-2609201011306840 -->
  the origin it was saved for, as iss-2609181116082211 settled; the offer
  line, the per-server sections and the fallback path all inherit that rule,
  and picking a server that wants a key asks for one rather than reusing
  another server's.
- **Deduplication is by identity, under an interim rule until the server <!-- cond: cond-2609201011309289 -->
  advertises one.** Where TLS is on, the certificate's spki fingerprint is the
  identity; otherwise the instance name is, and the list is as good as Bonjour
  makes it. The stable per-install server id is iss-2609200830185068's work,
  on the server side, and this intent does not wait for it or block on it.
- **The answer styles are not this intent's.** The top right of the window <!-- cond: cond-2609201011306914 -->
  belongs to itd-2609200850330402's segmented control of three marks; this
  intent touches neither the styles nor the reply labels, and the two controls
  must remain visually distinguishable at every window width.
- **The menu bar's Model menu mirrors the picker and adds nothing.** Every <!-- cond: cond-2609201011309498 -->
  action the picker offers exists as a menu command, as the HIG asks; the menu
  is not a second place where behaviour is decided, and a capability that
  exists in only one of the two is a defect.

## Acceptance Criteria

- Given a clean install, when Bob launches GropiusChat for the first time,
  then a welcome sheet appears that carries no model control, and dismissing
  it leaves an open chat whose picker reads **On this Mac**, with nothing
  blocking the first message. Held by the XCUITest tier on a reset simulator,
  and by an architecture test that the welcome view declares no picker or
  selection control.
- Given the welcome sheet has been dismissed, when Bob quits and launches
  again, then it does not appear, and no setting anywhere brings it back.
  Held by the XCUITest tier across two launches, and by an architecture test
  that the sheet is presented on a single stored flag that dismissal sets.
- Given no server has been seen before, when one or more Gropius servers
  appear on the network, then exactly one line appears above the composer
  naming one of them, offering **Use it**, **See all** and a dismissal that
  is permanent, and no model changes until Bob clicks. Held by an architecture
  test over the offer view for the three actions and the dismissal flag, and
  by a recorded hand check on the maintainer's own network with a second Mac.
- Given the chat window, when Bob looks at the start of the message pill, then
  the picker sits there carrying the model's name and where it runs, and no
  model picker exists in the toolbar. Held by an architecture test over the
  composer view, on the pattern of the existing composer test.
- Given the window is narrowed, when the label no longer fits, then the picker
  shows the server glyph or the Apple mark with a chevron and no dot, the
  model's name is in its tooltip and its accessibility label, and nothing
  changes on hover. Held by the XCUITest tier for the drawn collapse, and by
  an architecture test that the collapsed form names no circle glyph.
- Given servers on the network, when Bob opens the picker, then **On this Mac**
  is the first section, each server is its own section listing only its chat
  models, a server wanting a key shows the lock, an unreachable server is
  greyed with **No response** and stays listed, and a paired server is shown
  as paired with the pairing's own glyph. Held by an architecture test over
  `Picker.swift` for the sections, states and distinct glyphs, and by a
  recorded hand check against a server that is switched off while the picker
  is open.
- Given Bob stars two models, when he reopens the picker, then those two sit
  at the top of the menu in a favourites group, and unstarring returns a model
  to its server's section. Held by the Swift unit target over the ordering
  function, and by the XCUITest tier for the drawn group.
- Given Settings, when Bob opens the model section, then it holds exactly two
  controls — the default model for a new chat and the fallback to this Mac,
  on by default — and no ranked or reorderable list of servers exists
  anywhere in the client. Held by an architecture test over `SettingsView`,
  on the pattern of the settings-surface test.
- Given a chat on a server model and the fallback switched on, when the server
  cannot be reached at the moment Bob sends, then the turn is answered on this
  Mac, a row in the transcript says which server failed and offers a retry on
  it, and the picker's tint says the chat is not on its chosen model. Held by
  the Swift unit target over the answerer-decision function, and by the
  XCUITest tier for the transcript row.
- Given a reply is streaming, when the server fails mid-stream, then no switch
  occurs: the turn ends as a failed turn Bob can retry, and no reply ever
  contains text from two answerers. Held by the Swift unit target over the
  decision function, and by an architecture test that no fallback call site
  exists inside the streaming loop.
- Given a chat on this Mac's model, when a server would answer instead, then
  Bob is asked first and nothing moves without his answer; given a chat on a
  server, when this Mac takes over, then it announces and continues. Held by
  the Swift unit target over the decision function for both directions.
- Given the on-device model is unavailable, when Bob opens the picker, then
  its row says which of the three reasons applies in the client's own words,
  offers the fix for the two that have one, and the row is never silently
  absent. Held by an architecture test over `Backends.swift` that all three
  cases carry distinct text, and by a recorded hand check on a Mac with Apple
  Intelligence switched off.
- Given a key stored for one server, when a newly found server is picked, then
  no `Authorization` header carries that key to it, and the client asks for a
  key of its own. Held by the Swift unit target over the request builder for
  a foreign origin, and by the existing architecture test on the origin check.
- Given two advertisements for one server — a Bonjour rename, or a stale twin
  after a restart — when the picker lists servers, then one row appears,
  matched by the spki fingerprint where TLS is on and by instance name
  otherwise. Held by the Swift unit target over the deduplication function
  with both advertisement shapes.
- Given the menu bar, when Bob opens the **Model** menu, then it offers every
  model and every action the picker offers, and picking there is picking here.
  Held by an architecture test that each picker action has a matching command,
  and by a recorded hand check of the menu against an open picker.
- Given the client's documentation, when a reader opens `client/README.md`,
  then it describes the picker at the composer, the welcome sheet, the offer
  line, the two Settings controls and the announced fallback, and describes no
  toolbar picker. Held by an architecture test over the docs, on the pattern
  of the existing docs tests.

## Open Questions

All six are answered; none is open.

- **The maintainer proposed a welcome sheet in which Bob must choose a model
  before his first message.** Answered at the second planning interview of
  2026-09-20 (`.abcd/work/DECISIONS.md`): no must-choose sheet, but a
  first-start welcome sheet that is friendly, dismissible, shown once, selects
  nothing and never gates a message. The chat still opens working on this
  Mac's model and the on-network offer line remains the demonstration moment.
- **The maintainer proposed ranked preferences across servers.** Answered at
  the same interview: one default, one fixed fallback, and stars that pin
  favourites to the top of the menu; no reorderable preference list.
- **Where the picker lives.** Answered on the 2026-09-20 mock (boards 2 and
  6A over 6B): the pill composer with the picker at its start, collapsing to a
  server glyph rather than a dot; no toolbar picker; the window title is the
  conversation; the top right carries the answer styles, filed as
  itd-2609200850330402.
- **Server identity for deduplication.** Answered by the scope condition
  above: the interim rule is the spki fingerprint under TLS and the instance
  name otherwise, and the stable id is iss-2609200830185068's server-side
  work, which this intent does not wait for.
- **The on-device availability message.** Answered on iss-2609181116081228 at
  the 2026-09-20 interview: the client keeps reading availability and shows
  its own message; this intent refines that message into three states and
  reopens nothing.
- **The API key for a newly found server.** Answered on iss-2609181116082211:
  a stored key is sent only to the origin it was saved for, and the offer line
  and the per-server sections inherit that rule.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the maintainer answered the open questions at the second interview of 2026-09-20 and the itd-1 sections were written from the answers; what would show this wrong is a criterion that cannot be held by the holder it names

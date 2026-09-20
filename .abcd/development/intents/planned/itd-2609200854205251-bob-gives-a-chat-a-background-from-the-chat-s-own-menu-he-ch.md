---
id: itd-2609200854205251
slug: bob-gives-a-chat-a-background-from-the-chat-s-own-menu-he-ch
spec_id: spc-2609201011306700
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609181102147562]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob gives a chat a background

## Press Release

Bob gives a chat a background. He is deep in one long conversation and wants
it to look like that conversation rather than like every other one, so he
opens the chat's own menu, picks Background, and chooses from the set the
client carries: the plain colours, the gradients and the dynamic backgrounds
Messages offers on macOS 26. The transcript takes it as he picks it. The chat
he opens next is still on the plain window — the background belongs to the
conversation he set it on — and the one he decorated is still decorated when
he comes back to the client tomorrow.

He can use a picture of his own as well. Photo… opens the system's photo
picker, and Bob gives it the single image he has in mind rather than his whole
library. The picture is copied into the client's own container, so the chat
keeps it whether or not he later tidies the library it came from. On a picture
the bubbles change the way Messages' do: Bob's own messages keep the solid
accent tint he chose in Theme (iss-2609200818340380), and the model's replies
move onto the system's regular material, so the picture shows through the
reply while the words on it stay crisp. In Light and in Dark, the text reads;
there is no dimming sheet laid over the picture to make that true.

Carol, who wants one look everywhere, sets it once: Settings holds the
background a new chat starts with, and leaves every chat she has already
decorated alone. None puts a transcript back on the plain window, and it is
the first item in the menu rather than something to hunt for. Carol reads the
same chats on her iPad, and the iPad offers the same built-in set and the same
limited-access picker — the set is bundled with the client or drawn by it, so
neither system is the one with fewer backgrounds.

Alice, who runs the Dessau server the two of them talk to, sees nothing new
whatsoever. A background is a file the client reads and draws: no request body
carries it, no header names it, no log line or statistics record mentions it,
and no model is ever told what a conversation looks like. If a picture stops
resolving, that chat falls back to None without a word.

## Why This Matters

The client's one standing promise is that it is the system's own app — the
system's controls, the system's colours, the system's text. Messages on
macOS 26 gives every conversation a background, so a chat client on the same
Mac that cannot reads as an app that stopped short, and the gap is visible in
the exact place people compare the two: a window full of bubbles. Matching a
native precedent is cheaper than inventing a look, and it is the only way this
client can be said to be finished.

It is also the personalisation that costs nothing anywhere else. A background
is decoration the client draws for itself; it needs no server field, no model
prompt, no account and no sync. That makes it the cheapest possible
demonstration of the claim the whole product rests on — that what happens on
this Mac stays on it — because here the temptation to send something is
entirely absent, and the tests that hold it are the same source-reading tests
that already hold the built-in answerer's "nothing sent anywhere". A feature
that proves a promise while adding no surface to the server is worth more than
its size.

And it does real work in the transcript. Bob keeps several conversations going
at once; a chat he can recognise by its colour before he reads a word of it is
a chat he stops opening by mistake. Per conversation is what makes that true —
a single global background decorates the app, not the chats, and tells him
nothing about which one he is in.

## Mechanism

We expect a background to be provably local because it is drawn, never sent: the
transcript's drawing and the request builder are separate types in the client's
source, and the background store is named by the first and by nothing else. That
is a claim about source, which is how this project already holds "nothing sent
anywhere" for the built-in answerer — an architecture test reads the Swift and
fails when the request-building type so much as mentions the store. What would
show this wrong: a background that has to be resolved inside the request path
(for instance an export or a share that renders the transcript), which would put
the two types back in contact.

We expect the system's regular material behind the model bubble to keep the
reply legible on any picture, with no scrim over the background. The material is
the system's own blur and vibrancy, computed from whatever is behind it at the
moment it is drawn, and the label colour is specified against it; a scrim is one
fixed opacity that cannot be right for both a snow photograph and a night one.
The user bubble needs no such help because it stays a solid accent fill, opaque
by definition, and the arithmetic that gives a picked colour a face per
appearance (the decision of 2026-09-19) is what keeps its label apart from it in
both appearances. What would show this wrong: a picture whose bright region
behind a model bubble still fails the contrast the arithmetic test asserts, in
either appearance.

We expect a missing file to be harmless because the conversation record stores a
reference and not an image: resolution is attempted at draw time, a failure
yields None, and None is a state the transcript already draws every day. There
is no path on which a chat refuses to open because of its decoration, and no
migration is owed when the reference shape changes, pre-1.0. What would show
this wrong: a decode of the conversation store that fails outright on an
unresolvable reference rather than falling through to None.

We expect bundling or drawing the built-in set to be what makes the iPad equal,
because a set read from the Mac's Desktop Pictures is a set the iPad does not
have and a sandboxed client may not be entitled to read. Assets in the bundle
and gradients drawn in code compile for both builds of the one source, so the
two systems offer the same menu by construction rather than by a second
implementation kept in step. What would show this wrong: a dynamic background
that cannot be drawn without a private system asset, which would have to be
dropped from the set on both systems rather than on one.

## Scope Conditions

- The Mac client on macOS 27 and the iPad client on iPadOS 27, from the one <!-- cond: cond-2609201011300784 -->
  Swift source built twice; both offer the same menu.
- The built-in set is bundled with the client or drawn by it — plain colours, <!-- cond: cond-2609201011301249 -->
  gradients and dynamic backgrounds — and is never read from the Mac's Desktop
  Pictures folder or any other system path.
- A picture comes from the person's photo library through the system's <!-- cond: cond-2609201011303278 -->
  limited-access picker; there is no full-library access and no browsing of
  arbitrary folders.
- The choice is per conversation and is remembered with the conversation; <!-- cond: cond-2609201011303817 -->
  Settings holds the background a new chat starts with and changes no chat that
  already exists.
- Nothing about a background reaches a server or a model: not a request body, a <!-- cond: cond-2609201011309491 -->
  header, a query, a log line or a statistics record.
- The user bubble keeps the solid accent tint of iss-2609200818340380; the model <!-- cond: cond-2609201011302819 -->
  bubble moves to the system's regular material only when the background is a
  picture, and keeps its tint on a colour or a gradient.
- Pre-1.0 and without migration: the conversation record gains an optional <!-- cond: cond-2609201011305747 -->
  background reference, a reference that no longer resolves falls back to None,
  and no compatibility shim is written for stores saved before the field
  existed.
- The server side is untouched; this is a change to the chat client alone. <!-- cond: cond-2609201011307256 -->

## Acceptance Criteria

- Given a chat that is open, when Bob opens the chat's own menu, then it offers
  Background with None first and the bundled set beneath it, and Photo… for a
  picture from the library; held by an architecture test over the client's Swift
  source, which reads the menu's items as the existing client architecture tests
  read the Chat menu's.
- Given two chats, when Bob sets a background on one of them, then that chat
  draws it and the other is unchanged, and both are as he left them after a
  relaunch; held by the Swift unit target over the conversation store, plus the
  XCUITest tier for the drawn result.
- Given a default chosen in Settings, when Bob starts a new chat, then the new
  chat opens on that background and no existing chat has changed; held by the
  Swift unit target over the store and the default's resolution.
- Given a decorated chat, when Bob chooses None, then the transcript is back on
  the plain window with no residue of the previous background; held by the
  XCUITest tier.
- Given a picture chosen through the limited-access picker, when the choice is
  made, then the file is copied into the client's own container and the
  conversation record holds a reference to that copy and not a path into the
  library; held by the Swift unit target over the copy-and-reference step, and
  by an architecture test asserting the store holds a reference rather than
  image data.
- Given a conversation whose referenced picture has been removed from the
  container, when Bob opens that chat, then it draws as None, no error is shown,
  and the record is not refused; held by the Swift unit target over the
  resolver.
- Given a picture background, when the transcript draws, then the user bubble is
  the solid accent tint and the model bubble is the system's regular material,
  and the label on each keeps its contrast in Light and in Dark; the colour
  arithmetic is held by the Swift unit target (the existing appearance-aware
  arithmetic test extended to the material case), and the drawn result by the
  XCUITest tier.
- Given the whole of the client's Swift source, when the architecture tests run,
  then no request-building, logging or statistics type names the background
  store, and the background types import no networking; held by a source-reading
  architecture test beside the one that holds the built-in answerer's "nothing
  sent anywhere".
- Given the iPad build, when Carol opens the chat's menu, then it offers the
  same bundled set as the Mac's and Photo… opens the limited-access picker;
  held by an architecture test that the set is declared in one shared file with
  no macOS-only branch, and by a named hand check on the maintainer's own iPad,
  recorded with its date.
- Given the shipped change, when the client's documentation is read, then
  `client/README.md` describes choosing a background, the Settings default, the
  fallback to None, and that nothing about a background leaves the device; held
  by a docs test in the settings-docs family.

## Open Questions

- **Which "system backgrounds".** Answered: the Messages-on-macOS-26 set —
  built-in colours, gradients and dynamic backgrounds bundled or drawn by the
  client, plus a picture from the photo library; Desktop wallpapers declined.
  Maintainer's planning interview of 2026-09-20, recorded in
  `.abcd/work/DECISIONS.md` and carried by the Scope Conditions above.
- **Per conversation or global.** Answered: per conversation, with a Settings
  default for new chats; global-only declined. Same interview, same ledger line;
  carried by the Scope Conditions and the second and third criteria.
- **Legibility on a picture.** Answered: the model bubble sits on the system's
  regular material and the user bubble keeps its solid accent tint, with a test
  on the colour arithmetic holding contrast in Light and Dark; a scrim, and
  material behind both bubbles, declined. Same interview; carried by the second
  Mechanism claim and the seventh criterion.
- **The iPad.** Answered: the same bundled set and the limited-access picker,
  from the one source built twice. Same interview; carried by the fourth
  Mechanism claim and the ninth criterion.
- **Storage.** Answered as drafted: the picture is copied into the client's own
  container and referenced from the conversation record, and a reference that no
  longer resolves falls back to None, pre-1.0 and without migration. Same
  interview; carried by the third Mechanism claim and the fifth and sixth
  criteria.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the maintainer answered the open questions at the second interview of 2026-09-20 and the itd-1 sections were written from the answers; what would show this wrong is a criterion that cannot be held by the holder it names

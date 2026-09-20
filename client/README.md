# Dessau Chat

A native chat app for the Mac and the iPad. It chats with the device's own
model out of the box — the language model Apple ships with the system, on the
device, with nothing sent anywhere — and it talks to any OpenAI-compatible
server: give it a base URL and that server's models appear in the same picker,
with streaming replies, no browser and no configuration.

[Dessau Server](../README.md) is the server it finds by itself, over Bonjour on
the local network, and the two are released together. It is not a component of
the server and the server is not a requirement of it: anything answering
`GET /v1/models` and `POST /v1/chat/completions` is a server this client can
use, and the Mac's own model needs no server at all.

**Requires macOS 27** and Apple Silicon, which is every Mac that runs macOS 27.
Dessau Server has the same floor, so one requirement covers both; a Mac below
it is refused by `install.sh` before anything is downloaded.

## Build

Needs the installed Xcode (27 or later) on the Mac you build on: the macOS 27
SDK's SwiftUI is implemented with compiler macros whose plugin ships only
inside Xcode, so the Command Line Tools alone cannot build this app. No Xcode
project, though — a handful of Swift files, one script:

```sh
./build.sh
open dist/DessauChat.app
```

`build.sh` compiles with `xcrun swiftc`, writes the App Intents metadata with
the toolchain's own processor (and fails if it did not), and assembles an
Apple Silicon `dist/DessauChat.app`.

## Use

1. Launch it and type. The picker in the toolbar reads **On this Mac**: the
   Mac's own model answers, on the device. It needs Apple Intelligence switched
   on; if the Mac cannot answer — Apple Intelligence off, the model still
   downloading, an unsupported language — the empty chat says which, and what
   would fix it, and offers a server instead.

   The device's own model holds a small context window. A long conversation
   keeps going: the client sends the instructions and the most recent turns
   that fit alongside the message you just typed. A single message too long for
   the window is a different matter — it is not sent at all, and the chat says
   so and points at a server.
2. To use a server on the network, click the picker. Under **Servers on your
   network** it lists every Dessau Server it can find while the picker is
   open: each row names the Mac the server runs on — "Alice's Mac", the name
   that Mac carries in System Settings, with no product name in front of it —
   and says whether it needs an API key and how many models it can serve.
   Click one and its chat models appear under its name; click a model and the
   conversation carries on there. Nothing switches by itself, and **On this
   Mac** is always one click away.

   macOS asks for permission to search the local network the first time the
   picker opens — the client browses for servers only while the picker is
   open. When a server answered last, the client does reconnect to its stored
   address at launch, so a `.local` address can bring that question then.
   Without it the list stays empty; a server elsewhere — another Dessau Server,
   or anything else that speaks the OpenAI API — can be typed into **Settings**
   (Cmd-,) as a base URL: the Mac's `.local` name or its LAN address with port
   `11535` and no path, for example `http://your-mac.local:11535`. A server
   that needs an API key asks for it when you pick it, once: the key is kept in
   your Keychain, sent only to the server it was entered for, and changeable
   later in Settings.
3. **Return** sends; **Option-Return** starts a new line; the arrow button sends
   too, and the stop button interrupts a reply. Drop a text file on the message
   box and its content becomes part of the prompt. Select text in the box or in
   a reply and the system's **Writing Tools** — proofread, rewrite, summarise —
   are in the context menu when Apple Intelligence is on.

   The picker offers a server's models that can hold a conversation. It works
   that out from the HuggingFace pipeline tag and tags the server publishes for
   each model; the rule is yours to change under **Models to offer** in
   Settings. The Mac's own model is not a served model and is always offered.

Settings also offers the appearance (Light, Dark or System), the text size
— five steps, the system's own, and the whole window follows; Default sets
nothing, so the Mac's own text size is what you get — and the colour of each
speaker's bubble. The bubbles start in colours the system supplies — your
accent colour for your messages, the system's secondary fill for the model's —
which follow Light and Dark by themselves. A colour you pick is kept as you
picked it and drawn so that it stays apart from the window in either
appearance, with the message on it in whichever of the system's label colours
reads on that bubble.

Replies render their markdown — emphasis, code, lists, headings, block quotes,
code blocks, links — and a reply's context menu offers **Copy**, which copies
what the model wrote, marks and all. Two constructs are not drawn: a table is
shown as the model wrote it, marks and all, and a nested list is drawn as one
level of items. A few words (congratulations, well done,
warning, careful, wow, amazing) animate once when a reply arrives; switch that
off under **Replies** in Settings.

Loading a model on a server takes seconds to a minute. While that is happening
the reply reads **Loading**, with the model's name, rather than showing the same
spinner a slow answer shows. Thinking models stream their reasoning; a
**Thoughts** row above the answer expands to show it.

### Chats and windows

The sidebar shows each conversation as a card — who answered, the title,
the date and how many exchanges and words it holds — with a search bar above
it that filters by title and message text. Type several words and the list
keeps the chats that hold every one of them, in any order and anywhere in the
conversation; case and accents do not matter. **New Chat** (Cmd-N) starts one;
the **Chat** menu carries every action with its shortcut. Each window remembers
which chat it shows and comes back after a relaunch, the way macOS restores
windows. Delete a chat by swiping or right-clicking it. Everything is saved
to `~/Library/Application Support/DessauChat/conversations.json` and restored
on next launch — history lives on the machine running the client, not on the
server.

### Shortcuts and Spotlight

The client declares its actions as App Intents, so Shortcuts lists **Ask
Dessau Chat** (a prompt in, the reply out as text, kept as a new chat), **New Chat**
and **Open Chat** (by title) under the app, and Spotlight offers them. Chats are
looked up when you pick one; nothing is added to the system's index.

## On the iPad

The same Swift files build an iPad app: one source, two systems. It is the
client you know — the sidebar of chats, the composer, the model picker in the
toolbar, the menu bar when a keyboard is attached — with Settings behind the
gear in the toolbar rather than a Settings window, since iPadOS has no such
window. **Requires iPadOS 27**, and it is an iPad app: there is no iPhone
version.

On an iPad that can run Apple Intelligence — an M1 or later, or the iPad mini
with the A17 Pro — the picker reads **On this iPad** and the iPad answers on
the device, with nothing sent anywhere. On any other iPad the empty chat says the iPad cannot answer and
offers a server instead: pick a Dessau Server on your network and the
conversation carries on there, exactly as it does on a Mac.

The app is built for **your own iPad**, signed with your own Apple ID's free
personal team, and installed over USB from the Mac you build on. Nothing is
published: there is no release asset, and `install.sh` knows nothing of it.
A personal team's profile expires after seven days, so a build is good for a
week and re-running the script is what renews it.

### The one sign-in

Open Xcode > Settings > Accounts, add your Apple ID and let the personal team
issue an **Apple Development** certificate. That sign-in also writes the
matching provisioning profile for `sh.intentdriven.dessau.chat` into
`~/Library/Developer/Xcode/UserData/Provisioning Profiles`. Nothing else in
this build needs Xcode's project files.

### Building it

```sh
SIM=1 ./build-ipad.sh       # the simulator: build, install, launch
```

The simulator run is the check you can make without an iPad: it builds,
installs the bundle on an available iPad simulator, launches it and reports
whether it stayed running. The simulator borrows the Mac's own language model,
so the on-device answer can be tried there even when no iPad to hand is
eligible for one.

For your iPad, name the certificate and the profile, and the device to install
on:

```sh
IPAD_SIGNING_IDENTITY="Apple Development: <your name> (XXXXXXXXXX)" \
IPAD_PROFILE="/path/to/profile.mobileprovision" \
IPAD_DEVICE="<the iPad's identifier>" \
./build-ipad.sh
```

`IPAD_DEVICE` is optional — without it the bundle is built and signed and the
script prints the `devicectl` command that installs it. `IPAD_SIM` picks a
particular simulator; `VERSION` sets the version string. Without an identity
and a profile the script refuses before it builds anything, because an iPad
installs neither an unsigned bundle nor an ad-hoc signed one.

## Distributing it to another Mac

The app is **ad-hoc signed**, not signed with an Apple Developer ID. That's fine
for your own network but macOS Gatekeeper will quarantine it after it's copied or
downloaded. On the receiving Mac, either:

- **Right-click the app → Open** the first time, then confirm the dialog, or
- clear the quarantine flag from a terminal:

  ```sh
  xattr -dr com.apple.quarantine /path/to/DessauChat.app
  ```

For friction-free distribution to Macs you don't control, you'd sign and
**notarise** the app with an Apple Developer ID — out of scope here.

The app talks plain HTTP to a LAN address; its `Info.plist` allows that
(`NSAllowsLocalNetworking`) and declares Local Network access, which macOS asks
you to approve the first time the picker looks for a server.

## What it is under the hood

- `DessauChat/DessauChat.swift` — the app, the model and the views (SwiftUI).
- `DessauChat/Backends.swift` — the two answerers: the Mac's own model through
  the Foundation Models framework, and a server over `GET /v1/models` and
  `POST /v1/chat/completions` with `stream: true`, parsed as SSE.
- `DessauChat/Discovery.swift` — what both clients know about the local
  network: the service type, the browse, and resolving the server a person
  picked to an address, all on the Network framework.
- `DessauChat/Picker.swift` — the model picker and the Bonjour browse that runs
  only while it is open.
- `DessauChat/Markdown.swift`, `Effects.swift`, `Intents.swift` — the reply
  rendering, the word effects, the App Intents.
- `Info.plist` — bundle metadata, local-network entitlements, and the Bonjour
  service type the app may browse for (`_dessau._tcp`, the one the server
  advertises).
- `Info-iPad.plist` — the same metadata for the iPad bundle, plus the launch
  screen and the one device family it is built for.
- `build.sh`, `build-ipad.sh` — compile with `xcrun swiftc`, write the App
  Intents metadata and assemble the `.app` for each system.

Settings persist across launches: the server address and chosen model in
`UserDefaults`, the API key in the macOS Keychain.

---
id: itd-2609200823510967
slug: gropius-runs-on-macos-27-on-apple-silicon-and-nowhere-else-s
spec_id: spc-2609200831079320
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Gropius runs on macOS 27 on Apple Silicon and nowhere else, server and client alike: Alice installs it on a Mac that runs macOS 27, the installer and the launcher refuse an older macOS or an Intel Mac in one sentence naming the floor, the release carries one slice per binary and no kept older client, and every page that names a version names 27. Nothing is kept for macOS 26 or for Intel.

## Press Release

Gropius runs on macOS 27 on Apple Silicon, and nowhere else. Alice installs
the server on a Mac running macOS 27 with the one-line command, and Bob
installs the chat client with the same command on the same kind of Mac. A Mac
on an older macOS is turned away by the installer in one sentence that names
the floor, before anything is downloaded; an Intel Mac is turned away in one
sentence that names Apple Silicon. Nothing is half-installed and nothing is
downloaded for a Mac that cannot run it. Every page that states a
requirement states the same one: macOS 27, Apple Silicon. Carol, reading the
project's page before she installs anything, sees one requirement rather than
two floors and a footnote about which build her Mac would get.

There is one release, and it carries one bundle per app, one slice each. The
installer fetches from that release only: one set of checksums, one archive
per app, one placer. Nothing points at an older release any more.

## Why This Matters

Two floors cost more than they bought. The server said macOS 26 and the
client said 27, so the installer carried two gates, two release paths and two
checksum fetches, and the pages carried a paragraph explaining which Mac gets
which build. That paragraph is the product's first impression, and it
described a compatibility matrix rather than a product. The older client it
pointed at is a frozen build that receives no fix — a promise of support
that is not support.

The floor is where the product is developed and tested: macOS 27 on Apple
Silicon, which is the only configuration any of this is verified on. Saying
so removes a gate, a release path, a checksum fetch, a kept tag and a
paragraph, and leaves one sentence a reader can act on.

## Mechanism

We expect one floor to be strictly safer to install than two, because every
refusal then rests on a single number that one architecture test holds
against both bundles' `LSMinimumSystemVersion`: the installer cannot admit a
Mac that Launch Services will refuse, and it cannot fetch an asset from a
release whose checksums it did not fetch, because there is only one release
path left to fetch from. We expect nothing to be lost by dropping the second
release path, because the integrity control it needed — a second
`SHA256SUMS.txt` for a second release — is the one place in the installer
where two different origins were verified in one run, and a control written
once cannot differ from itself.

What would show this wrong: a tester who was meant to be served on macOS 26
or on an Intel Mac; or a CI runner image below the new floor, which would
turn the installer gate into a silent no-op instead of a red build.

## Scope Conditions

- macOS 27 on Apple Silicon is the product floor for both apps. The server <!-- cond: cond-2609200831070235 -->
  declares it in `build/Info.plist` and the client in `client/Info.plist`,
  and every other surface that names a minimum is held to those by
  `internal/archtest`.
- The installer refuses below the floor before it downloads anything, and <!-- cond: cond-2609200831079615 -->
  refuses an Intel Mac in a sentence naming Apple Silicon and the floor.
  There is no fallback download for either refusal.
- The published v0.6.0 release is not deleted and its tag stays; nothing in <!-- cond: cond-2609200831070857 -->
  the tree points at it. Whether it is ever unpublished is the maintainer's
  deliberate act.
- Pre-1.0: this is `impact: breaking` and carries no migration path. A Mac <!-- cond: cond-2609200831073867 -->
  that was installed under the old floor keeps what it has; the installer
  simply will not serve it again.
- CI's macOS runner images are at or above the floor, because the installer <!-- cond: cond-2609200831073158 -->
  gate is the only thing that executes `install.sh` and a runner below the
  floor would skip it silently. The `xcode-27` image is macOS 27.0, verified
  in the release run of 2026-09-19.
- The iPad client's floor is its own (`client/Info-iPad.plist`) and is not <!-- cond: cond-2609200831079725 -->
  touched here.
- Nothing about the Python runtime, MLX, or the models changes: the floor is <!-- cond: cond-2609200831073559 -->
  a platform statement, not a behaviour change to the server.

## Acceptance Criteria

- Given a Mac running macOS 26 or older, when Alice runs the one-line
  installer for either app, then it refuses in one sentence naming macOS 27
  and nothing is downloaded or written. Held by the installer subtest of
  `TestEverySurfaceDeclaresTheSameMacOSFloor`, which holds the script's one
  `MIN_MACOS_MAJOR` to both bundles' declared major; the refusal itself
  cannot be executed here, because every command in `install.sh` is named by
  absolute path and `sw_vers` therefore cannot be stubbed, so it is a named
  hand check on a Mac below the floor.
- Given an Intel Mac, when Bob runs the installer for either app, then it
  refuses in one sentence naming Apple Silicon and the floor, and names no
  fallback build. Held by `TestTheInstallerKeepsNoSecondReleasePath` (no
  branch assigns a second release path) and, for the wording, by a named hand
  check on an Intel Mac.
- Given both app bundles, when their `LSMinimumSystemVersion` values are
  compared, then both are 27.0 and every declared minimum in the tree agrees
  with them. Held by `TestEverySurfaceDeclaresTheSameMacOSFloor`.
- Given `install.sh`, when it is read for a second release path, then it has
  none: no kept client tag, no client-specific floor, one release path and
  exactly one `SHA256SUMS.txt` fetch per run. Held by
  `TestTheInstallerKeepsNoSecondReleasePath`.
- Given the workflows, when their macOS runner labels are compared with the
  floor, then none names an image below it. Held by
  `TestEveryMacOSRunnerIsAtOrAboveTheFloor`.
- Given the README, the getting-started guide and the client README, when a
  reader looks for the requirement, then each states "Requires macOS 27" and
  none names macOS 26, an Intel Mac, a universal build or release v0.6.0.
  Held by the prose subtest of `TestEverySurfaceDeclaresTheSameMacOSFloor`
  and by `abcd docs lint`.
- Given the landing page, when its requirement is compared with the server
  bundle's declared minimum, then both say macOS 27. Held by
  `TestPageAndBundleAgreeOnTheMinimumMacOS` in `internal/sitetest`.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: one floor at macOS 27 on Apple Silicon is expected to be safer to install and cheaper to state than two, because every refusal then rests on a single number held to both bundles' LSMinimumSystemVersion and the installer keeps one release path and one checksums fetch; what would show it wrong is a tester who was meant to be served on macOS 26 or an Intel Mac, or a CI runner image below the floor turning the installer gate into a silent skip

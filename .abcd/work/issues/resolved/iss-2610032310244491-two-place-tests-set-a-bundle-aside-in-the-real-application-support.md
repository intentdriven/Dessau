---
schema_version: 1
id: "iss-2610032310244491"
slug: "two-place-tests-set-a-bundle-aside-in-the-real-application-support"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "review of the HOME guard (iss-2609200823589977)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/lifecycle/place_test.go"
remedy: "Give both tests their own HOME with swapHome(t, dir), as the swap tests do."
resolution: "Both place tests call swapHome(t, dir); with the guard made to panic they failed before the change and pass after."
impact: fix
resolved_by:
  commit: "5b3073c"
---

TestPlaceReplacesAnInstalledBundleRatherThanNestingInsideIt and TestPlaceReplacesASymlinkAtTheDestinationRatherThanFollowingIt never point HOME anywhere, and RunPlace sets the replaced bundle aside through setAsideDir, accountSwapHome and config.AccountHome, so on a Mac they created and wrote retired-bundle staging directories under the developer's real ~/Library/Application Support/Dessau: the hazard iss-2609200823589977 names. With the guard, AccountHome refuses and setAsideDir swallows the refusal into its staged-in-the-destination fallback, so the tests pass, write nothing real, and no longer exercise the own-home path they were written for.

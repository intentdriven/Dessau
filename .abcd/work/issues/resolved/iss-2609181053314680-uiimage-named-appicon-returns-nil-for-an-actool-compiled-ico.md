---
schema_version: 1
id: "iss-2609181053314680"
slug: "uiimage-named-appicon-returns-nil-for-an-actool-compiled-ico"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "Measured in the simulator: UIImage(named: \"AppIcon\") is nil for an actool-compiled icon set, so the empty state reads the bundle's CFBundleIcons file names (both load) and keeps the symbol only for a bundle whose icon step did not run."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

UIImage(named: "AppIcon") returns nil for an actool-compiled icon set, so the iPad's empty state always shows the symbol fallback; either load the icon from the bundle's CFBundleIcons entry or drop the branch.

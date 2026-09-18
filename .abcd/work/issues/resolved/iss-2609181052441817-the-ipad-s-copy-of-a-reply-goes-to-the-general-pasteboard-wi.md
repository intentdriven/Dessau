---
schema_version: 1
id: "iss-2609181052441817"
slug: "the-ipad-s-copy-of-a-reply-goes-to-the-general-pasteboard-wi"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "security review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The iPad's Copy writes the reply with UIPasteboard's localOnly option, so a copied reply does not travel over Universal Clipboard."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

The iPad's copy of a reply goes to the general pasteboard without the localOnly option, so a copied reply can leave the device through Universal Clipboard; the client README's 'nothing sent anywhere' does not cover the pasteboard. Set localOnly on the iPad.

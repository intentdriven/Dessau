---
schema_version: 1
id: "iss-2609181052435807"
slug: "the-keychain-item-that-holds-the-api-key-is-ksecattraccessib"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "security review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The API key's Keychain item is kSecAttrAccessibleWhenUnlockedThisDeviceOnly, so it is not carried in an encrypted backup onto another device."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

The Keychain item that holds the API key is kSecAttrAccessibleWhenUnlocked, which on iPadOS is carried in encrypted backups and restorable onto another device; it should be the ThisDeviceOnly variant on both platforms.

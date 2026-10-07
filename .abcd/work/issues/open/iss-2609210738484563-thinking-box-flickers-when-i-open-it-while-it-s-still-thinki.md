---
schema_version: 1
id: "iss-2609210738484563"
slug: "thinking-box-flickers-when-i-open-it-while-it-s-still-thinki"
severity: "minor"
category: "bug"
source: "manual-test"
found_during: "maintainer's use of Dessau Chat against Dessau Server v0.9.1, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "client/DessauChat/DessauChat.swift"
remedy: "Keep the reasoning box stable while it streams: render its text without re-laying out the whole disclosure on every chunk (throttle updates or append plain text, parsing markdown once the reasoning ends), then check by hand that an open box no longer flickers while thinking."
---

Thinking box flickers when I open it while it's still thinking. It shows stable once it is done. (The maintainer's words. The box is the reasoning DisclosureGroup under a reply; opened while the reasoning is still streaming in, it flickers with each update and settles only once the answer has finished.)

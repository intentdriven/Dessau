---
schema_version: 1
id: "iss-2609190200066074"
slug: "a-relative-rel-next-page-url-in-the-hub-s-link-header-is-ref"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "the adversarial security review of the same-origin rule added for iss-2609190023171129"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
---

A relative rel="next" page URL in the Hub's Link header is refused by Files and told to the operator as cross-origin. sameOrigin in internal/hub/hub.go counts a URL with no host as a different origin, which is the right default for an unparseable value, but RFC 8288 permits a relative URI-reference and the Hub's own paging URLs are absolute only by current practice. If that changes, file listing stops at page one with an error naming a same-origin path as cross-origin, which sends a debugger the wrong way. Resolve a relative next page against the URL of the page it came from before the check, or say 'unparseable' rather than 'cross-origin' for a value with no host.

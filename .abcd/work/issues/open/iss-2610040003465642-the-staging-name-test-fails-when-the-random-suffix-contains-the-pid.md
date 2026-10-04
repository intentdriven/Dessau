---
schema_version: 1
id: "iss-2610040003465642"
slug: "the-staging-name-test-fails-when-the-random-suffix-contains-the-pid"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "CI on PR 180, 2026-10-04 (linux job, a records-only change)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/lifecycle/swap_test.go"
remedy: "Refuse a suffix that is the pid, not one that contains it: compare the part after the prefix with the pid exactly."
---

TestTheStagingNameIsUnguessable fails at random: it refuses any staging name whose text contains the process id as a substring, but os.MkdirTemp's suffix is a random decimal of up to ten digits, so a four- or five-digit pid appears inside it by chance (on PR 180's linux run the name was .dessau-incoming-1415576321). The hazard the test exists for is a name derived from the pid, not a random number that happens to share its digits.

---
schema_version: 1
id: "iss-2609190301298100"
slug: "the-closed-spec-spc-2609181018499470-cites-a-method-the-brid"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "fixing the bridge's conversation lifetime, which the brief also called Bridge.Stop"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/closed/spc-2609181018499470-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md"
---

The closed spec spc-2609181018499470 cites a method the bridge does not have. Its acceptance-criteria map, criterion 6, reads 'Switch off closes the connection — Bridge.Stop', but internal/bridge/discord/bridge.go has no Stop: the switch off is Apply(on=false, token="") and the shutdown path is Close(). A reader following the citation to the code finds nothing, and a later change that wanted to keep the switch's promise would have no named seam to hold. The criterion itself is held by TestSwitchingTheBridgeOffClosesTheConnection. A closed spec is not edited, so this is either a note on the record or a line in the decisions ledger naming the two methods that carry the criterion.

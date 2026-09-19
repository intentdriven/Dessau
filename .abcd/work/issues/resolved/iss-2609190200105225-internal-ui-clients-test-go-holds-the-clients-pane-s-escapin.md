---
schema_version: 1
id: "iss-2609190200105225"
slug: "internal-ui-clients-test-go-holds-the-clients-pane-s-escapin"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/clients_test.go"
resolution: "TestAClientNameThatIsMarkupIsDrawnAsText renders the Clients pane with a name of '<img src=x onerror=1>' through a recording document that serialises what was set as text apart from what was set as markup, and asserts the name is escaped in the markup and present in the text. The source scan stays beside it as the rule that produces that outcome."
impact: internal
---

internal/ui/clients_test.go holds the Clients pane's escaping by scanning the render function's source for textContent and for the absence of innerHTML. The spec promised a test that plants a client name containing markup and asserts it is drawn as text. The source scan passes over any future path that builds a row some other way, and nothing exercises the rendering with a hostile name — which matters because the name is chosen by whoever pairs and the pane is drawn on the panel that has the whole settings API behind it.

## Grounds

- pursued: we expect an outcome test to survive a row built some other way, because a source scan passes over any future path it does not name; wrong if the recording document diverges from a browser's own textContent semantics.

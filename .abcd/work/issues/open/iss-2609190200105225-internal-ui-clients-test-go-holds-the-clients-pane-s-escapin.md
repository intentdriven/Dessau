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
---

internal/ui/clients_test.go holds the Clients pane's escaping by scanning the render function's source for textContent and for the absence of innerHTML. The spec promised a test that plants a client name containing markup and asserts it is drawn as text. The source scan passes over any future path that builds a row some other way, and nothing exercises the rendering with a hostile name — which matters because the name is chosen by whoever pairs and the pane is drawn on the panel that has the whole settings API behind it.

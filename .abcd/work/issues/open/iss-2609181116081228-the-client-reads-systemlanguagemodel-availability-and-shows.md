---
schema_version: 1
id: "iss-2609181116081228"
slug: "the-client-reads-systemlanguagemodel-availability-and-shows"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Backends.swift"
---

The client reads SystemLanguageModel.availability and shows its own Apple-Intelligence-off message for the built-in backend, against the text-intelligence intent's promise that it neither checks the state nor shows an error of its own.

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

## Triage 2026-09-18

Left open for the maintainer: either the client stops reading availability or the text-intelligence intent stops promising it does not — a choice between two shipped promises, and the delivered message is the more useful of the two.

## Deferral 2026-09-19

Waits on the maintainer: a choice between two shipped promises (the client stops reading availability, or the text-intelligence intent stops promising it does not).

## Decision 2026-09-20

Maintainer's decision at interview: keep the delivered behaviour; the shipped promise is corrected by a decision line in `.abcd/work/DECISIONS.md`. Resolved on that grounds.

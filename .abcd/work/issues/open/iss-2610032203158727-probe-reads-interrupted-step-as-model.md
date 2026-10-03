---
schema_version: 1
id: "iss-2610032203158727"
slug: "probe-reads-interrupted-step-as-model"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "implementing the maintainer's decision on iss-2610031818057157"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/contextprobe/probe.go"
remedy: "A step whose session was cancelled while its request was out carries no reading: check the session after every request, whatever the answer, and classify the step as cancelled."
---

When the panel's Unload interrupts a context-probe step, the step's own request can come back as the gateway's upstream failure (the server it was asking was just stopped) before the probe sees its session cancelled. step() then reads that answer as the model's: at calibration the probe gives up as 'did not answer at the floor', and mid-bisection it records the interrupted size as the model's own limit in the bounds it keeps for the resume, so the measurement can come out smaller than the model's real window. Which way a given Unload goes is a race, which is also why TestUnloadFromThePanelTakesTheModelBackFromTheProbe saw the model loaded back on some runs.

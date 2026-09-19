# Loud staging

## The rule

> Loud staging: a stage that no-ops or degrades must say so; never manufacture
> a false green — see .abcd/development/principles/loud-staging.md.

## Why

The repository's clearest case is `iss-2609161712555136`. A queued "Measure
now" whose model fails the fit check is dropped silently on every tick:
`selftest.Runner.tick` asks `Server.Fits`, and on false moves on with no
`held_by`, no log line, and the model left in `probe_queue` for ever. The
manual check that found it saw `job null, held_by null` for three hours and
forty minutes while the panel showed the model queued — a stage that had
stopped doing anything while every surface read as fine. The fix is the rule
stated positively: the tick reports the job as `held_by` `no_room` with `due`
naming the model, says once per model in the log that the measurement is held
because the model does not fit the memory budget, and the panel and
`docs/context-probe.md` word the reason.

`iss-2609180608280915` is the same failure in the client. `AppModel.ask`
returns an empty reply as success when nothing can answer: `send` returns
silently on `cannotSend`, `ask` awaits the previous reply task and returns
empty, and the person is left looking at an empty chat. The record's own
statement of the fix is that `ask` should throw the client's own reason rather
than hand back a success that is not one.

Records drawn on: `iss-2609161712555136`, `iss-2609180608280915`.

## How to apply here

- A loop that skips work reports the skip on the surface that shows the work:
  a `held_by` reason on the control plane, one log line per cause rather than
  one per tick, and the reason worded in `docs/`.
- A function that cannot do what it was asked returns an error naming the
  reason. An empty success is a false green.
- A test that cannot run says so; the repository does not accept a gate that
  passes because it did nothing, which is what the release gate's step-level
  placement exists to prevent.

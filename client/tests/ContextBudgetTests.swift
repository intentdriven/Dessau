// What the device's own model's context arithmetic promises, checked on its
// own: `client/tests/context-budget.sh` compiles this with
// `client/DessauChat/ContextBudget.swift` and runs it. The client has no test
// target, so this is a main that prints each promise and exits non-zero on the
// first one that is broken.

import Foundation

/// A tally of promises, so nothing in here needs a mutable global.
private struct Checks {
    private(set) var passed = 0
    private(set) var failures: [String] = []

    mutating func check(_ condition: Bool, _ what: String) {
        if condition {
            passed += 1
            print("ok   - \(what)")
        } else {
            print("FAIL - \(what)")
            failures.append(what)
        }
    }
}

/// How many of the prior turns a budget takes, walked newest first, the way
/// the built-in backend walks them.
private func kept(_ budget: ContextBudget, costsNewestFirst: [Int]) -> Int {
    var b = budget
    var n = 0
    for cost in costsNewestFirst {
        if !b.take(cost) { break }
        n += 1
    }
    return n
}

@main
enum ContextBudgetTests {
    static func main() {
        var t = Checks()

        // The new prompt is budgeted with the prior turns, not after them: a
        // window of 1000 with a 250 reserve, 50 of instructions and a 200-token
        // prompt leaves 500 for history, not 700.
        let ordinary = ContextBudget(window: 1000, reserve: 250, instructions: 50, prompt: 200)
        t.check(ordinary.fits, "an ordinary prompt fits")
        t.check(ordinary.room == 500,
                "the prompt's own tokens come out of the window (room \(ordinary.room), want 500)")

        // Prior turns are dropped oldest first: the walk stops at the first
        // turn that does not fit rather than sieving the small ones out of the
        // middle of the conversation.
        t.check(kept(ordinary, costsNewestFirst: [200, 200, 200, 10]) == 2,
                "the newest turns that fit are kept and the walk stops at the first that does not")
        t.check(kept(ordinary, costsNewestFirst: [500]) == 1, "a turn that exactly fills the room is kept")
        t.check(kept(ordinary, costsNewestFirst: [501]) == 0, "a turn one token too large is dropped")

        // A long conversation with a short prompt still answers: that is the
        // built-in model's shipped promise (itd-2609170718438919, ac-4), and
        // budgeting the prompt must not turn it into a refusal.
        let shortPrompt = ContextBudget(window: 4096, reserve: 1024, instructions: 60, prompt: 12)
        t.check(shortPrompt.fits, "a short prompt after a long conversation still fits")
        t.check(kept(shortPrompt, costsNewestFirst: Array(repeating: 100, count: 100)) == 30,
                "a hundred turns of a hundred tokens are trimmed to the thirty that fit")

        // A prompt that overflows the window on its own is refused rather than
        // sent: no amount of dropped history makes room for it.
        let overlong = ContextBudget(window: 4096, reserve: 1024, instructions: 60, prompt: 4000)
        t.check(!overlong.fits, "a prompt larger than the window less its reserve does not fit")
        t.check(kept(overlong, costsNewestFirst: [1, 1, 1]) == 0,
                "a prompt that does not fit keeps no history either")

        // The boundary: a prompt that fills the room exactly, to the last
        // token, is sent with no history rather than refused.
        let exact = ContextBudget(window: 4096, reserve: 1024, instructions: 60, prompt: 3012)
        t.check(exact.fits, "a prompt that fills the room to the last token is sent")
        t.check(exact.room == 0, "and leaves no room for history (room \(exact.room), want 0)")
        let oneOver = ContextBudget(window: 4096, reserve: 1024, instructions: 60, prompt: 3013)
        t.check(!oneOver.fits, "one token more and the turn is not sent")

        // The reply's reserve is real room, not a hint: a prompt that would
        // fit the raw window but leave nothing to answer with is refused.
        let noRoomToAnswer = ContextBudget(window: 4096, reserve: 1024, instructions: 60, prompt: 4030)
        t.check(!noRoomToAnswer.fits, "a prompt that leaves no room for the reply is not sent")

        if t.failures.isEmpty {
            print("\nall \(t.passed) checks passed")
        } else {
            print("\n\(t.failures.count) check(s) failed:")
            for f in t.failures { print("  - \(f)") }
            exit(1)
        }
    }
}

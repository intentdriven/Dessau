// What a sidebar card's exchange count promises, checked on its own:
// `client/tests/card-summary.sh` compiles this with
// `client/DessauChat/CardSummary.swift` and runs it. The client has no test
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

/// The conversation's turns, written the way they read: `p` for the person's
/// message, `a` for a reply.
private func turns(_ shape: String) -> [Bool] {
    shape.map { $0 == "p" }
}

@main
enum CardSummaryTests {
    static func main() {
        var t = Checks()

        t.check(exchangeCount(fromPerson: turns("")) == 0,
                "an empty conversation holds no exchanges")
        t.check(exchangeCount(fromPerson: turns("pa")) == 1,
                "a message and its reply are one exchange")
        t.check(exchangeCount(fromPerson: turns("papapa")) == 3,
                "three messages, each answered, are three exchanges")

        // The half turns: each is one message short of an exchange, and the
        // count that took the replies alone got both of these wrong.
        t.check(exchangeCount(fromPerson: turns("p")) == 0,
                "a message still waiting for its reply is not yet an exchange")
        t.check(exchangeCount(fromPerson: turns("papap")) == 2,
                "the message just sent does not add an exchange until it is answered")
        t.check(exchangeCount(fromPerson: turns("a")) == 0,
                "a reply with no message of the person's before it is no exchange")

        // A reply can arrive in more than one part, and a person can send
        // twice before an answer comes; neither makes a second exchange.
        t.check(exchangeCount(fromPerson: turns("paa")) == 1,
                "a second reply with no new message between is still one exchange")
        t.check(exchangeCount(fromPerson: turns("ppa")) == 1,
                "two messages answered once are one exchange")
        t.check(exchangeCount(fromPerson: turns("ppaa")) == 1,
                "the reply that follows an answered pair closes nothing new")

        print("\(t.passed) passed, \(t.failures.count) failed")
        if !t.failures.isEmpty {
            for f in t.failures { print("broken: \(f)") }
            exit(1)
        }
    }
}

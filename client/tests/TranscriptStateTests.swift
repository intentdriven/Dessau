// What the picker's transcript icon promises, checked on its own:
// `client/tests/transcript-state.sh` compiles this with
// `client/DessauChat/TranscriptState.swift` and runs it. The client has no
// test target, so this is a main that prints each promise and exits non-zero
// on the first one that is broken.

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

@main
enum TranscriptStateTests {
    static func main() {
        var t = Checks()

        // The decode: an entry that says nothing about recording carries no
        // state, an entry that says false or true carries that.
        let states = transcriptStates([
            ("org/Recorded", true),
            ("org/Quiet", false),
            ("org/Older", nil),
        ])
        t.check(transcriptRecorded("org/Recorded", in: states) == true,
                "an entry with recording true is recorded")
        t.check(transcriptRecorded("org/Quiet", in: states) == false,
                "an entry with recording false keeps no transcript")
        t.check(transcriptRecorded("org/Older", in: states) == nil,
                "an entry that says nothing carries no state, so no icon is drawn")
        t.check(transcriptRecorded("org/Unknown", in: states) == nil,
                "a model the list did not name carries no state")

        // The join is folded on both sides, as every join on a repo id is:
        // the server's spelling and the client's stored one may differ in
        // case, and the icon has to be right for either.
        t.check(transcriptRecorded("ORG/QUIET", in: states) == false,
                "a lookup under another spelling finds the same state")
        let spelled = transcriptStates([("ORG/Quiet", false)])
        t.check(transcriptRecorded("org/quiet", in: spelled) == false,
                "a state published under another spelling is found")

        // An empty list carries nothing, and nothing errors.
        t.check(transcriptStates([]).isEmpty,
                "an empty models list carries no state")

        print("\(t.passed) passed, \(t.failures.count) failed")
        if !t.failures.isEmpty { exit(1) }
    }
}

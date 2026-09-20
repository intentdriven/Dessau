// What the sidebar's search promises, checked on its own:
// `client/tests/sidebar-search.sh` compiles this with
// `client/DessauChat/SidebarSearch.swift` and runs it. The client has no test
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

/// One conversation, the way the sidebar hands it to the search: a title and
/// the text of every message in it.
private func shows(_ query: String, title: String, messages: [String]) -> Bool {
    SidebarSearch.matches(words: SidebarSearch.words(in: query), title: title, messages: messages)
}

@main
enum SidebarSearchTests {
    static func main() {
        var t = Checks()

        let holiday = (title: "Trip to Lisbon",
                       messages: ["What should we pack for the boat?", "A jumper, and the blue bag."])

        // An empty search, and a search of nothing but spaces, leave every
        // conversation: the field is cleared by deleting what was typed, and
        // the last delete must not leave the list empty.
        t.check(shows("", title: holiday.title, messages: holiday.messages),
                "an empty search shows every conversation")
        t.check(shows("   ", title: holiday.title, messages: holiday.messages),
                "a search of nothing but spaces shows every conversation")

        // One word is found in the title or in any message, as it always was.
        t.check(shows("lisbon", title: holiday.title, messages: holiday.messages),
                "a word of the title is found, whatever its case")
        t.check(shows("jumper", title: holiday.title, messages: holiday.messages),
                "a word of a message is found")
        t.check(!shows("bicycle", title: holiday.title, messages: holiday.messages),
                "a word in neither the title nor the messages is not found")

        // The promise this exists for (iss-2609181213191220): every word of a
        // multi-word search is looked for on its own, so words that are in the
        // conversation apart still find it.
        t.check(shows("boat jumper", title: holiday.title, messages: holiday.messages),
                "two words found in different messages match, though they are never adjacent")
        t.check(shows("lisbon blue", title: holiday.title, messages: holiday.messages),
                "a word of the title and a word of a message match together")
        t.check(shows("blue bag", title: holiday.title, messages: holiday.messages),
                "words that ARE adjacent still match, in the order they are typed")
        t.check(shows("bag blue", title: holiday.title, messages: holiday.messages),
                "and in the other order, which a substring search could never do")

        // Every word has to be there: the search narrows the list, so a word
        // that is nowhere rules the conversation out however many others match.
        t.check(!shows("boat bicycle", title: holiday.title, messages: holiday.messages),
                "one word missing rules the conversation out, however many match")

        // The words are what is typed between the spaces, however many there
        // are and wherever they fall.
        t.check(shows("  boat   jumper  ", title: holiday.title, messages: holiday.messages),
                "runs of spaces around and between the words are not words themselves")
        t.check(SidebarSearch.words(in: "  boat   jumper  ") == ["boat", "jumper"],
                "the query splits into its words")
        t.check(SidebarSearch.words(in: "boat\njumper") == ["boat", "jumper"],
                "a newline separates words as a space does")
        t.check(SidebarSearch.words(in: "   ").isEmpty, "a query of spaces has no words")

        // Case and accents are not what a search is about: the match is the
        // system's standard one, as it is in Finder.
        let cafe = (title: "Café plans", messages: ["Meet at the Pontão at six."])
        t.check(shows("CAFE PONTAO", title: cafe.title, messages: cafe.messages),
                "words match without their accents, and in any case")
        t.check(shows("café pontão", title: cafe.title, messages: cafe.messages),
                "and with them")

        // A conversation with nothing in it yet is not matched by a word.
        t.check(!shows("boat", title: "", messages: []),
                "an empty conversation is not matched by a word")

        if t.failures.isEmpty {
            print("\nall \(t.passed) checks passed")
        } else {
            print("\n\(t.failures.count) check(s) failed:")
            for f in t.failures { print("  - \(f)") }
            exit(1)
        }
    }
}

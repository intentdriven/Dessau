// The sidebar's search, on its own. The client has no test target, so the one
// piece of this that can be got wrong invisibly — which chats a typed query
// leaves — lives here, in a file that imports Foundation and names nothing of
// the client's, so `swiftc` compiles it beside a main and runs it
// (client/tests/sidebar-search.sh).

import Foundation

/// Which chats a sidebar search leaves. The search is over words: the query is
/// split on its spaces and a chat is left when every word is somewhere in its
/// title or in one of its messages, which need not be the same place and need
/// not be adjacent. Matching the whole query as one substring would find
/// "blue bag" and never "bag blue", and never a word of the title with a word
/// of an answer (iss-2609181213191220).
enum SidebarSearch {
    /// The words of a query: what was typed between the spaces, in order.
    /// Spaces and newlines around and between the words are separators, never
    /// words of their own, so a query of nothing but spaces has no words.
    static func words(in query: String) -> [String] {
        query.split(whereSeparator: \.isWhitespace).map(String.init)
    }

    /// Whether a chat with this title and these message texts is left by a
    /// search for these words. No words is an empty search, which leaves every
    /// chat. The comparison is the platform's standard one, as Finder's is:
    /// neither case nor accents are what somebody looking for a chat means.
    static func matches(words: [String], title: String, messages: [String]) -> Bool {
        words.allSatisfy { word in
            title.localizedStandardContains(word)
                || messages.contains { $0.localizedStandardContains(word) }
        }
    }
}

// The arithmetic behind a sidebar card's summary line, and nothing else: no
// framework types, nothing imported, so it compiles on its own and can be
// checked on its own. `client/tests/card-summary.sh` compiles this file with a
// small main and runs it; a test in the server's suite runs that script.
//
// The card says how many exchanges a conversation holds, and an exchange is a
// person's message and the reply that answers it — a pair, not a reply
// (itd-2609181104490133, iss-2609181213194439).

/// How many exchanges a conversation holds, from its turns in order, each
/// flagged as the person's or not.
///
/// The turns are walked from the first: a message of the person's opens an
/// exchange and the next reply closes it. So a message still waiting for its
/// answer counts for nothing until the answer arrives, a reply that follows no
/// message of the person's counts for nothing at all, and a reply that arrives
/// in two parts is still the one exchange.
nonisolated func exchangeCount(fromPerson turns: [Bool]) -> Int {
    var exchanges = 0
    var awaitingReply = false
    for fromPerson in turns {
        if fromPerson {
            awaitingReply = true
        } else if awaitingReply {
            exchanges += 1
            awaitingReply = false
        }
    }
    return exchanges
}

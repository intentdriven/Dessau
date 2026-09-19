// The context arithmetic for the device's own model, and nothing else: no
// framework types, nothing imported, so it compiles on its own and can be
// checked on its own. `client/tests/context-budget.sh` compiles this file with
// a small main and runs it; a test in the server's suite runs that script.
//
// The framework counts the tokens; this decides what they are spent on. The
// four claims on the window are the client's instructions, the conversation so
// far, the message just typed and the room the reply needs — and the message
// just typed is counted with the rest, not after them, so a message that
// cannot fit is found before it is sent (iss-2609181116072455).

/// One turn's share-out of the model's context window, driven by the caller as
/// it walks the prior turns from the newest backwards.
struct ContextBudget {
    /// What is left for the prior turns once the instructions, the new prompt
    /// and the reply's reserve have taken their share. Negative when the new
    /// prompt does not fit at all.
    let room: Int
    private var used = 0

    init(window: Int, reserve: Int, instructions: Int, prompt: Int) {
        self.room = window - reserve - instructions - prompt
    }

    /// False when the new prompt cannot fit beside the instructions and the
    /// room the reply needs — no amount of dropped history makes room for it,
    /// so the turn is not sent at all.
    var fits: Bool { room >= 0 }

    /// True when a prior turn of this cost still fits, and takes it; false
    /// when it does not, and the caller stops there, so the turns that are
    /// dropped are the oldest rather than whichever happen to be small.
    mutating func take(_ cost: Int) -> Bool {
        guard fits, used + cost <= room else { return false }
        used += cost
        return true
    }
}

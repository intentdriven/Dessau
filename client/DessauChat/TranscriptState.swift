// The rule behind the picker's transcript icon, and nothing else: no
// framework types, nothing imported, so it compiles on its own and can be
// checked on its own. `client/tests/transcript-state.sh` compiles this file
// with a small main and runs it; a test in the server's suite runs that
// script.
//
// A Dessau server says, per models-list entry, whether a conversation with
// that model is written down on the Mac that runs it — the `recording`
// field (itd-2609091715089488). The picker shows that beside each model
// before it is chosen, so the person knows rather than asking. A server that
// says nothing — an older one, or any other OpenAI-compatible server — draws
// no icon at all, because an absent fact is not a promise.

/// The transcript state of every model that carries one, keyed by the folded
/// repo id: `true` for a model whose conversations are recorded on the
/// server, `false` for one that keeps no transcript. An entry that says
/// nothing is left out.
///
/// Folded — lower-cased — on the way in, as the server folds every join on
/// a repo id: the spelling the list publishes and the spelling the client
/// stored may differ in case, and the icon has to be right for either.
nonisolated func transcriptStates(_ entries: [(id: String, recording: Bool?)]) -> [String: Bool] {
    var out: [String: Bool] = [:]
    for entry in entries {
        if let recording = entry.recording {
            out[entry.id.lowercased()] = recording
        }
    }
    return out
}

/// Whether a conversation with this model is recorded on the server, or nil
/// when the server did not say — and nil draws no icon.
nonisolated func transcriptRecorded(_ id: String, in states: [String: Bool]) -> Bool? {
    states[id.lowercased()]
}

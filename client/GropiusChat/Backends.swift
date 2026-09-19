// The two things that can answer a message: the Mac's own model, through the
// Foundation Models framework, and a Gropius server over its OpenAI-compatible
// endpoint. Both sit behind one seam so the rest of the client neither knows
// nor cares which one is answering.
//
// The built-in backend imports no networking and builds no request: that is
// the whole promise ("nothing sent anywhere"), and a test in the server's suite
// reads this file to hold it.

import Foundation
import FoundationModels

/// Which of the two answers. One choice for the whole client, as the model
/// choice always was; typed, so a server model that happens to be called
/// "On this Mac" can never be mistaken for the device's own.
enum Answerer: Equatable {
    case builtIn
    case server(model: String)

    /// The name the picker button and the transcript show.
    var displayName: String {
        switch self {
        case .builtIn: return BuiltInBackend.displayName
        case .server(let model):
            return model.split(separator: "/").last.map(String.init) ?? model
        }
    }

    /// What a message records as its author.
    var recordedName: String {
        switch self {
        case .builtIn: return BuiltInBackend.recordedName
        case .server(let model): return model
        }
    }
}

/// What a backend says while it answers.
enum ReplyEvent {
    /// The server is loading the model; nothing has been generated yet.
    case loading
    /// Reply text. `replaces` says whether this is the whole text so far (the
    /// Mac's model streams snapshots) or a chunk to append (a server streams
    /// deltas).
    case text(String, replaces: Bool)
    /// A thinking model's reasoning, always a chunk to append.
    case reasoning(String)
}

/// One answerer. `reply` runs to completion or throws; cancellation of the
/// surrounding task is how a reply is stopped.
protocol ChatBackend {
    func reply(to history: [Message], deliver: @escaping (ReplyEvent) -> Void) async throws
}

/// An error a backend wants shown in the reply row, in its own words.
struct BackendMessage: LocalizedError {
    let text: String
    var errorDescription: String? { text }
}

// MARK: - The Mac's own model

/// The Foundation Models framework: the language model Apple ships with the
/// system, on the Mac and on an eligible iPad, on the device only.
struct BuiltInBackend: ChatBackend {
    // The device the person is holding, in the words they would use for it.
    // The same framework answers on both, but "On this Mac" on an iPad is a
    // name for something that is not there.
    #if os(macOS)
    static let deviceNoun = "Mac"
    /// What the system's own settings app is called on this system.
    static let systemSettings = "System Settings"
    #else
    static let deviceNoun = "iPad"
    static let systemSettings = "Settings"
    #endif
    static let displayName = "On this \(deviceNoun)"
    /// Recorded on each reply it writes; never a name a server could serve.
    static let recordedName = "On this \(deviceNoun) (Apple Intelligence)"

    /// The client's own words, trusted; the person's words go only in the
    /// prompt. Instructions win over a prompt, which is what keeps a prompt
    /// from rewriting the rules.
    static let instructions = """
        You are a helpful assistant in a chat app on this \(deviceNoun). Answer \
        plainly, \
        in the language the person writes in. Use markdown for structure when \
        it helps: lists, headings, code blocks with a language.
        """

    /// Why the Mac cannot answer, in plain words, with what would fix it.
    struct Unavailable: Equatable {
        let reason: String
        let fix: String
    }

    /// The Mac's answer right now: nil when the model can answer.
    static func unavailability() -> Unavailable? {
        let model = SystemLanguageModel.default
        switch model.availability {
        case .available:
            if !model.supportsLocale() {
                return Unavailable(
                    reason: "The \(deviceNoun)'s own model does not support this language.",
                    fix: "Change the language in \(systemSettings), or pick a server from the model picker.")
            }
            return nil
        case .unavailable(let why):
            switch why {
            case .appleIntelligenceNotEnabled:
                return Unavailable(
                    reason: "Apple Intelligence is switched off, so the \(deviceNoun) cannot answer.",
                    fix: "Turn it on in \(systemSettings), or pick a server from the model picker.")
            case .modelNotReady:
                return Unavailable(
                    reason: "The \(deviceNoun)'s own model is not ready yet.",
                    fix: "It is still being downloaded; try again shortly, or pick a server from the model picker.")
            case .deviceNotEligible:
                return Unavailable(
                    reason: "This \(deviceNoun) cannot run the system's own model.",
                    fix: "Pick a server from the model picker.")
            @unknown default:
                return Unavailable(
                    reason: "The Mac's own model is not available.",
                    fix: "Pick a server from the model picker.")
            }
        }
    }

    func reply(to history: [Message], deliver: @escaping (ReplyEvent) -> Void) async throws {
        let model = SystemLanguageModel.default
        guard let last = history.last, last.role == .user else { return }
        let prior = Array(history.dropLast())

        var reserve = max(512, model.contextSize / 4)
        for attempt in 0..<2 {
            let entries = try await Self.trimmed(prior, prompt: last.text, model: model, reserve: reserve)
            let session = LanguageModelSession(model: model, transcript: Transcript(entries: entries))
            do {
                let stream = session.streamResponse(to: last.text)
                for try await partial in stream {
                    try Task.checkCancellation()
                    deliver(.text(partial.content, replaces: true))
                }
                return
            } catch let error as LanguageModelError {
                switch error {
                case .contextSizeExceeded where attempt == 0:
                    // Once more with less history; the second time it is shown.
                    reserve = model.contextSize / 2
                    continue
                default:
                    throw BackendMessage(text: try await Self.words(for: error))
                }
            } catch let error as LanguageModelSession.Error {
                throw BackendMessage(text: Self.words(for: error))
            } catch let error as SystemLanguageModel.Error {
                throw BackendMessage(text: Self.words(for: error))
            }
        }
    }

    /// What the transcript says when the message just typed is longer than the
    /// window can hold on its own. Nothing is sent: there is no history to
    /// drop that would make room for it.
    static let promptTooLong =
        "That message is too long for the \(deviceNoun)'s own model to read in one go — shorten it, or pick a server from the model picker."

    /// The instructions plus the most recent turns that fit beside the new
    /// prompt, oldest dropped first, so the model sees what the person sees
    /// minus the start. The new prompt is budgeted with the prior turns rather
    /// than after them; when it does not fit on its own, this throws the
    /// client's own sentence and the turn is never sent.
    static func trimmed(_ history: [Message], prompt: String, model: SystemLanguageModel, reserve: Int) async throws -> [Transcript.Entry] {
        let instructions = Transcript.Entry.instructions(
            Transcript.Instructions(segments: [.text(Transcript.TextSegment(content: Self.instructions))],
                                    toolDefinitions: []))
        let promptEntry = Transcript.Entry.prompt(
            Transcript.Prompt(segments: [.text(Transcript.TextSegment(content: prompt))]))
        var budget = ContextBudget(
            window: model.contextSize,
            reserve: reserve,
            instructions: try await model.tokenCount(for: [instructions]),
            prompt: try await model.tokenCount(for: [promptEntry]))
        guard budget.fits else { throw BackendMessage(text: Self.promptTooLong) }

        var kept: [Transcript.Entry] = []
        for m in history.reversed() where !m.text.isEmpty {
            let entry: Transcript.Entry
            switch m.role {
            case .user:
                entry = .prompt(Transcript.Prompt(segments: [.text(Transcript.TextSegment(content: m.text))]))
            case .assistant:
                entry = .response(Transcript.Response(assetIDs: [], segments: [.text(Transcript.TextSegment(content: m.text))]))
            }
            let cost = try await model.tokenCount(for: [entry])
            if !budget.take(cost) { break }
            kept.append(entry)
        }
        // A transcript must not start with a response; drop a leading one.
        var turns = Array(kept.reversed())
        if case .response = turns.first { turns.removeFirst() }
        return [instructions] + turns
    }

    private static func words(for error: LanguageModelError) async throws -> String {
        switch error {
        case .contextSizeExceeded:
            return "This conversation is longer than the Mac's own model can hold. Start a new chat, or pick a server from the model picker."
        case .guardrailViolation:
            return "The Mac's own model will not answer that."
        case .refusal(let refusal):
            let why = (try? await refusal.explanation.content) ?? ""
            return why.isEmpty ? "The Mac's own model declined to answer." : why
        case .rateLimited:
            return "The Mac's own model is busy; try again in a moment."
        case .unsupportedLanguageOrLocale:
            return "The Mac's own model does not support this language."
        default:
            return error.localizedDescription
        }
    }

    private static func words(for error: LanguageModelSession.Error) -> String {
        switch error {
        case .concurrentRequests:
            return "The Mac's own model is still answering the last message."
        default:
            return error.localizedDescription
        }
    }

    private static func words(for error: SystemLanguageModel.Error) -> String {
        switch error {
        case .assetsUnavailable:
            return "The Mac's own model is not ready yet; try again shortly."
        default:
            return error.localizedDescription
        }
    }
}

// MARK: - A Gropius server

/// The OpenAI-compatible streaming endpoint a Gropius server exposes.
struct ServerBackend: ChatBackend {
    let model: String
    /// Builds an authenticated request for a path under the API base, or nil
    /// when the stored address is not one a request may be sent to.
    let request: (String) -> URLRequest?

    func reply(to history: [Message], deliver: @escaping (ReplyEvent) -> Void) async throws {
        guard var req = request("/chat/completions") else {
            throw BackendMessage(text: "That server address is not valid.")
        }
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let messages: [[String: String]] = history.map {
            ["role": $0.role == .user ? "user" : "assistant", "content": $0.text]
        }
        let body: [String: Any] = [
            "model": model,
            "messages": messages,
            "stream": true,
            "max_tokens": 2048,
        ]
        req.httpBody = try? JSONSerialization.data(withJSONObject: body)

        let (bytes, resp) = try await URLSession.shared.bytes(for: req)
        if let http = resp as? HTTPURLResponse, http.statusCode != 200 {
            if http.statusCode == 401 {
                throw BackendMessage(text: "This server needs an API key; add it in Settings.")
            }
            throw BackendMessage(text: "Server returned HTTP \(http.statusCode).")
        }
        // Assemble SSE lines from the raw byte stream under hard caps, rather
        // than using bytes.lines: a hostile or broken server can stream an
        // unbounded body with no newline, and bytes.lines would buffer it
        // without limit. maxLineBytes bounds a single line; maxTotalBytes
        // bounds the whole response.
        let maxLineBytes = 1 << 20   // 1 MiB per SSE line
        let maxTotalBytes = 64 << 20 // 64 MiB per response
        var lineBuf = [UInt8]()
        var total = 0

        // An SSE comment, stripped of its colon and the optional space after
        // it. Both spellings of the loading marker are the same comment.
        func comment(_ line: String) -> String {
            String(line.dropFirst()).trimmingCharacters(in: .whitespaces)
        }
        let loadingMarker = comment(modelLoadingComment)

        // Returns true at the end of the stream.
        func handle(_ line: String) -> Bool {
            // A line beginning with a colon is a comment, which the SSE format
            // says to ignore. Exactly one is not ignored: the load marker. A
            // bare ":" keep-alive passes without touching anything.
            if line.hasPrefix(":") {
                if comment(line).hasPrefix(loadingMarker) { deliver(.loading) }
                return false
            }
            guard line.hasPrefix("data:") else { return false }
            var payload = String(line.dropFirst(5))
            if payload.hasPrefix(" ") { payload.removeFirst() }
            if payload == "[DONE]" { return true }
            guard let d = payload.data(using: .utf8),
                  let chunk = try? JSONDecoder().decode(StreamChunk.self, from: d),
                  let delta = chunk.choices.first?.delta else { return false }
            if let c = delta.content, !c.isEmpty { deliver(.text(c, replaces: false)) }
            if let r = delta.reasoning ?? delta.reasoning_content, !r.isEmpty {
                deliver(.reasoning(r))
            }
            return false
        }

        for try await b in bytes {
            try Task.checkCancellation()
            total += 1
            if total > maxTotalBytes {
                throw BackendMessage(text: "Response exceeded \(maxTotalBytes >> 20) MB — stopped.")
            }
            if b == 0x0A { // LF: end of an SSE line
                if let line = String(bytes: lineBuf, encoding: .utf8), handle(line) { break }
                lineBuf.removeAll(keepingCapacity: true)
                continue
            }
            if b == 0x0D { continue } // ignore CR so CRLF is handled
            // Past the per-line cap, drop bytes until the next newline rather
            // than buffer an unbounded line.
            if lineBuf.count < maxLineBytes { lineBuf.append(b) }
        }
    }
}

/// One streamed chunk from /v1/chat/completions with stream=true.
struct StreamChunk: Decodable {
    struct Choice: Decodable {
        struct Delta: Decodable {
            let content: String?
            let reasoning: String?
            let reasoning_content: String?
        }
        let delta: Delta
    }
    let choices: [Choice]
}

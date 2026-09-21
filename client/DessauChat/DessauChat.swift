// DessauChat — a native chat app for the Mac and the iPad.
//
// It chats with the Mac's own model out of the box and, when a Dessau server
// is on the network, offers that server's models in the picker: GET /v1/models
// to list them, POST /v1/chat/completions (streaming) to chat. Servers are
// found over Bonjour while the picker is open. Conversations are kept in a
// sidebar and persisted to disk.
//
// Built without an Xcode project: several Swift files, one script for each
// system (build.sh, build-ipad.sh), the installed Xcode's toolchain. This file
// holds the app, the model and the views; the answerers, the discovery, the
// picker, the markdown, the effects and the App Intents each have a file of
// their own.
//
// One source, two systems: the four places where the systems differ are
// guarded below, and each says what it is standing in for.

#if os(macOS)
import AppKit
#else
import UIKit
#endif
import Network
import SwiftUI
import Security
import UniformTypeIdentifiers

// MARK: - Keychain

/// Keychain-backed storage for the one secret this app holds: the server API
/// key. Storing it in UserDefaults (as an earlier build did) leaves it in
/// cleartext in the preferences plist, readable by any process running as the
/// user and by anything that syncs or backs up the home directory. The Keychain
/// gates it behind the login-keychain ACL instead, and the item never leaves
/// the device it was entered on.
enum Keychain {
    private static let service = "sh.intentdriven.dessau.chat"
    private static let account = "apiKey"

    private static var baseQuery: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
    }

    static func read() -> String {
        var query = baseQuery
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var out: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &out) == errSecSuccess,
              let data = out as? Data,
              let value = String(data: data, encoding: .utf8)
        else { return "" }
        return value
    }

    static func write(_ value: String) {
        // Empty means "no key" — remove the item rather than store an empty secret.
        if value.isEmpty {
            SecItemDelete(baseQuery as CFDictionary)
            return
        }
        let attrs: [String: Any] = [
            kSecValueData as String: Data(value.utf8),
            // ThisDeviceOnly: another person's server key is not the kind of
            // thing to travel in an encrypted backup and come back on a
            // different device. It is re-entered on a new device instead.
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        ]
        if SecItemUpdate(baseQuery as CFDictionary, attrs as CFDictionary) == errSecItemNotFound {
            var add = baseQuery
            add.merge(attrs) { _, new in new }
            SecItemAdd(add as CFDictionary, nil)
        }
    }
}

// MARK: - Model types

/// One turn in a conversation.
struct Message: Identifiable, Codable, Equatable {
    enum Role: String, Codable { case user, assistant }
    var id = UUID()
    var role: Role
    var text: String = ""
    /// Thinking models (e.g. Qwen3) stream their reasoning separately; we keep it
    /// so a reply that spends its whole budget reasoning is not shown as blank.
    var reasoning: String = ""
    /// Who answered, for the assistant's turns: a server model's id, or the
    /// Mac's own model. VoiceOver reads it out — the side a message sits on is
    /// not available to a screen reader, so the speaker has to be in the label.
    ///
    /// Optional on purpose: Codable's synthesized decoder falls back for a
    /// missing key only on an optional property, and a saved conversation
    /// written before this field existed must still load.
    var model: String? = nil
}

/// A saved chat: a title plus its messages.
struct Conversation: Identifiable, Codable {
    var id = UUID()
    var title: String = "New Chat"
    var messages: [Message] = []
    var createdAt = Date()
}

/// The residency value the models list carries for a model that is already in
/// memory. The server publishes three — `loaded`, `loading`, `not_loaded` — and
/// only this one means a request is served without a wait. A test in the
/// server's suite holds it to the value the pool reports.
let residencyLoaded = "loaded"

/// The SSE comment the server sends, about once a second, while it is loading a
/// model to serve a streaming request; the data frames follow once the model is
/// up. The wire form's one home is `gateway.LoadingComment` on the server; a
/// test in the server's suite holds this declaration to it.
let modelLoadingComment = ": loading"

/// GET /v1/models
private struct ModelsResponse: Decodable {
    struct Model: Decodable {
        let id: String
        /// Residency: `loaded`, `loading` or `not_loaded`. Absent on a server
        /// that does not publish residency to this client, which is not the
        /// same as "not loaded" — it is "not said", and nothing is claimed
        /// from it.
        let state: String?
        /// Whether the model can serve a chat request at all, under the
        /// server's own rule. Absent on a server that does not publish the
        /// capability.
        let chat: Bool?
        /// Whether a conversation with this model is written down on the Mac
        /// that runs the server (itd-2609091715089488). Absent on a server
        /// that does not publish it, which is "not said" rather than either
        /// answer: the picker draws no icon from it.
        let recording: Bool?
        /// What HuggingFace says this model is: the repo's pipeline tag and its
        /// tags, recorded by the server when the model was downloaded.
        let pipeline_tag: String?
        let tags: [String]?

        /// Absent means yes. A server that publishes no capability is an older
        /// one, and every model it serves must still be offered.
        var chattable: Bool { chat ?? true }
    }
    let data: [Model]
}

/// The client's own rule for which served models it offers, read from the
/// words the models list publishes. The Mac's own model is not a served model
/// and is not judged by it.
///
/// It ships with the server's default — a test in the server's suite holds the
/// two together — and a person changes it in Settings.
struct ChatRule {
    let pipelineTags: [String]
    let requiredTags: [String]

    init(pipelineTags: String, requiredTags: String) {
        self.pipelineTags = ChatRule.words(pipelineTags)
        self.requiredTags = ChatRule.words(requiredTags)
    }

    private static func words(_ field: String) -> [String] {
        field.split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespaces).lowercased() }
            .filter { !$0.isEmpty }
    }

    /// Whether this model belongs in the picker. A model the server publishes
    /// no words for is judged by the server's own verdict; a server that
    /// publishes neither offers everything it serves.
    fileprivate func offers(_ m: ModelsResponse.Model) -> Bool {
        guard m.pipeline_tag != nil || m.tags != nil else { return m.chattable }
        if !pipelineTags.isEmpty {
            let tag = (m.pipeline_tag ?? "").trimmingCharacters(in: .whitespaces).lowercased()
            if !pipelineTags.contains(tag) { return false }
        }
        let have = Set((m.tags ?? []).map { $0.trimmingCharacters(in: .whitespaces).lowercased() })
        return requiredTags.allSatisfy { have.contains($0) }
    }
}

// MARK: - App model

final class AppModel: ObservableObject {
    /// The one instance: the windows hold it, and the App Intents reach the
    /// same object, so the conversations file has one writer.
    static let shared = AppModel()

    /// The base path the OpenAI-compatible API is mounted under.
    static let defaultAPIPath = "/v1"

    // The stored server. The default address is this Mac's own server, the
    // one address that is right before the person has picked or typed
    // anything; nothing connects to it until a server model is chosen.
    @AppStorage("serverURL") var serverURL: String = "http://localhost:11535"
    @AppStorage("serverPath") var serverPath: String = AppModel.defaultAPIPath
    @AppStorage("selectedModel") var selectedModel: String = ""
    /// Which answerer: "builtin" (the default) or "server".
    @AppStorage("answerer") var answererKind: String = "builtin"
    /// The origin (scheme, host and port) the API key was entered for; the
    /// key is sent to no other.
    @AppStorage("apiKeyHost") var apiKeyHost: String = ""
    /// Whether certain words in a reply animate once.
    @AppStorage("effectsEnabled") var effectsEnabled: Bool = true

    /// The pairing this client holds, read from the Keychain at launch.
    ///
    /// In the Keychain rather than in UserDefaults, for the reason the API key
    /// already is: a preferences plist is rewritable by anything running as the
    /// user, and a pin anything can rewrite is not a pin.
    @Published var pairedServer: PairedServer?

    // Which served models the picker offers, as two comma-separated lists of
    // HuggingFace's own words. The defaults are the server's shipped rule; a
    // test in the server's suite holds them to it.
    @AppStorage("chatPipelineTags") var chatPipelineTags: String = "text-generation, image-text-to-text"
    @AppStorage("chatRequiredTags") var chatRequiredTags: String = "conversational"

    var chatRule: ChatRule {
        ChatRule(pipelineTags: chatPipelineTags, requiredTags: chatRequiredTags)
    }

    /// The bearer token, persisted to the Keychain, never UserDefaults.
    @Published var apiKey: String = ""

    @Published var conversations: [Conversation] = []
    /// The chat the App Intents and single-window flows act on; each window
    /// keeps its own selection in scene storage.
    @Published var selectedID: UUID?
    /// A chat an App Intent asked to show: the key window follows this, and
    /// only this, so two windows can otherwise show two chats.
    struct IntentSelection: Equatable {
        let id: UUID
        let at = Date()
    }
    @Published var intentSelection: IntentSelection?

    /// Every model the server serves, and the subset the picker offers.
    @Published var models: [String] = []
    @Published var chatModels: [String] = []
    /// Whether a conversation with each served model is recorded on the
    /// server, by folded repo id, for the models whose entry said
    /// (TranscriptState.swift). The picker's icon reads this.
    @Published var recordedModels: [String: Bool] = [:]
    @Published var status: String = "No server chosen"
    @Published var connected: Bool = false
    @Published var connecting: Bool = false
    @Published var sending: Bool = false
    /// The chat a reply is streaming into, while one is.
    @Published var sendingIn: UUID?
    /// Why the Mac's own model cannot answer right now, or nil when it can.
    @Published var builtInUnavailable: BuiltInBackend.Unavailable?
    /// The server answered the last models request with 401: it wants a key
    /// the client does not hold for it. The picker asks for one on the spot.
    @Published var needsAPIKey = false
    /// Replies whose words are due to animate: added when a reply finishes
    /// with a match, removed the moment a row starts drawing it.
    @Published var effectsToPlay: Set<UUID> = []

    /// What the client is waiting for once a message has been sent: nothing
    /// known, a server loading a model, or a reply being generated.
    enum Activity: Equatable { case idle, loading, generating }
    @Published var activity: Activity = .idle

    private var replyTask: Task<Void, Never>?
    private var residencyTask: Task<Void, Never>?

    init() {
        // One session for the whole app, carrying the delegate that answers the
        // two TLS challenges a paired connection raises. URLSession.shared
        // takes no delegate at all, which is why nothing here uses it any more.
        let pinning = PinningDelegate()
        self.pinning = pinning
        self.session = URLSession(configuration: .default, delegate: pinning, delegateQueue: nil)
        if let legacy = UserDefaults.standard.string(forKey: "apiKey"), !legacy.isEmpty {
            Keychain.write(legacy)
            UserDefaults.standard.removeObject(forKey: "apiKey")
        }
        apiKey = Keychain.read()
        pairedServer = PairingStore.read()
        pinning.pinnedSPKI = pairedServer?.pinnedSPKI
        load()
        if conversations.isEmpty {
            let c = Conversation()
            conversations = [c]
            selectedID = c.id
        } else {
            selectedID = conversations.first?.id
        }
        builtInUnavailable = BuiltInBackend.unavailability()
        if answererKind == "server" { Task { await connect() } }
    }

    // MARK: Who answers

    var answerer: Answerer {
        if answererKind == "server", !selectedModel.isEmpty { return .server(model: selectedModel) }
        return .builtIn
    }

    func chooseBuiltIn() {
        answererKind = "builtin"
        objectWillChange.send()
    }

    func chooseServerModel(_ id: String) {
        selectedModel = id
        answererKind = "server"
        objectWillChange.send()
    }

    func refreshBuiltInAvailability() {
        builtInUnavailable = BuiltInBackend.unavailability()
    }

    /// Whether a message can be sent right now, and if not, why.
    var cannotSend: String? {
        if sending { return nil }
        switch answerer {
        case .builtIn:
            if let why = builtInUnavailable { return why.reason + " " + why.fix }
            return nil
        case .server:
            if !connected { return "Not connected to \(serverHost): \(status)" }
            if selectedModel.isEmpty { return "No model chosen on \(serverHost)." }
            return nil
        }
    }

    var canSend: Bool { !sending && cannotSend == nil }

    /// The line shown while a server loads a model, or nil.
    var loadingLabel: String? {
        guard sending, activity == .loading else { return nil }
        return "Loading \(answerer.displayName)…"
    }

    // MARK: Conversation management

    func index(of id: UUID?) -> Int? {
        guard let id else { return nil }
        return conversations.firstIndex { $0.id == id }
    }

    func messages(in id: UUID?) -> [Message] {
        index(of: id).map { conversations[$0].messages } ?? []
    }

    @discardableResult
    func newChat() -> UUID {
        let c = Conversation()
        conversations.insert(c, at: 0)
        selectedID = c.id
        save()
        return c.id
    }

    func select(_ id: UUID) {
        guard conversations.contains(where: { $0.id == id }) else { return }
        selectedID = id
        intentSelection = IntentSelection(id: id)
    }

    func deleteChat(_ id: UUID) {
        if sendingIn == id { stop() }
        conversations.removeAll { $0.id == id }
        if selectedID == id { selectedID = conversations.first?.id }
        if conversations.isEmpty {
            let c = Conversation()
            conversations = [c]
            selectedID = c.id
        }
        save()
    }

    // MARK: Persistence

    private var saveURL: URL {
        let base = FileManager.default
            .urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("DessauChat", isDirectory: true)
        return base.appendingPathComponent("conversations.json")
    }

    private func load() {
        guard let data = try? Data(contentsOf: saveURL),
              let saved = try? JSONDecoder().decode([Conversation].self, from: data)
        else { return }
        conversations = saved
    }

    func save() {
        let dir = saveURL.deletingLastPathComponent()
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        if let data = try? JSONEncoder().encode(conversations) {
            try? data.write(to: saveURL, options: .atomic)
        }
    }

    // MARK: The server

    private var base: String {
        var s = serverURL.trimmingCharacters(in: .whitespaces)
        while s.hasSuffix("/") { s.removeLast() }
        return s
    }

    /// The host of the stored address, for the picker and the hints.
    var serverHost: String {
        URL(string: base)?.host ?? base
    }

    /// The stored address's origin — scheme, host and port — which is what the
    /// API key is bound to, so a key entered for https is not sent over http.
    var serverOrigin: String {
        guard let url = URL(string: base), let scheme = url.scheme?.lowercased(), let host = url.host else { return base }
        return "\(scheme)://\(host.lowercased()):\(url.port ?? (scheme == "https" ? 443 : 80))"
    }

    /// The base path, sanitized. serverPath can come from a TXT record, which
    /// is unauthenticated network input: a value like "@example.net" appended
    /// raw would turn "host:11535" into userinfo and hand the bearer token to
    /// whatever host followed. So a path must be a plain, single-rooted path
    /// or it is not used at all.
    private var apiPath: String {
        var p = serverPath.trimmingCharacters(in: .whitespaces)
        while p.hasSuffix("/") { p.removeLast() }
        if p.isEmpty { return AppModel.defaultAPIPath }
        if !p.hasPrefix("/") { p = "/" + p }
        let allowed = CharacterSet(charactersIn:
            "/ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")
        let segments = p.dropFirst().split(separator: "/", omittingEmptySubsequences: false)
        guard p.count <= 128,
              !p.contains("//"),
              p.unicodeScalars.allSatisfy(allowed.contains),
              !segments.contains(where: { $0 == "." || $0 == ".." })
        else { return AppModel.defaultAPIPath }
        return p
    }

    /// Point the client at a hand-typed address. The path resets with it.
    func useTypedAddress(_ address: String) {
        serverURL = address
        serverPath = AppModel.defaultAPIPath
    }

    /// Point the client at a server found on the network.
    func use(_ server: DiscoveredServer, resolvedAddress: String) {
        serverURL = resolvedAddress
        serverPath = server.path.isEmpty ? AppModel.defaultAPIPath : server.path
    }

    /// Save the key for the server the client is pointed at now; it is sent
    /// to that host and no other.
    func saveAPIKey(_ value: String) {
        apiKey = value
        apiKeyHost = value.isEmpty ? "" : serverOrigin
        Keychain.write(value)
    }

    // MARK: Pairing

    /// The pairing this client holds, if the server it is pointed at is the one
    /// it paired with. A pairing is a property of a server, so pointing the
    /// client somewhere else does not make it paired there.
    var paired: PairedServer? {
        guard let p = pairedServer, p.origin == serverOrigin else { return nil }
        return p
    }

    /// Whether the connection to the server in front of the user is a paired
    /// one — which is what the lock beside its name means, and it means nothing
    /// else: it is not about the API key.
    var pairedHere: Bool { paired != nil }

    /// The session every request goes through. One session, not one per
    /// request: a session retains its delegate until it is invalidated, and the
    /// delegate is what answers both TLS challenges.
    let session: URLSession
    let pinning: PinningDelegate

    /// Pair with the server this client is pointed at.
    ///
    /// Three steps, and the third is the one that matters. The key is made
    /// here and its private half never leaves the device; the public half goes
    /// to the server, which signs it a certificate. Then this client opens a
    /// TLS connection of its own and pins the key THAT handshake presented —
    /// never the fingerprint in the pairing answer, which arrived over a plain
    /// port anything on the network can answer on.
    func pair(as name: String) async throws {
        // `makeKey` throws rather than returning nil when the Secure Enclave
        // refuses for a reason that is not this build's signature, so the
        // reason reaches the sheet instead of a quieter key
        // (iss-2609190200098392).
        let key = try PairingStore.makeKey()
        guard let pub = SecKeyCopyPublicKey(key),
              let spki = spkiOf(pub)
        else { throw PairingError.noKey }

        guard let url = URL(string: serverOrigin + "/pair") else { throw PairingError.noKey }
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.timeoutInterval = 30
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = try JSONEncoder().encode(
            PairRequest(name: name, public_key: spki.base64EncodedString()))

        let (data, resp) = try await session.data(for: req)
        guard let http = resp as? HTTPURLResponse else { throw PairingError.badAnswer }
        guard http.statusCode == 200 else {
            throw PairingError.refused(serverMessage(in: data)
                ?? "That server refused the pairing request.")
        }
        guard let answer = try? JSONDecoder().decode(PairAnswer.self, from: data),
              let der = Data(base64Encoded: answer.leaf),
              PairingStore.store(leaf: der)
        else { throw PairingError.badAnswer }

        // The pin comes from a handshake this client made, and from nothing a
        // server or a Bonjour record told it.
        var record = PairedServer(origin: serverOrigin, tlsPort: answer.tls_port,
                                  pinnedSPKI: "", name: answer.name,
                                  clientSPKI: spkiFingerprint(spki),
                                  leafDER: answer.leaf)
        guard let httpsBase = record.httpsBase,
              let probe = URL(string: httpsBase + "/health")
        else {
            PairingStore.removeLeaf(der)
            throw PairingError.noHandshake
        }
        // Learning a pin means having none for the length of one handshake. The
        // pin in force is put back on every way out of here that does not set a
        // new one: leaving the delegate unpinned would make the next connection
        // to an already-paired server accept any certificate at all
        // (iss-2609190100212365).
        let previous = pinning.pinnedSPKI
        pinning.pinnedSPKI = nil
        // And forget whatever handshake was last seen, or a probe that never
        // completes one leaves the fingerprint of the PREVIOUS server standing
        // — which would write a pairing for this server holding that one's key,
        // and show it under "That server's key", which is the one human
        // comparison the whole design rests on (iss-2609190110244227).
        pinning.forgetLastPresented()
        var reached = false
        if let (_, response) = try? await session.data(from: probe) {
            reached = (response as? HTTPURLResponse) != nil
        }
        guard reached, let presented = pinning.lastPresentedSPKI else {
            pinning.pinnedSPKI = previous
            PairingStore.removeLeaf(der)
            throw PairingError.noHandshake
        }
        record.pinnedSPKI = presented
        pinning.pinnedSPKI = presented
        PairingStore.write(record)
        pairedServer = record
    }

    /// Forget a pairing, so this client asks for an API key again.
    func unpair() {
        PairingStore.forget()
        pairedServer = nil
        pinning.pinnedSPKI = nil
    }

    private func serverMessage(in data: Data) -> String? {
        struct Envelope: Decodable {
            struct Inner: Decodable { let message: String? }
            let error: Inner?
        }
        return (try? JSONDecoder().decode(Envelope.self, from: data))?.error?.message
    }

    /// An authenticated request for a path under the API base, or nil when
    /// the stored address is not one a request may be sent to.
    ///
    /// Two ways of proving who this client is, and never both. A PAIRED server
    /// is reached over TLS on its own port, with the client's own key, and the
    /// API key is neither sent nor asked for. Everything else is exactly what
    /// it was: the bearer token is attached only to the host it was entered
    /// for, so a server picked from the network is asked without it and says
    /// so if it needs one.
    ///
    /// A paired server is reached over TLS or not at all. There is deliberately
    /// no fallback to the plain port on a TLS failure: a fallback is an off
    /// switch anything on the network could reach for, and it would put this
    /// client's traffic back in the clear at the moment it most matters.
    func request(_ path: String) -> URLRequest? {
        if let p = paired, let httpsBase = p.httpsBase {
            guard let url = URL(string: httpsBase + apiPath + path) else { return nil }
            var r = URLRequest(url: url)
            r.timeoutInterval = 60
            return r
        }
        guard let url = URL(string: base + apiPath + path),
              let scheme = url.scheme?.lowercased(),
              scheme == "http" || scheme == "https",
              let host = url.host, !host.isEmpty
        else { return nil }
        var r = URLRequest(url: url)
        r.timeoutInterval = 60
        let origin = "\(scheme)://\(host.lowercased()):\(url.port ?? (scheme == "https" ? 443 : 80))"
        if !apiKey.isEmpty, origin == apiKeyHost {
            r.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
        }
        return r
    }

    /// Fetch the model list; doubles as the connection test.
    func connect() async {
        guard var req = request("/models") else {
            status = "That server address is not valid."
            connected = false
            return
        }
        req.httpMethod = "GET"
        connecting = true
        defer { connecting = false }
        status = "Connecting…"
        needsAPIKey = false
        do {
            let (data, resp) = try await session.data(for: req)
            guard let http = resp as? HTTPURLResponse else {
                status = "No response from the server."; connected = false; return
            }
            if http.statusCode == 401 {
                status = "This server needs an API key."
                needsAPIKey = true
                connected = false; return
            }
            guard http.statusCode == 200 else {
                status = "Server returned HTTP \(http.statusCode)."; connected = false; return
            }
            let list = try JSONDecoder().decode(ModelsResponse.self, from: data)
            models = list.data.map(\.id).sorted()
            // The picker offers what this client's own rule counts as able to
            // hold a conversation; the rest stay served over the API by name.
            let rule = chatRule
            chatModels = list.data.filter { rule.offers($0) }.map(\.id).sorted()
            recordedModels = transcriptStates(list.data.map { (id: $0.id, recording: $0.recording) })
            // The chosen model is never moved here: a server that does not
            // serve it is shown as not offering it, and the person picks.
            connected = true
            if models.isEmpty {
                status = "Connected, but no models are downloaded yet."
            } else if chatModels.isEmpty {
                status = "Connected · \(models.count) model\(models.count == 1 ? "" : "s"), none of them for chat"
            } else {
                status = "Connected · \(models.count) model\(models.count == 1 ? "" : "s")"
            }
        } catch {
            connected = false
            status = "Could not reach \(serverHost). Is the server running and on the same network?"
        }
    }

    // MARK: Sending

    func send(_ text: String, in convoID: UUID) {
        let prompt = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !prompt.isEmpty, canSend, let idx = index(of: convoID) else { return }
        conversations[idx].messages.append(Message(role: .user, text: prompt))
        if conversations[idx].title == "New Chat" {
            conversations[idx].title = String(prompt.prefix(48))
        }
        let reply = Message(role: .assistant, model: answerer.recordedName)
        conversations[idx].messages.append(reply)
        save()

        sending = true
        sendingIn = convoID
        activity = .idle
        replyTask = Task { await stream(convoID: convoID, messageID: reply.id) }
        if case .server(let model) = answerer { startResidencyPoll(for: model) }
    }

    func stop() { replyTask?.cancel() }

    /// Sends a prompt as a new chat and returns the finished reply: the App
    /// Intents' way in. Throws, in the client's own words, when nothing can
    /// answer, rather than returning an empty reply as success.
    func ask(prompt: String) async throws -> String {
        while sending { try await Task.sleep(for: .milliseconds(200)) }
        if case .server = answerer, !connected { await connect() }
        if let why = cannotSend { throw BackendMessage(text: why) }
        let id = newChat()
        intentSelection = IntentSelection(id: id)
        send(prompt, in: id)
        await replyTask?.value
        return messages(in: id).last?.text ?? ""
    }

    private func streamSpoke() {
        residencyTask?.cancel()
        residencyTask = nil
    }

    /// Watch the models list while a server request waits, for a server that
    /// sends no loading comments. Only a definite "not in memory" moves the
    /// client into the loading state.
    private func startResidencyPoll(for model: String) {
        residencyTask?.cancel()
        guard !model.isEmpty else { residencyTask = nil; return }
        residencyTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(1))
                guard !Task.isCancelled, let self, self.sending else { return }
                guard let state = await self.residency(of: model) else { continue }
                guard !Task.isCancelled, self.sending else { return }
                self.activity = state == residencyLoaded ? .idle : .loading
            }
        }
    }

    private func residency(of model: String) async -> String? {
        guard var req = request("/models") else { return nil }
        req.httpMethod = "GET"
        req.timeoutInterval = 10
        guard let (data, resp) = try? await session.data(for: req),
              (resp as? HTTPURLResponse)?.statusCode == 200,
              let list = try? JSONDecoder().decode(ModelsResponse.self, from: data)
        else { return nil }
        return list.data.first { $0.id.caseInsensitiveCompare(model) == .orderedSame }?.state
    }

    private func stream(convoID: UUID, messageID: UUID) async {
        defer {
            sending = false
            sendingIn = nil
            activity = .idle
            streamSpoke()
            queueEffects(for: messageID, in: convoID)
            save()
        }
        func edit(_ f: (inout Message) -> Void) {
            guard let ci = index(of: convoID),
                  let mi = conversations[ci].messages.firstIndex(where: { $0.id == messageID }) else { return }
            f(&conversations[ci].messages[mi])
        }
        func current() -> Message? {
            guard let ci = index(of: convoID) else { return nil }
            return conversations[ci].messages.first { $0.id == messageID }
        }

        let history = messages(in: convoID).filter { $0.id != messageID }
        let backend: ChatBackend
        switch answerer {
        case .builtIn: backend = BuiltInBackend()
        case .server(let model): backend = ServerBackend(model: model, request: { [weak self] in self?.request($0) }, session: session)
        }

        do {
            try await backend.reply(to: history) { [weak self] event in
                guard let self else { return }
                switch event {
                case .loading:
                    self.streamSpoke()
                    if self.activity != .generating { self.activity = .loading }
                case .text(let s, let replaces):
                    self.streamSpoke()
                    self.activity = .generating
                    edit { if replaces { $0.text = s } else { $0.text += s } }
                case .reasoning(let r):
                    self.streamSpoke()
                    self.activity = .generating
                    edit { $0.reasoning += r }
                }
            }
            // A thinking model can exhaust its budget before answering; show
            // the reasoning rather than nothing.
            if let m = current(), m.text.isEmpty, !m.reasoning.isEmpty {
                edit { $0.text = $0.reasoning; $0.reasoning = "" }
            }
        } catch is CancellationError {
            if current()?.text.isEmpty == true { edit { $0.text = "⏹ Stopped." } }
        } catch {
            edit { $0.text += (($0.text.isEmpty ? "" : "\n") + "⚠️ " + error.localizedDescription) }
        }
    }

    // MARK: Effects

    private func queueEffects(for messageID: UUID, in convoID: UUID) {
        guard effectsEnabled,
              let text = messages(in: convoID).first(where: { $0.id == messageID })?.text,
              !EffectWords.matches(in: text).isEmpty else { return }
        effectsToPlay.insert(messageID)
    }

    /// A row is about to draw the effect; it plays once, so the id goes.
    func effectStarted(_ messageID: UUID) {
        effectsToPlay.remove(messageID)
    }
}

// MARK: - App

/// The five text sizes Settings offers: the system's own Dynamic Type sizes,
/// so every font stays the system's at a different scale, and the whole
/// window follows one setting read at each scene's root.
enum TextSize: String, CaseIterable {
    case smaller, standard, larger, extraLarge, huge

    var label: String {
        switch self {
        case .smaller: return "Smaller"
        case .standard: return "Default"
        case .larger: return "Larger"
        case .extraLarge: return "Extra Large"
        case .huge: return "Huge"
        }
    }

    /// The Dynamic Type size this step pins, where the default step pins
    /// nothing at all: like Appearance's System case it is the absence of a
    /// preference, so a Mac whose own text size is not large keeps it rather
    /// than being overridden by this client.
    var dynamicType: DynamicTypeSize? {
        switch self {
        case .smaller: return .small
        case .standard: return nil
        case .larger: return .xLarge
        case .extraLarge: return .xxLarge
        case .huge: return .xxxLarge
        }
    }
}

extension View {
    /// The chosen Dynamic Type size, or the view left untouched when no size
    /// is chosen. SwiftUI's own `dynamicTypeSize(_:)` takes no optional the
    /// way `preferredColorScheme(_:)` does, so "no preference" has to be the
    /// modifier not being applied; the label keeps the two apart.
    @ViewBuilder func dynamicTypeSize(ifSet size: DynamicTypeSize?) -> some View {
        if let size {
            dynamicTypeSize(size)
        } else {
            self
        }
    }
}

/// Light, Dark or System: the preferred colour scheme at each scene's root,
/// where System is no preference at all — the Mac's own appearance, as
/// before.
enum Appearance: String, CaseIterable {
    case system, light, dark

    var label: String {
        switch self {
        case .system: return "System"
        case .light: return "Light"
        case .dark: return "Dark"
        }
    }

    var colorScheme: ColorScheme? {
        switch self {
        case .system: return nil
        case .light: return .light
        case .dark: return .dark
        }
    }
}

/// The chat scene's id, so a command can open another window of it.
let chatWindowID = "chat"

@main
struct DessauChatApp: App {
    @StateObject private var model = AppModel.shared
    @FocusedValue(\.chatActions) private var actions
    #if os(macOS)
    // Replacing the .newItem group takes the system's own New Window item
    // with it; the client opens the second window itself.
    @Environment(\.openWindow) private var openWindow
    #endif
    @AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue
    @AppStorage("appearance") private var appearance: String = Appearance.system.rawValue
    #if !os(macOS)
    /// The iPad has the one window, and its Settings sheet is shown from two
    /// places — the toolbar's gear and the menu bar's command — so the state
    /// that says whether it is up lives here, above both.
    @State private var settingsShown = false
    #endif

    private var dynamicType: DynamicTypeSize? {
        TextSize(rawValue: textSize)?.dynamicType
    }

    private var colorScheme: ColorScheme? {
        Appearance(rawValue: appearance)?.colorScheme
    }

    var body: some Scene {
        WindowGroup("Dessau Chat", id: chatWindowID) {
            // A minimum window size is a Mac's business; on the iPad the app
            // is given the screen (or a Split View share of it) and fits it.
            // The chosen text size and appearance are the person's on both.
            #if os(macOS)
            RootView(model: model)
                .frame(minWidth: 720, minHeight: 480)
                .dynamicTypeSize(ifSet: dynamicType)
                .preferredColorScheme(colorScheme)
            #else
            RootView(model: model, settingsShown: $settingsShown)
                .dynamicTypeSize(ifSet: dynamicType)
                .preferredColorScheme(colorScheme)
            #endif
        }
        .commands {
            CommandGroup(replacing: .newItem) {
                Button("New Chat") { actions?.newChat() }
                    .keyboardShortcut("n", modifiers: .command)
                    .disabled(actions == nil)
                #if os(macOS)
                // The Mac's second window onto the same chats, on the shortcut
                // the system gives New Window everywhere else. The iPad has
                // the one window and no such item to restore.
                Button("New Window") { openWindow(id: chatWindowID) }
                    .keyboardShortcut("n", modifiers: [.command, .shift])
                #endif
            }
            CommandMenu("Chat") {
                Button("Send") { actions?.send() }
                    .keyboardShortcut(.return, modifiers: .command)
                    .disabled(actions?.canSend != true)
                Button("Stop") { model.stop() }
                    .keyboardShortcut(".", modifiers: .command)
                    .disabled(!model.sending)
                Divider()
                Button("Choose Model…") { actions?.chooseModel() }
                    .keyboardShortcut("m", modifiers: [.command, .shift])
                    .disabled(actions == nil)
                Button("Reconnect to Server") { Task { await model.connect() } }
                    .keyboardShortcut("r", modifiers: .command)
                Divider()
                Button("Delete Chat") { actions?.deleteChat() }
                    .keyboardShortcut(.delete, modifiers: .command)
                    .disabled(actions == nil)
                #if !os(macOS)
                // On macOS this is the system's own Cmd-, onto the Settings
                // scene; iPadOS has neither, so the command is ours and opens
                // the same sheet the toolbar's gear does.
                Divider()
                Button("Settings…") { settingsShown = true }
                    .keyboardShortcut(",", modifiers: .command)
                #endif
            }
        }
        // iPadOS has no Settings scene and no Cmd-, to open one: there the
        // same view is a sheet from the toolbar's gear (see ChatDetail).
        #if os(macOS)
        Settings {
            SettingsView(model: model)
                .dynamicTypeSize(ifSet: dynamicType)
                .preferredColorScheme(colorScheme)
        }
        #endif
    }
}

/// What the menu bar can do to the front window's chat.
struct ChatActions {
    var canSend: Bool
    var newChat: () -> Void
    var send: () -> Void
    var chooseModel: () -> Void
    var deleteChat: () -> Void
}

struct ChatActionsKey: FocusedValueKey {
    typealias Value = ChatActions
}

extension FocusedValues {
    var chatActions: ChatActions? {
        get { self[ChatActionsKey.self] }
        set { self[ChatActionsKey.self] = newValue }
    }
}

// MARK: - Views

struct RootView: View {
    @ObservedObject var model: AppModel
    #if !os(macOS)
    /// Whether the Settings sheet is up; the app owns it (see above).
    @Binding var settingsShown: Bool
    #endif
    /// The window's own chat, kept with the window so it comes back after a
    /// relaunch.
    @SceneStorage("selectedConversation") private var stored: String = ""
    @State private var selection: UUID?
    @State private var draft = ""
    @State private var pickerShown = false
    #if os(macOS)
    @Environment(\.controlActiveState) private var activeState
    #endif

    var body: some View {
        NavigationSplitView {
            Sidebar(model: model, selection: $selection)
                .navigationSplitViewColumnWidth(min: 200, ideal: 240, max: 340)
        } detail: {
            #if os(macOS)
            ChatDetail(model: model, conversationID: selection, draft: $draft, pickerShown: $pickerShown)
            #else
            ChatDetail(model: model, conversationID: selection, draft: $draft,
                       pickerShown: $pickerShown, settingsShown: $settingsShown)
            #endif
        }
        .focusedSceneValue(\.chatActions, ChatActions(
            canSend: model.canSend && !draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
            newChat: { selection = model.newChat() },
            send: { if let id = selection { model.send(draft, in: id); draft = "" } },
            chooseModel: { pickerShown = true },
            deleteChat: { if let id = selection { model.deleteChat(id); selection = model.conversations.first?.id } }))
        .onAppear {
            // An intent that launched the app has already said which chat to
            // show; otherwise the window's own last chat comes back.
            if let asked = model.intentSelection, Date().timeIntervalSince(asked.at) < 10 {
                selection = asked.id
                model.intentSelection = nil
            } else {
                selection = UUID(uuidString: stored).flatMap { id in
                    model.conversations.contains { $0.id == id } ? id : nil
                } ?? model.selectedID ?? model.conversations.first?.id
            }
            model.refreshBuiltInAvailability()
        }
        .onChange(of: selection) { _, new in
            stored = new?.uuidString ?? ""
            if let new { model.selectedID = new }
        }
        .onChange(of: model.intentSelection) { _, asked in
            // An App Intent chose a chat; the key window follows, the others
            // keep what they show. The iPad has the one window, which is
            // always the one being looked at, so it follows unconditionally.
            guard let asked else { return }
            #if os(macOS)
            guard activeState == .key else { return }
            #endif
            selection = asked.id
            model.intentSelection = nil
        }
    }
}

struct Sidebar: View {
    @ObservedObject var model: AppModel
    @Binding var selection: UUID?
    @State private var query = ""

    /// The conversations the search leaves: all of them for an empty
    /// query, else those whose title or messages contain the words — each
    /// word looked for on its own, wherever it falls. `SidebarSearch` is the
    /// match itself, in a file `client/tests/sidebar-search.sh` can run.
    private var shown: [Conversation] {
        let words = SidebarSearch.words(in: query)
        guard !words.isEmpty else { return model.conversations }
        return model.conversations.filter { c in
            SidebarSearch.matches(words: words, title: c.title, messages: c.messages.map(\.text))
        }
    }

    var body: some View {
        List(selection: $selection) {
            ForEach(shown) { c in
                ConversationCard(conversation: c)
                    .tag(c.id)
                    .contextMenu {
                        Button("Delete", role: .destructive) {
                            model.deleteChat(c.id)
                            if selection == c.id { selection = model.conversations.first?.id }
                        }
                    }
            }
            .onDelete { offsets in
                let shownNow = shown
                offsets.map { shownNow[$0].id }.forEach(model.deleteChat)
                if let s = selection, !model.conversations.contains(where: { $0.id == s }) {
                    selection = model.conversations.first?.id
                }
            }
        }
        .listStyle(.sidebar)
        .searchable(text: $query, placement: .sidebar, prompt: "Search")
        .navigationTitle("Chats")
        // One thin toolbar, as Messages has: left automatic, the title is
        // drawn large and collapses as the list scrolls, which gives the
        // header two heights.
        .toolbarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem {
                Button { selection = model.newChat() } label: { Image(systemName: "square.and.pencil") }
                    .help("New chat")
            }
        }
    }
}

/// One conversation in the sidebar, the way Messages shows one: an icon for
/// who answered last, the title, the date it started, and a summary of
/// exchanges and words. The row's selection colour is the sidebar's own;
/// nothing is drawn behind it.
struct ConversationCard: View {
    let conversation: Conversation

    private var title: String {
        conversation.title.isEmpty ? "New Chat" : conversation.title
    }

    /// The symbol for who answered last: the Mac's own model, a server, or
    /// nobody yet.
    private var icon: String {
        guard let last = conversation.messages.last(where: { $0.role == .assistant })?.model, !last.isEmpty else {
            return "bubble.left"
        }
        return last == BuiltInBackend.recordedName ? "apple.intelligence" : "network"
    }

    private var summary: String {
        let exchanges = exchangeCount(fromPerson: conversation.messages.map { $0.role == .user })
        let words = conversation.messages.reduce(0) { $0 + $1.text.split(whereSeparator: \.isWhitespace).count }
        return "\(exchanges) exchange\(exchanges == 1 ? "" : "s") · \(words) word\(words == 1 ? "" : "s")"
    }

    var body: some View {
        Label {
            VStack(alignment: .leading, spacing: 3) {
                HStack(alignment: .firstTextBaseline) {
                    Text(title).font(.headline).lineLimit(1)
                    Spacer(minLength: 8)
                    Text(conversation.createdAt.formatted(date: .numeric, time: .omitted))
                        .font(.caption).foregroundStyle(.secondary)
                }
                Text(summary).font(.caption).foregroundStyle(.secondary).lineLimit(1)
            }
        } icon: {
            Image(systemName: icon).font(.title2)
        }
        .padding(.vertical, 6)
    }
}

struct ChatDetail: View {
    @ObservedObject var model: AppModel
    let conversationID: UUID?
    @Binding var draft: String
    @Binding var pickerShown: Bool
    #if !os(macOS)
    /// iPadOS has no Settings scene: the gear in the toolbar and the menu
    /// bar's command show the same view as a sheet, which is where an iPad
    /// app keeps its settings.
    @Binding var settingsShown: Bool
    #endif

    private var messages: [Message] { model.messages(in: conversationID) }

    var body: some View {
        VStack(spacing: 0) {
            transcript
            Divider()
            composer
        }
        .navigationTitle("Dessau Chat")
        // The header is one fixed height whatever the state
        // (iss-2609190004097595): the title never switches between a large
        // form and a collapsed one, so the transcript never moves under it.
        .toolbarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem {
                Button {
                    pickerShown = true
                } label: {
                    Label {
                        Text(model.answerer.displayName)
                    } icon: {
                        // One size across both states: a spinner and a symbol
                        // do not measure the same, and a toolbar item that
                        // changes height takes the header's height with it.
                        Group {
                            if model.connecting {
                                ProgressView().controlSize(.small)
                            } else {
                                Image(systemName: model.answerer == .builtIn ? "apple.intelligence" : "network")
                            }
                        }
                        .frame(width: 16, height: 16)
                    }
                    .labelStyle(.titleAndIcon)
                }
                .help("Choose who answers: this \(BuiltInBackend.deviceNoun), or a server on your network")
                .popover(isPresented: $pickerShown) {
                    ModelPickerView(model: model)
                }
            }
            #if !os(macOS)
            ToolbarItem(placement: .topBarLeading) {
                Button { settingsShown = true } label: { Image(systemName: "gearshape") }
                    .help("Settings")
            }
            #endif
        }
        #if !os(macOS)
        .sheet(isPresented: $settingsShown) {
            NavigationStack {
                SettingsView(model: model)
                    .navigationTitle("Settings")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { settingsShown = false }
                        }
                    }
            }
        }
        #endif
    }

    private var transcript: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 14) {
                    if messages.isEmpty {
                        EmptyState(model: model, choose: { pickerShown = true })
                    }
                    ForEach(messages) { m in
                        MessageRow(model: model, message: m,
                                   loadingLabel: m.id == messages.last?.id && model.sendingIn == conversationID
                                       ? model.loadingLabel : nil)
                            .id(m.id)
                    }
                }
                // Room on both sides: a bubble never touches the window's edge.
                .padding(.horizontal, 28)
                .padding(.vertical, 12)
            }
            // A conversation opens at its newest message — and only that.
            // The roleless form of this modifier anchors the ALIGNMENT role
            // as well, which pins a conversation shorter than the window to
            // the window's foot and opens an empty band under the toolbar.
            // The roles do not compose across two of these modifiers — a
            // second one replaces the first — so this is the one; the scroll
            // to the newest message while a reply streams is explicit below.
            .defaultScrollAnchor(.bottom, for: .initialOffset)
            // The scroll-edge effect belongs to the toolbar, not to the
            // transcript: the hard style ends at the bar, where the automatic
            // one is a soft blur that reaches well down the window and washes
            // out the reply being read.
            .scrollEdgeEffectStyle(.hard, for: .top)
            // A link a model wrote opens only as a web address: a served reply
            // is not trusted to hand the Mac a file, shortcut or settings URL.
            .environment(\.openURL, OpenURLAction { url in
                let scheme = url.scheme?.lowercased()
                return scheme == "http" || scheme == "https" ? .systemAction : .discarded
            })
            .onChange(of: messages.last?.text) { _, _ in
                if let last = messages.last { proxy.scrollTo(last.id, anchor: .bottom) }
            }
            .onChange(of: conversationID) { _, _ in
                if let last = messages.last { proxy.scrollTo(last.id, anchor: .bottom) }
            }
        }
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 6) {
            // The form of Messages' composer: a capsule field and a round,
            // filled send button. The button is a standard control given the
            // system's circle; the capsule is drawn in Composer.swift,
            // because SwiftUI's bordered capsule leaves its text against the
            // curve and offers no way to inset it (iss-2609190004092322).
            HStack(alignment: .bottom, spacing: 8) {
                TextField("Message…", text: $draft, axis: .vertical)
                    .lineLimit(1...8)
                    .composerFieldCapsule()
                    .onSubmit(send)
                    .disabled(!model.canSend)
                    .accessibilityLabel("Message")
                if model.sending {
                    Button { model.stop() } label: { Image(systemName: "stop.fill") }
                        .buttonStyle(.borderedProminent)
                        .buttonBorderShape(.circle)
                        .controlSize(.large)
                        .composerButtonCircle()
                        .help("Stop")
                } else {
                    Button(action: send) { Image(systemName: "arrow.up") }
                        .buttonStyle(.borderedProminent)
                        .buttonBorderShape(.circle)
                        .controlSize(.large)
                        .composerButtonCircle()
                        .keyboardShortcut(.return, modifiers: .command)
                        .disabled(!model.canSend || draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        .help("Send")
                }
            }
            // A disabled control explains nothing by itself; say why.
            if let why = model.cannotSend {
                Label(why, systemImage: "exclamationmark.circle")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding()
        // A text file dropped on the composer becomes part of the prompt.
        .dropDestination(for: URL.self) { urls, _ in
            // A regular text file of ordinary size; a prompt is not a place
            // for a gigabyte, and a symlink to one is followed by the read.
            let cap = 1 << 20
            for url in urls {
                guard let values = try? url.resourceValues(forKeys: [.contentTypeKey, .isRegularFileKey, .fileSizeKey]),
                      values.isRegularFile == true,
                      let type = values.contentType, type.conforms(to: .text),
                      let size = values.fileSize, size <= cap,
                      let text = try? String(contentsOf: url, encoding: .utf8) else { continue }
                draft += (draft.isEmpty ? "" : "\n\n") + text
            }
        }
    }

    private func send() {
        guard let id = conversationID else { return }
        model.send(draft, in: id)
        draft = ""
    }
}

/// The empty chat: what will answer, or why nothing can yet.
struct EmptyState: View {
    @ObservedObject var model: AppModel
    var choose: () -> Void

    /// The app's own icon. macOS hands over the icon the Dock shows. iPadOS
    /// has no such call and does not answer to the asset's name either: the
    /// compiled icon is listed in the bundle's own CFBundleIcons, under the
    /// file names actool wrote, and that is what is loaded here. The system's
    /// chat symbol stands in for a bundle whose icon step did not run.
    private var icon: Image {
        #if os(macOS)
        return Image(nsImage: NSApp.applicationIconImage)
        #else
        let icons = Bundle.main.object(forInfoDictionaryKey: "CFBundleIcons") as? [String: Any]
        let primary = icons?["CFBundlePrimaryIcon"] as? [String: Any]
        let files = primary?["CFBundleIconFiles"] as? [String] ?? []
        // The last is the largest actool wrote.
        for name in files.reversed() {
            if let app = UIImage(named: name) { return Image(uiImage: app) }
        }
        return Image(systemName: "bubble.left.and.bubble.right")
        #endif
    }

    var body: some View {
        VStack(spacing: 12) {
            icon
                .resizable()
                .frame(width: 64, height: 64)
            if let why = model.cannotSend {
                Text(why)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 440)
                Button("Choose Who Answers…", action: choose)
            } else {
                Text("Ask \(model.answerer.displayName) anything.")
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 60)
    }
}

/// One message. The person's are plain and right-aligned; a reply is a
/// grouped box under the speaker's name, its markdown rendered block by block.
struct MessageRow: View {
    @ObservedObject var model: AppModel
    let message: Message
    var loadingLabel: String? = nil
    @State private var showReasoning = false
    private let colors = BubbleColors()
    /// The rendered blocks of what this row draws — the reply's and the
    /// thoughts' — keyed by the text each was parsed from, parsed once per
    /// change of that text and, while a reply is streaming, at most a few
    /// times a second. One store, so one scheduler and one throttle serve
    /// both: a pane whose text has not changed keeps its blocks without
    /// being parsed again, and a text this row no longer shows is dropped.
    @State private var parsed: [String: [MarkdownBlock]] = [:]
    @State private var lastParse = Date.distantPast
    @State private var parseTask: Task<Void, Never>?
    /// Whether this row is playing the effect, and whether it already has.
    /// The reply animates at the moment it FINISHES: the model queues the id
    /// in its stream's defer, and the row watches for that change. A row
    /// recreated on scroll sees no change — the id was already there when it
    /// appeared — so it draws plain text, and a re-evaluation cannot switch
    /// the branch mid-play.
    @State private var playing = false
    @State private var played = false

    private var isUser: Bool { message.role == .user }
    private var displayText: String { message.text.trimmingCharacters(in: .whitespacesAndNewlines) }
    private var displayReasoning: String { message.reasoning.trimmingCharacters(in: .whitespacesAndNewlines) }
    private var blocks: [MarkdownBlock] { parsed[displayText] ?? [] }
    private var reasoningBlocks: [MarkdownBlock] { parsed[displayReasoning] ?? [] }

    /// Who spoke, for the label and for VoiceOver.
    private var speaker: String {
        if isUser { return "You" }
        guard let id = message.model, !id.isEmpty else { return "The model" }
        return id.split(separator: "/").last.map(String.init) ?? id
    }

    var body: some View {
        if isUser {
            HStack {
                Spacer(minLength: 56)
                Text(displayText)
                    .textSelection(.enabled)
                    .multilineTextAlignment(.leading)
                    .bubble(colors.userColor, isUser: true)
            }
            .accessibilityLabel("\(speaker) said: \(displayText)")
        } else {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(speaker).font(.caption).foregroundStyle(.secondary).padding(.leading, 12)
                    VStack(alignment: .leading, spacing: 8) {
                        if !displayReasoning.isEmpty { reasoningDisclosure }
                        if displayText.isEmpty && displayReasoning.isEmpty {
                            waiting
                        } else if !displayText.isEmpty {
                            reply
                        }
                    }
                    .bubble(colors.modelColor, isUser: false)
                    .contextMenu {
                        Button("Copy") {
                            // The two systems' pasteboards, which differ in
                            // name and in whether the old contents are
                            // cleared first.
                            #if os(macOS)
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(message.text, forType: .string)
                            #else
                            // localOnly: a copied reply is for this iPad.
                            // Without it the general pasteboard is a Universal
                            // Clipboard one, and what a model wrote here
                            // appears on the person's other devices.
                            UIPasteboard.general.setItems(
                                [[UTType.utf8PlainText.identifier: message.text]],
                                options: [.localOnly: true])
                            #endif
                        }
                    }
                }
                Spacer(minLength: 56)
            }
            .accessibilityLabel("\(speaker) said: \(displayText)")
        }
    }

    /// The reply, rendered: one selectable Text per block, the matched words
    /// animated once if the reply earned it.
    @ViewBuilder private var reply: some View {
        blockViews(blocks, animate: playing)
        .onAppear {
            // A reply whose id is queued before this row exists finished while
            // the row was off screen: the effect's moment has passed, so the
            // id goes without playing and no later row can fire on it.
            if model.effectsToPlay.contains(message.id) { model.effectStarted(message.id) }
            if blocks.isEmpty { parseNow() }
        }
        // The reply finishing is the queueing of its id; that change, and only
        // that change, plays the effect, once.
        .onChange(of: model.effectsToPlay.contains(message.id)) { _, due in
            guard due, !played else { return }
            played = true
            playing = true
            model.effectStarted(message.id)
        }
        .onChange(of: displayText) { _, _ in scheduleParse() }
    }

    /// The blocks a reply or a thought is made of, one selectable Text each.
    @ViewBuilder private func blockViews(_ blocks: [MarkdownBlock], animate: Bool) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(Array(blocks.enumerated()), id: \.offset) { _, block in
                switch block {
                case .paragraph(let s):
                    styled(s, animate: animate)
                case .heading(let level, let s):
                    styled(s, animate: animate)
                        .font(level <= 1 ? .title2 : level == 2 ? .title3 : .headline)
                case .listItem(let ordinal, let s):
                    HStack(alignment: .firstTextBaseline, spacing: 6) {
                        Text(ordinal.map { "\($0)." } ?? "•").foregroundStyle(.secondary)
                        styled(s, animate: animate)
                    }
                    .padding(.leading, 8)
                case .quote(let s):
                    HStack(alignment: .top, spacing: 8) {
                        Divider()
                        styled(s, animate: animate).foregroundStyle(.secondary)
                    }
                case .code(let code):
                    GroupBox {
                        Text(code)
                            .font(.body.monospaced())
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                case .plain(let text):
                    Text(text)
                        .font(.body.monospaced())
                        .textSelection(.enabled)
                }
            }
        }
    }

    /// Parse now if the last parse is older than a quarter of a second, else
    /// once at that deadline — four parses a second at most, for the reply
    /// and the thoughts together; a finished reply's text never changes
    /// again. The Thoughts row is on this scheduler too: one pending parse
    /// for the row, so neither pane can be left behind by the other.
    private func scheduleParse() {
        let wait = 0.25 - Date().timeIntervalSince(lastParse)
        parseTask?.cancel()
        if wait <= 0 {
            parseNow()
            return
        }
        parseTask = Task {
            try? await Task.sleep(for: .seconds(wait))
            guard !Task.isCancelled else { return }
            parseNow()
        }
    }

    /// The row's one parse pass: whichever of the two texts is shown and has
    /// changed is parsed, the other keeps the blocks it already has.
    private func parseNow() {
        var next: [String: [MarkdownBlock]] = [:]
        if !displayText.isEmpty {
            next[displayText] = parsed[displayText] ?? MarkdownBlocks.parse(displayText)
        }
        if !displayReasoning.isEmpty {
            next[displayReasoning] = parsed[displayReasoning] ?? MarkdownBlocks.parse(displayReasoning)
        }
        parsed = next
        lastParse = Date()
    }

    @ViewBuilder private func styled(_ s: AttributedString, animate: Bool) -> some View {
        if animate {
            EffectText(text: s, matches: EffectWords.matches(in: String(s.characters)))
        } else {
            Text(s).textSelection(.enabled)
        }
    }

    @ViewBuilder private var waiting: some View {
        if let loadingLabel {
            Label { Text(loadingLabel).foregroundStyle(.secondary) } icon: { ProgressView().controlSize(.small) }
        } else {
            ProgressView().controlSize(.small)
        }
    }

    /// A click anywhere in the row, and anywhere in the expanded thinking,
    /// toggles it — not only the disclosure triangle — so the thinking can be
    /// hidden while it is being read. Selection is per block: each block the
    /// thinking is drawn as carries its own textSelection, and the row carries
    /// none, so a drag selects within one block and does not run across two.
    @ViewBuilder private var reasoningDisclosure: some View {
        DisclosureGroup(isExpanded: $showReasoning) {
            blockViews(reasoningBlocks, animate: false)
                .font(.callout).italic()
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
                .onTapGesture { withAnimation { showReasoning = false } }
                .onAppear {
                    if reasoningBlocks.isEmpty { parseNow() }
                }
                .onChange(of: displayReasoning) { _, _ in scheduleParse() }
        } label: {
            Label {
                Text(displayText.isEmpty ? "Thinking…" : "Thoughts").font(.caption)
            } icon: {
                if displayText.isEmpty { ProgressView().controlSize(.mini) }
            }
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .contentShape(Rectangle())
            .onTapGesture { withAnimation { showReasoning.toggle() } }
        }
    }
}

/// Settings: the server the client is pointed at, its key, which served
/// models the picker offers, and the reply effects.
struct SettingsView: View {
    @ObservedObject var model: AppModel
    @State private var key = ""
    @State private var pairingName = ""
    @State private var pairing = false
    @State private var pairingFailure: String?
    @AppStorage("bubbleColorUser") private var bubbleUser: String = ""
    @AppStorage("bubbleColorModel") private var bubbleModel: String = ""
    @AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue
    @AppStorage("appearance") private var appearance: String = Appearance.system.rawValue

    private var typedAddress: Binding<String> {
        Binding(get: { model.serverURL }, set: { model.useTypedAddress($0) })
    }

    /// Pair with the server the client is pointed at, and say plainly when it
    /// does not work rather than leaving a button that did nothing.
    private func pair() async {
        pairing = true
        pairingFailure = nil
        defer { pairing = false }
        do {
            try await model.pair(as: pairingName.trimmingCharacters(in: .whitespaces))
            pairingName = ""
        } catch {
            pairingFailure = error.localizedDescription
        }
    }

    var body: some View {
        Form {
            Section("Server") {
                TextField("Address", text: typedAddress, prompt: Text("http://alices-mac.local:11535"))
                Text("The server's address: its .local name or LAN address, port 11535 for Dessau Server. Servers on your network are offered in the model picker without typing anything.")
                    .font(.caption).foregroundStyle(.secondary)
                SecureField("API key", text: $key, prompt: Text("Only if that server requires one"))
                    // Saved only when the person changed it: the initial load
                    // below must not re-bind the stored key to whatever server
                    // the client happens to be pointed at now.
                    .onChange(of: key) { _, new in
                        guard new != model.apiKey else { return }
                        model.saveAPIKey(new)
                    }
                Text(model.apiKeyHost.isEmpty
                     ? "The key is sent only to the server it is entered for."
                     : "The key is sent only to \(model.apiKeyHost).")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Pairing") {
                if let paired = model.paired {
                    LabeledContent("Paired as", value: paired.name)
                    // Shown so it can be compared with the fingerprint on the
                    // server's own Clients page. That comparison is the only
                    // check there is on a pairing nobody approved: anything on
                    // the network can answer a pairing request, and a client
                    // that pinned the wrong answer looks exactly like one that
                    // pinned the right one.
                    LabeledContent("That server's key", value: paired.pinnedSPKI)
                        .textSelection(.enabled)
                    LabeledContent("This client's key", value: paired.clientSPKI)
                        .textSelection(.enabled)
                    Text("Compare the server's key with the one shown on its control panel. If they differ, this client is talking to something else — forget the pairing and look at your network.")
                        .font(.caption).foregroundStyle(.secondary)
                    Button("Forget this pairing") { model.unpair() }
                } else {
                    TextField("Name for this \(BuiltInBackend.deviceNoun)", text: $pairingName,
                              prompt: Text("Bob's \(BuiltInBackend.deviceNoun)"))
                    Button("Pair with this server") {
                        Task { await pair() }
                    }
                    .disabled(pairingName.trimmingCharacters(in: .whitespaces).isEmpty || pairing)
                    if let failure = pairingFailure {
                        Text(failure).font(.caption).foregroundStyle(.secondary)
                    }
                    Text("Pairing gives this \(BuiltInBackend.deviceNoun) a key of its own on that server. Once paired it never asks for an API key, and what passes between them is no longer readable by everything on your network. Anything on the network can pair with a Dessau Server, so whoever runs it should check the list on its control panel.")
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
            Section("Models to offer") {
                TextField("Pipeline tags", text: $model.chatPipelineTags, prompt: Text("text-generation, image-text-to-text"))
                TextField("Required tags", text: $model.chatRequiredTags, prompt: Text("conversational"))
                Text("The picker offers a server's models carrying these HuggingFace words — a pipeline tag from the first list, and every tag in the second. Every model stays reachable over the API by name. Clear a field to stop testing it. The \(BuiltInBackend.deviceNoun)'s own model is always offered.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Appearance") {
                Picker("Appearance", selection: $appearance) {
                    ForEach(Appearance.allCases, id: \.rawValue) { a in
                        Text(a.label).tag(a.rawValue)
                    }
                }
                .pickerStyle(.segmented)
            }
            Section("Text") {
                Picker("Text size", selection: $textSize) {
                    ForEach(TextSize.allCases, id: \.rawValue) { size in
                        Text(size.label).tag(size.rawValue)
                    }
                }
                Text("The whole window follows, at the system's own text sizes. Default is whatever this \(BuiltInBackend.deviceNoun) is already set to.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Bubbles") {
                BubbleColorRow(title: "Your messages", stored: $bubbleUser, fallback: BubbleColors.defaultUser)
                BubbleColorRow(title: "The model's replies", stored: $bubbleModel, fallback: BubbleColors.defaultModel)
                Text("The defaults are the system's accent colour and grey.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Replies") {
                Toggle("Animate certain words", isOn: $model.effectsEnabled)
                Text(EffectWords.settingsSentence)
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        // A Settings window is sized by the app on macOS; the iPad's sheet is
        // sized by the system.
        #if os(macOS)
        .frame(width: 520)
        #endif
        .onAppear { key = model.apiKey }
    }
}

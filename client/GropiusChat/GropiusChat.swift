// GropiusChat — a native macOS chat app.
//
// It chats with the Mac's own model out of the box and, when a Gropius server
// is on the network, offers that server's models in the picker: GET /v1/models
// to list them, POST /v1/chat/completions (streaming) to chat. Servers are
// found over Bonjour while the picker is open. Conversations are kept in a
// sidebar and persisted to disk.
//
// Built without an Xcode project: several Swift files, one script (build.sh),
// the installed Xcode's toolchain. This file holds the app, the model and the
// views; the answerers, the picker, the markdown, the effects and the App
// Intents each have a file of their own.

import AppKit
import Network
import SwiftUI
import Security
import UniformTypeIdentifiers

// MARK: - Keychain

/// Keychain-backed storage for the one secret this app holds: the server API
/// key. Storing it in UserDefaults (as an earlier build did) leaves it in
/// cleartext in the preferences plist, readable by any process running as the
/// user and by anything that syncs or backs up the home directory. The Keychain
/// gates it behind the login-keychain ACL instead.
enum Keychain {
    private static let service = "dev.gropius.chat"
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
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlocked,
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

// MARK: - Local network discovery

/// The mDNS service type a Gropius server advertises itself on.
///
/// It has to match the server's own (internal/discovery), and it is declared a
/// second time in Info.plist's NSBonjourServices — macOS Local Network Privacy
/// answers a browse for an undeclared type with an empty result set rather than
/// an error. A test in the server's suite holds all three to one value.
let gropiusServiceType = "_gropius._tcp"

/// One Gropius server seen on the local network. Everything here comes out of
/// the browse itself; nothing has been resolved, because resolving costs an
/// mDNS round trip per server and only the one the person picks is worth it.
struct DiscoveredServer: Identifiable, Equatable {
    let name: String
    let type: String
    let domain: String
    /// TXT "api": the dialect the endpoint speaks. This client speaks "openai".
    let api: String
    /// TXT "path": the base path the API is mounted under, e.g. "/v1".
    let path: String
    /// TXT "auth": "bearer" when the server requires an API key, "none" when it
    /// does not, "" when the record did not say.
    let auth: String
    /// TXT "models": how many models the server can serve, nil when unstated.
    let models: Int?

    var id: String { "\(name)|\(type)|\(domain)" }
    var authRequired: Bool { auth == "bearer" }
    var authStated: Bool { auth == "bearer" || auth == "none" }
    var speaksThisClientsAPI: Bool { api.isEmpty || api == "openai" }

    /// The one-line description under the server's name. Every part of it is a
    /// hint from the network, so it says what was advertised and never more.
    var summary: String {
        var parts: [String] = []
        if authStated {
            parts.append(authRequired ? "API key required" : "No API key needed")
        } else {
            parts.append("Does not say whether it needs an API key")
        }
        if let models {
            parts.append("\(models) model\(models == 1 ? "" : "s")")
        }
        if !speaksThisClientsAPI {
            parts.append("speaks the \(api) API, not openai")
        }
        return parts.joined(separator: " · ")
    }

    init?(_ result: NWBrowser.Result) {
        guard case let .service(name, type, domain, _) = result.endpoint else { return nil }
        self.name = name
        self.type = type
        self.domain = domain
        var txt: [String: String] = [:]
        if case let .bonjour(record) = result.metadata {
            for key in ["api", "path", "auth", "models"] {
                txt[key] = record[key]
            }
        }
        api = txt["api"] ?? ""
        path = txt["path"] ?? ""
        auth = txt["auth"] ?? ""
        models = txt["models"].flatMap(Int.init)
    }
}

/// Browses the local network for Gropius servers. It only ever lists what it
/// finds; connecting is the person's decision.
final class ServerBrowser: ObservableObject {
    enum Status: Equatable {
        case stopped
        case searching
        /// The browse cannot run yet — most often local network access has not
        /// been granted. The reason is shown, because the person is the only one
        /// who can clear it.
        case waiting(String)
        case failed(String)
    }

    @Published private(set) var servers: [DiscoveredServer] = []
    @Published private(set) var status: Status = .stopped

    private var browser: NWBrowser?

    func start() {
        guard browser == nil else { return }
        status = .searching
        servers = []

        let browser = NWBrowser(
            for: .bonjourWithTXTRecord(type: gropiusServiceType, domain: nil),
            using: NWParameters())
        browser.stateUpdateHandler = { [weak self] state in
            Task { @MainActor in self?.apply(state) }
        }
        // The handler is called with the complete current result set, not a
        // delta, so replacing the list is also how a server that has left the
        // network stops being offered.
        browser.browseResultsChangedHandler = { [weak self] results, _ in
            Task { @MainActor in self?.apply(results.compactMap(DiscoveredServer.init)) }
        }
        self.browser = browser
        browser.start(queue: .main)
    }

    func stop() {
        browser?.cancel()
        browser = nil
        servers = []
        status = .stopped
    }

    private func apply(_ state: NWBrowser.State) {
        switch state {
        case .ready, .setup:
            status = .searching
        case .waiting(let error):
            status = .waiting(error.localizedDescription)
        case .failed(let error):
            browser?.cancel()
            browser = nil
            status = .failed(error.localizedDescription)
        case .cancelled:
            status = .stopped
        @unknown default:
            status = .searching
        }
    }

    private func apply(_ found: [DiscoveredServer]) {
        // One server seen on two interfaces arrives as two results; they carry
        // the same instance name, so collapse them.
        var seen = Set<String>()
        servers = found
            .filter { seen.insert($0.id).inserted }
            .sorted { $0.name.localizedStandardCompare($1.name) == .orderedAscending }
    }
}

/// Turns a browsed service into an address that can be stored: browse with
/// NWBrowser, resolve with NetService, which yields the host name the server
/// publishes its address records under and keeps resolving after the address
/// changes.
final class ServiceResolver: NSObject, NetServiceDelegate {
    enum Outcome {
        case address(String)
        case failure(String)
    }

    private let service: NetService
    private var completion: ((Outcome) -> Void)?
    /// Held until an outcome is delivered.
    private var keepAlive: ServiceResolver?

    init(server: DiscoveredServer) {
        service = NetService(domain: Self.qualified(server.domain),
                             type: Self.qualified(server.type),
                             name: server.name)
        super.init()
        service.delegate = self
    }

    private static func qualified(_ s: String) -> String {
        s.hasSuffix(".") ? s : s + "."
    }

    /// Resolves, then calls completion exactly once on the main queue.
    func resolve(timeout: TimeInterval = 5, completion: @escaping (Outcome) -> Void) {
        self.completion = completion
        keepAlive = self
        service.resolve(withTimeout: timeout)
    }

    private func deliver(_ outcome: Outcome) {
        service.stop()
        let done = completion
        completion = nil
        DispatchQueue.main.async {
            done?(outcome)
            self.keepAlive = nil
        }
    }

    nonisolated func netServiceDidResolveAddress(_ sender: NetService) {
        // Only the two values are carried onto the main actor, not the service.
        let host = sender.hostName ?? ""
        let port = sender.port
        MainActor.assumeIsolated { resolved(host: host, port: port) }
    }

    private func resolved(host reported: String, port: Int) {
        var host = reported
        while host.hasSuffix(".") { host.removeLast() }

        // An SRV target is not a trusted string: a host name is accepted only
        // as the letters, digits, dots and hyphens a host name is made of, and
        // the URL is built field by field, so nothing in the host can reach
        // across into another component and send the bearer token elsewhere.
        let hostCharacters = CharacterSet(charactersIn:
            "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-.")
        guard !host.isEmpty, host.count <= 253,
              host.unicodeScalars.allSatisfy(hostCharacters.contains),
              (1...65535).contains(port)
        else {
            deliver(.failure("That server reported an address this client will not use."))
            return
        }

        var url = URLComponents()
        url.scheme = "http"
        url.host = host
        url.port = port
        guard let address = url.string else {
            deliver(.failure("That server did not report an address."))
            return
        }
        deliver(.address(address))
    }

    nonisolated func netService(_ sender: NetService, didNotResolve errorDict: [String: NSNumber]) {
        MainActor.assumeIsolated {
            deliver(.failure("Could not work out that server's address — it may have left the network."))
        }
    }

    /// Abandons the resolve: the completion is never called.
    func cancel() {
        service.stop()
        completion = nil
        DispatchQueue.main.async { self.keepAlive = nil }
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
        if let legacy = UserDefaults.standard.string(forKey: "apiKey"), !legacy.isEmpty {
            Keychain.write(legacy)
            UserDefaults.standard.removeObject(forKey: "apiKey")
        }
        apiKey = Keychain.read()
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
            .appendingPathComponent("GropiusChat", isDirectory: true)
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

    /// An authenticated request for a path under the API base, or nil when
    /// the stored address is not one a request may be sent to. The bearer
    /// token is attached only to the host it was entered for: a server picked
    /// from the network is asked without it, and says so if it needs one.
    func request(_ path: String) -> URLRequest? {
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
            let (data, resp) = try await URLSession.shared.data(for: req)
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
            status = "Could not reach \(serverHost). Is Gropius running and on the same network?"
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
        guard let (data, resp) = try? await URLSession.shared.data(for: req),
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
        case .server(let model): backend = ServerBackend(model: model, request: { [weak self] in self?.request($0) })
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

    var dynamicType: DynamicTypeSize {
        switch self {
        case .smaller: return .small
        case .standard: return .large
        case .larger: return .xLarge
        case .extraLarge: return .xxLarge
        case .huge: return .xxxLarge
        }
    }
}

@main
struct GropiusChatApp: App {
    @StateObject private var model = AppModel.shared
    @FocusedValue(\.chatActions) private var actions
    @AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue

    private var dynamicType: DynamicTypeSize {
        TextSize(rawValue: textSize)?.dynamicType ?? .large
    }

    var body: some Scene {
        WindowGroup("Gropius Chat") {
            RootView(model: model)
                .frame(minWidth: 720, minHeight: 480)
                .dynamicTypeSize(dynamicType)
        }
        .commands {
            CommandGroup(replacing: .newItem) {
                Button("New Chat") { actions?.newChat() }
                    .keyboardShortcut("n", modifiers: .command)
                    .disabled(actions == nil)
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
            }
        }
        Settings {
            SettingsView(model: model)
                .dynamicTypeSize(dynamicType)
        }
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
    /// The window's own chat, kept with the window so it comes back after a
    /// relaunch.
    @SceneStorage("selectedConversation") private var stored: String = ""
    @State private var selection: UUID?
    @State private var draft = ""
    @State private var pickerShown = false
    @Environment(\.controlActiveState) private var activeState

    var body: some View {
        NavigationSplitView {
            Sidebar(model: model, selection: $selection)
                .navigationSplitViewColumnWidth(min: 200, ideal: 240, max: 340)
        } detail: {
            ChatDetail(model: model, conversationID: selection, draft: $draft, pickerShown: $pickerShown)
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
            // keep what they show.
            guard let asked, activeState == .key else { return }
            selection = asked.id
            model.intentSelection = nil
        }
    }
}

struct Sidebar: View {
    @ObservedObject var model: AppModel
    @Binding var selection: UUID?

    var body: some View {
        List(selection: $selection) {
            ForEach(model.conversations) { c in
                VStack(alignment: .leading, spacing: 2) {
                    Text(c.title.isEmpty ? "New Chat" : c.title)
                        .lineLimit(1)
                    Text("\(c.messages.count) message\(c.messages.count == 1 ? "" : "s")")
                        .font(.caption2).foregroundStyle(.secondary)
                }
                .tag(c.id)
                .contextMenu {
                    Button("Delete", role: .destructive) {
                        model.deleteChat(c.id)
                        if selection == c.id { selection = model.conversations.first?.id }
                    }
                }
            }
            .onDelete { offsets in
                offsets.map { model.conversations[$0].id }.forEach(model.deleteChat)
                if let s = selection, !model.conversations.contains(where: { $0.id == s }) {
                    selection = model.conversations.first?.id
                }
            }
        }
        .navigationTitle("Chats")
        .toolbar {
            ToolbarItem {
                Button { selection = model.newChat() } label: { Image(systemName: "square.and.pencil") }
                    .help("New chat")
            }
        }
    }
}

struct ChatDetail: View {
    @ObservedObject var model: AppModel
    let conversationID: UUID?
    @Binding var draft: String
    @Binding var pickerShown: Bool

    private var messages: [Message] { model.messages(in: conversationID) }

    var body: some View {
        VStack(spacing: 0) {
            transcript
            Divider()
            composer
        }
        .navigationTitle("Gropius Chat")
        .toolbar {
            ToolbarItem {
                Button {
                    pickerShown = true
                } label: {
                    Label {
                        Text(model.answerer.displayName)
                    } icon: {
                        if model.connecting {
                            ProgressView().controlSize(.small)
                        } else {
                            Image(systemName: model.answerer == .builtIn ? "apple.intelligence" : "network")
                        }
                    }
                    .labelStyle(.titleAndIcon)
                }
                .help("Choose who answers: this Mac, or a Gropius server on your network")
                .popover(isPresented: $pickerShown) {
                    ModelPickerView(model: model)
                }
            }
        }
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
                .padding()
            }
            .defaultScrollAnchor(.bottom)
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
            // filled send button, both standard controls given standard
            // shapes (the shapes are the system's, not a drawn background).
            HStack(alignment: .bottom, spacing: 8) {
                TextField("Message…", text: $draft, axis: .vertical)
                    .lineLimit(1...8)
                    .textInputBorderShape(.capsule)
                    .controlSize(.large)
                    .onSubmit(send)
                    .disabled(!model.canSend)
                    .accessibilityLabel("Message")
                if model.sending {
                    Button { model.stop() } label: { Image(systemName: "stop.fill") }
                        .buttonStyle(.borderedProminent)
                        .buttonBorderShape(.circle)
                        .controlSize(.large)
                        .help("Stop")
                } else {
                    Button(action: send) { Image(systemName: "arrow.up") }
                        .buttonStyle(.borderedProminent)
                        .buttonBorderShape(.circle)
                        .controlSize(.large)
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

    var body: some View {
        VStack(spacing: 12) {
            Image(nsImage: NSApp.applicationIconImage)
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
    /// The rendered blocks, parsed once per change of the text — and for the
    /// reply that is streaming, at most a few times a second.
    @State private var blocks: [MarkdownBlock] = []
    @State private var lastParse = Date.distantPast
    @State private var parseTask: Task<Void, Never>?
    /// Latched on first appearance: whether this row plays the effect. The
    /// model's set is consumed at that moment, so a row recreated on scroll
    /// draws plain text and a re-evaluation cannot switch the branch mid-play.
    @State private var playing: Bool?

    private var isUser: Bool { message.role == .user }
    private var displayText: String { message.text.trimmingCharacters(in: .whitespacesAndNewlines) }
    private var displayReasoning: String { message.reasoning.trimmingCharacters(in: .whitespacesAndNewlines) }

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
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(message.text, forType: .string)
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
        let animate = playing ?? false
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
        .onAppear {
            if playing == nil {
                let due = model.effectsToPlay.contains(message.id)
                playing = due
                if due { model.effectStarted(message.id) }
            }
            if blocks.isEmpty { blocks = MarkdownBlocks.parse(displayText); lastParse = Date() }
        }
        .onChange(of: displayText) { _, _ in scheduleParse() }
    }

    /// Parse now if the last parse is older than a fifth of a second, else
    /// once at that deadline; a finished reply's text never changes again.
    private func scheduleParse() {
        let wait = 0.2 - Date().timeIntervalSince(lastParse)
        parseTask?.cancel()
        if wait <= 0 {
            blocks = MarkdownBlocks.parse(displayText)
            lastParse = Date()
            return
        }
        parseTask = Task {
            try? await Task.sleep(for: .seconds(wait))
            guard !Task.isCancelled else { return }
            blocks = MarkdownBlocks.parse(displayText)
            lastParse = Date()
        }
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
    /// hidden while it is being read. Dragging still selects the text.
    @ViewBuilder private var reasoningDisclosure: some View {
        DisclosureGroup(isExpanded: $showReasoning) {
            Text(displayReasoning)
                .font(.callout).italic()
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
                .onTapGesture { withAnimation { showReasoning = false } }
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
    @AppStorage("bubbleColorUser") private var bubbleUser: String = ""
    @AppStorage("bubbleColorModel") private var bubbleModel: String = ""
    @AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue

    private var typedAddress: Binding<String> {
        Binding(get: { model.serverURL }, set: { model.useTypedAddress($0) })
    }

    var body: some View {
        Form {
            Section("Server") {
                TextField("Address", text: typedAddress, prompt: Text("http://alices-mac.local:11535"))
                Text("A Gropius server's address: the Mac's .local name or LAN address, port 11535. Servers on your network are offered in the model picker without typing anything.")
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
            Section("Models to offer") {
                TextField("Pipeline tags", text: $model.chatPipelineTags, prompt: Text("text-generation, image-text-to-text"))
                TextField("Required tags", text: $model.chatRequiredTags, prompt: Text("conversational"))
                Text("The picker offers a server's models carrying these HuggingFace words — a pipeline tag from the first list, and every tag in the second. Every model stays reachable over the API by name. Clear a field to stop testing it. The Mac's own model is always offered.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Text") {
                Picker("Text size", selection: $textSize) {
                    ForEach(TextSize.allCases, id: \.rawValue) { size in
                        Text(size.label).tag(size.rawValue)
                    }
                }
                Text("The whole window follows, at the system's own text sizes.")
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
        .frame(width: 520)
        .onAppear { key = model.apiKey }
    }
}

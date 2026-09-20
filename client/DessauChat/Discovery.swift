// Finding Dessau servers on the local network: the service type, what a
// browse yields, the browse itself, and turning the one server a person picked
// into an address the client can store.
//
// This file is compiled by both clients — the Mac's and the iPad's — which is
// why it is a file of its own and why nothing in it touches AppKit or UIKit.
// The browse is the Network framework's; the resolve is DNS-SD's own C API.
// The Foundation service class the Mac client resolved with until iPadOS came
// into the picture is deprecated on every platform at 27.2 and was never
// available on iPadOS at all — but what it answered with, the host NAME the
// server publishes, is what this client stores, so the resolve goes to the
// same API that class was written over.

import Combine
import Foundation
import Network
import dnssd

/// The mDNS service type a Dessau server advertises itself on.
///
/// It has to match the server's own (internal/discovery), and it is declared a
/// second time in each client bundle's NSBonjourServices — Local Network
/// Privacy, on both systems, answers a browse for an undeclared type with an
/// empty result set rather than an error. A test in the server's suite holds
/// every declaration to one value.
let dessauServiceType = "_dessau._tcp"

/// One Dessau server seen on the local network. Everything here comes out of
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

/// Browses the local network for Dessau servers. It only ever lists what it
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
            for: .bonjourWithTXTRecord(type: dessauServiceType, domain: nil),
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
/// NWBrowser, resolve with DNS-SD, which answers with the SRV record's target
/// — the host NAME the server publishes its addresses under — and its port.
///
/// The name is the point. An address is a lease: it moves, and a hostile
/// advertiser that later takes the same address would inherit everything bound
/// to it, including the API key this client binds to the host it was entered
/// for. A name is the thing the person picked, it keeps resolving after the
/// server's address changes, and it is what the Foundation service class this
/// resolver replaces returned. That class is deprecated at 27.2 and absent on
/// iPadOS; the C API underneath it is neither, and is the same one on both
/// systems.
final class ServiceResolver {
    enum Outcome {
        case address(String)
        case failure(String)
    }

    /// The two pointers, handed to the queue that frees them. Pointers carry
    /// no thread safety of their own; these are given away exactly once, to
    /// the one queue the callbacks run on.
    private nonisolated struct Handover: @unchecked Sendable {
        let service: DNSServiceRef
        let context: UnsafeMutableRawPointer?
    }

    private let name: String
    private let type: String
    private let domain: String

    private var completion: ((Outcome) -> Void)?
    private var timeout: Task<Void, Never>?
    /// Held until an outcome is delivered.
    private var keepAlive: ServiceResolver?
    private var service: DNSServiceRef?
    private var context: UnsafeMutableRawPointer?
    /// The callbacks' queue. Their work is two values and a hop to the main
    /// actor, and the service reference is deallocated here too, because a
    /// deallocation racing its own callback is the one way to crash this API.
    private let queue = DispatchQueue(label: "sh.intentdriven.dessau.chat.resolve")

    init(server: DiscoveredServer) {
        name = server.name
        type = server.type
        domain = server.domain
    }

    /// Resolves, then calls completion exactly once on the main actor.
    func resolve(timeout seconds: TimeInterval = 5, completion: @escaping (Outcome) -> Void) {
        self.completion = completion
        keepAlive = self

        let box = ResolveContext(self)
        let context = Unmanaged.passRetained(box).toOpaque()
        self.context = context

        var service: DNSServiceRef?
        let started = DNSServiceResolve(
            &service,
            0,
            0,  // every interface: the server may be on any of them.
            name, type, domain,
            dessauResolveReply,
            context)
        guard started == kDNSServiceErr_NoError, let service else {
            deliver(.failure("Could not work out that server's address — it may have left the network."))
            return
        }
        self.service = service
        DNSServiceSetDispatchQueue(service, queue)

        // DNS-SD keeps waiting for an answer that may never come, so the wait
        // is ours to bound, as it was when the old resolver was given the same
        // number.
        timeout = Task { [weak self] in
            try? await Task.sleep(for: .seconds(seconds))
            guard !Task.isCancelled else { return }
            self?.deliver(.failure("That server did not answer in time — it may have left the network."))
        }
    }

    fileprivate func failed() {
        deliver(.failure("Could not work out that server's address — it may have left the network."))
    }

    fileprivate func resolved(host reported: String, port: Int) {
        // The SRV target is fully qualified; the trailing dot is the root, and
        // no URL carries it.
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

    private func deliver(_ outcome: Outcome) {
        // Whichever of the three paths arrives first — an answer, a failure, a
        // timeout — the other two find no completion here and do nothing.
        guard let done = completion else { return }
        completion = nil
        stop()
        done(outcome)
        keepAlive = nil
    }

    /// Abandons the resolve: the completion is never called.
    func cancel() {
        completion = nil
        stop()
        keepAlive = nil
    }

    private func stop() {
        timeout?.cancel()
        timeout = nil
        guard let service else { return }
        let handover = Handover(service: service, context: context)
        self.service = nil
        self.context = nil
        // On the callbacks' own queue: deallocating elsewhere can free the
        // reference while a callback is running on it.
        queue.async {
            DNSServiceRefDeallocate(handover.service)
            if let context = handover.context {
                Unmanaged<ResolveContext>.fromOpaque(context).release()
            }
        }
    }
}

/// What the C callback is handed back: it takes no context of its own, so the
/// one pointer it carries is this box, retained for the resolve's lifetime and
/// released with the service reference.
private nonisolated final class ResolveContext: @unchecked Sendable {
    let resolver: ServiceResolver
    init(_ resolver: ServiceResolver) { self.resolver = resolver }
}

/// The DNS-SD reply. It is nonisolated, and a free function rather than a
/// closure, because a C callback arrives on whatever queue the API was given
/// and nothing about it is the main actor's; the two values it reads are
/// carried there before anything looks at them.
private nonisolated func dessauResolveReply(
    _ service: DNSServiceRef?,
    _ flags: DNSServiceFlags,
    _ interface: UInt32,
    _ error: DNSServiceErrorType,
    _ fullname: UnsafePointer<CChar>?,
    _ target: UnsafePointer<CChar>?,
    _ port: UInt16,
    _ txtLength: UInt16,
    _ txtRecord: UnsafePointer<UInt8>?,
    _ context: UnsafeMutableRawPointer?
) {
    guard let context else { return }
    let box = Unmanaged<ResolveContext>.fromOpaque(context).takeUnretainedValue()
    guard error == kDNSServiceErr_NoError, let target else {
        Task { @MainActor in box.resolver.failed() }
        return
    }
    // Only the two values cross to the main actor, never the service reference
    // or the C strings behind them.
    let host = String(cString: target)
    // The SRV port is on the wire in network byte order.
    let resolved = Int(UInt16(bigEndian: port))
    Task { @MainActor in box.resolver.resolved(host: host, port: resolved) }
}

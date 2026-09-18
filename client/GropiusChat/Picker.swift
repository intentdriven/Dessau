// The model picker: the Mac's own model first, then the Gropius servers on
// the network. It is a standard popover holding a List, and the Bonjour browse
// runs exactly as long as the popover is shown — which is why the Mac asks for
// local-network permission the first time the picker opens, not at launch.
// Finding a server is an offer: nothing here switches by itself.

import SwiftUI

struct ModelPickerView: View {
    @ObservedObject var model: AppModel
    @StateObject private var browser = ServerBrowser()
    /// The server being resolved and connected to, and the last failure.
    @State private var busy: String?
    @State private var failure: String?
    /// The server whose models are listed, once it has answered.
    @State private var expanded: String?
    @State private var resolver: ServiceResolver?
    @State private var generation = 0
    @State private var waitedLong = false

    var body: some View {
        List {
            Section("On this Mac") {
                Button {
                    model.chooseBuiltIn()
                } label: {
                    Label {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(BuiltInBackend.displayName)
                            if let why = model.builtInUnavailable {
                                Text(why.reason).font(.caption).foregroundStyle(.secondary)
                            } else {
                                Text("Apple Intelligence, on the device").font(.caption).foregroundStyle(.secondary)
                            }
                        }
                    } icon: {
                        Image(systemName: model.answerer == .builtIn ? "checkmark.circle.fill" : "circle")
                    }
                }
                .disabled(model.builtInUnavailable != nil)
            }
            Section("Servers on your network") {
                servers
                if let failure {
                    Label(failure, systemImage: "exclamationmark.triangle").font(.caption)
                }
                if !model.serverURL.isEmpty, expanded == nil, model.connected {
                    storedServer
                }
            }
        }
        .frame(minWidth: 320, minHeight: 240)
        .onAppear {
            browser.start()
            Task {
                try? await Task.sleep(for: .seconds(5))
                waitedLong = true
            }
        }
        .onDisappear {
            browser.stop()
            resolver?.cancel()
            resolver = nil
            busy = nil
        }
    }

    /// The server the client is already pointed at, when the browse has not
    /// (yet) listed it: its models are offered under its address.
    @ViewBuilder private var storedServer: some View {
        DisclosureGroup(model.serverHost) {
            modelRows
        }
    }

    @ViewBuilder private var servers: some View {
        switch browser.status {
        case .failed(let why):
            Label("Could not search the local network: \(why)", systemImage: "exclamationmark.triangle")
                .font(.caption)
        case .waiting(let why):
            Label("Waiting to search the local network — \(why)", systemImage: "clock")
                .font(.caption)
        case .stopped, .searching:
            if browser.servers.isEmpty {
                if waitedLong {
                    Text("No server found. A server elsewhere can be typed into Settings.")
                        .font(.caption).foregroundStyle(.secondary)
                } else {
                    Label { Text("Looking…") } icon: { ProgressView().controlSize(.small) }
                        .font(.caption).foregroundStyle(.secondary)
                }
            } else {
                ForEach(browser.servers) { server in
                    serverRow(server)
                }
            }
        }
    }

    @ViewBuilder private func serverRow(_ server: DiscoveredServer) -> some View {
        DisclosureGroup(isExpanded: Binding(
            get: { expanded == server.id },
            set: { open in
                if open { pick(server) } else if expanded == server.id { expanded = nil }
            })) {
            if busy == server.id {
                Label { Text("Asking for its models…") } icon: { ProgressView().controlSize(.small) }
                    .font(.caption)
            } else if expanded == server.id {
                modelRows
            }
        } label: {
            Label {
                VStack(alignment: .leading, spacing: 2) {
                    Text(server.name)
                    Text(server.summary).font(.caption).foregroundStyle(.secondary)
                }
            } icon: {
                Image(systemName: server.authRequired ? "lock" : "network")
            }
        }
    }

    /// The chat models of the server the client is connected to, or why there
    /// are none.
    @ViewBuilder private var modelRows: some View {
        if !model.connected {
            Text(model.status).font(.caption).foregroundStyle(.secondary)
        } else if model.chatModels.isEmpty {
            Text(model.models.isEmpty
                 ? "This server has no models downloaded yet."
                 : "None of this server's models can hold a conversation; they stay callable over the API by name.")
                .font(.caption).foregroundStyle(.secondary)
        } else {
            ForEach(model.chatModels, id: \.self) { id in
                Button {
                    model.chooseServerModel(id)
                } label: {
                    Label(Answerer.server(model: id).displayName,
                          systemImage: model.answerer == .server(model: id) ? "checkmark.circle.fill" : "circle")
                }
            }
        }
    }

    /// Resolve the picked server to an address, point the client at it and ask
    /// it for its models. Only the newest pick may land: a slow resolve for a
    /// server the person moved off must not overwrite a faster one.
    private func pick(_ server: DiscoveredServer) {
        resolver?.cancel()
        failure = nil
        busy = server.id
        generation += 1
        let mine = generation
        let r = ServiceResolver(server: server)
        resolver = r
        r.resolve { outcome in
            guard mine == generation else { return }
            resolver = nil
            switch outcome {
            case .address(let address):
                model.use(server, resolvedAddress: address)
                Task {
                    await model.connect()
                    guard mine == generation else { return }
                    busy = nil
                    expanded = server.id
                }
            case .failure(let why):
                busy = nil
                failure = "\(server.name): \(why)"
            }
        }
    }
}

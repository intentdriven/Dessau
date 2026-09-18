// The client's actions for Shortcuts and Spotlight, declared as App Intents.
// They run inside the app's own process (the system launches it in the
// background when it is not running) and reach the same model object the
// windows use, so the conversations file has one writer. Chats are looked up
// on demand when a person picks one; nothing is donated to the system index.

import AppIntents
import Foundation

/// Sends a prompt to whichever answerer the client is set to and returns the
/// reply as text; the exchange is kept as a new chat. Long-running, so a slow
/// model on a server does not hit the system's default limit.
struct AskIntent: AppIntent, LongRunningIntent {
    static let title: LocalizedStringResource = "Ask Gropius"
    static let description = IntentDescription("Sends a prompt to the current model and returns the reply as text. The exchange is kept as a new chat.")

    @Parameter(title: "Prompt") var prompt: String

    static var parameterSummary: some ParameterSummary { Summary("Ask \(\.$prompt)") }

    let progress = Progress(totalUnitCount: 1)

    func perform() async throws -> some IntentResult & ReturnsValue<String> {
        let reply = try await performBackgroundTask {
            try await AppModel.shared.ask(prompt: prompt)
        }
        progress.completedUnitCount = 1
        return .result(value: reply)
    }
}

/// Starts a new chat and brings the client forward.
struct NewChatIntent: AppIntent {
    static let title: LocalizedStringResource = "New Chat"
    static let description = IntentDescription("Opens GropiusChat on a new chat.")
    static let openAppWhenRun = true

    func perform() async throws -> some IntentResult {
        await AppModel.shared.newChat()
        return .result()
    }
}

/// Opens an existing chat, chosen by its title.
struct OpenChatIntent: AppIntent {
    static let title: LocalizedStringResource = "Open Chat"
    static let description = IntentDescription("Opens GropiusChat on one of its chats.")
    static let openAppWhenRun = true

    @Parameter(title: "Chat") var chat: ChatEntity

    static var parameterSummary: some ParameterSummary { Summary("Open \(\.$chat)") }

    func perform() async throws -> some IntentResult {
        await AppModel.shared.select(chat.id)
        return .result()
    }
}

/// One chat, as Shortcuts sees it: its id and its title — the title being the
/// first words of the first prompt — and nothing else of what was said.
struct ChatEntity: AppEntity {
    static let typeDisplayRepresentation: TypeDisplayRepresentation = "Chat"
    static let defaultQuery = ChatQuery()

    let id: UUID
    let title: String

    var displayRepresentation: DisplayRepresentation {
        DisplayRepresentation(title: "\(title)")
    }
}

struct ChatQuery: EntityStringQuery {
    func entities(for identifiers: [UUID]) async throws -> [ChatEntity] {
        await AppModel.shared.conversations
            .filter { identifiers.contains($0.id) }
            .map { ChatEntity(id: $0.id, title: $0.title) }
    }

    func entities(matching string: String) async throws -> [ChatEntity] {
        await AppModel.shared.conversations
            .filter { $0.title.localizedCaseInsensitiveContains(string) }
            .map { ChatEntity(id: $0.id, title: $0.title) }
    }

    func suggestedEntities() async throws -> [ChatEntity] {
        await AppModel.shared.conversations.prefix(10).map { ChatEntity(id: $0.id, title: $0.title) }
    }
}

struct GropiusShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(intent: AskIntent(),
                    phrases: ["Ask \(.applicationName)"],
                    shortTitle: "Ask",
                    systemImageName: "bubble.left.and.text.bubble.right")
        AppShortcut(intent: NewChatIntent(),
                    phrases: ["New chat in \(.applicationName)"],
                    shortTitle: "New Chat",
                    systemImageName: "square.and.pencil")
        AppShortcut(intent: OpenChatIntent(),
                    phrases: ["Open a chat in \(.applicationName)"],
                    shortTitle: "Open Chat",
                    systemImageName: "bubble.left")
    }
}

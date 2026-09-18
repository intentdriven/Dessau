// A reply's markdown, rendered: the system's parser turns the text into an
// attributed string whose runs carry presentation intents (paragraph, heading,
// list item, block quote, code block) and inline intents (emphasis, strong,
// code, link). Text draws the inline ones itself; the blocks are split here so
// each can be laid out as its kind. No third-party renderer, no dependency.

import Foundation

/// One block of a rendered reply. Blocks are laid out by position, so two
/// identical blocks are two rows.
enum MarkdownBlock: Equatable {
    case paragraph(AttributedString)
    case heading(level: Int, AttributedString)
    /// `ordinal` is nil for a bulleted item.
    case listItem(ordinal: Int?, AttributedString)
    case quote(AttributedString)
    /// Drawn in the monospaced face with its line breaks kept.
    case code(String)
    /// A construct Text cannot draw (a table); shown as the model wrote it.
    case plain(String)
}

enum MarkdownBlocks {
    /// The parser options: the full syntax, so block intents are produced, and
    /// a partial parse rather than nothing when a reply is cut mid-mark. The
    /// architecture test reads these two names.
    static let options = AttributedString.MarkdownParsingOptions(
        allowsExtendedAttributes: false,
        interpretedSyntax: .full,
        failurePolicy: .returnPartiallyParsedIfPossible)

    /// Splits a reply into blocks. Tables, which the intents describe and
    /// Text cannot draw, are cut out first and shown as the model wrote them.
    static func parse(_ text: String) -> [MarkdownBlock] {
        var blocks: [MarkdownBlock] = []
        for (isTable, segment) in tableSegments(text) {
            if isTable {
                blocks.append(.plain(segment.trimmingCharacters(in: .newlines)))
            } else {
                blocks.append(contentsOf: parseProse(segment))
            }
        }
        return blocks
    }

    /// Runs of lines that begin with a pipe are a table; everything else is
    /// prose. A pipe inside a fenced code block is code, not a table.
    private static func tableSegments(_ text: String) -> [(Bool, String)] {
        var out: [(Bool, String)] = []
        var current = ""
        var currentIsTable = false
        var inFence = false
        for line in text.split(separator: "\n", omittingEmptySubsequences: false) {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("```") || trimmed.hasPrefix("~~~") { inFence.toggle() }
            let isTable = !inFence && trimmed.hasPrefix("|")
            if isTable != currentIsTable, !current.isEmpty {
                out.append((currentIsTable, current))
                current = ""
            }
            currentIsTable = isTable
            current += (current.isEmpty ? "" : "\n") + line
        }
        if !current.isEmpty { out.append((currentIsTable, current)) }
        return out
    }

    private static func parseProse(_ text: String) -> [MarkdownBlock] {
        guard let parsed = try? AttributedString(markdown: text, options: options) else {
            return [.plain(text)]
        }
        var blocks: [MarkdownBlock] = []
        var currentKey: [Int] = []
        var current = AttributedString()
        var currentKind: MarkdownBlock? = nil

        func flush() {
            guard let kind = currentKind, !current.characters.isEmpty else { return }
            var s = current
            s.presentationIntent = nil
            switch kind {
            case .paragraph: blocks.append(.paragraph(s))
            case .heading(let level, _): blocks.append(.heading(level: level, s))
            case .listItem(let ordinal, _): blocks.append(.listItem(ordinal: ordinal, s))
            case .quote: blocks.append(.quote(s))
            case .code: blocks.append(.code(String(s.characters)))
            case .plain: blocks.append(.plain(String(s.characters)))
            }
            current = AttributedString()
        }

        for run in parsed.runs {
            let intent = run.presentationIntent
            let key = intent?.components.map(\.identity) ?? []
            if key != currentKey {
                flush()
                currentKey = key
                currentKind = kind(of: intent)
            }
            current.append(parsed[run.range])
        }
        flush()
        return blocks
    }

    /// The block a run belongs to, from the innermost intent outwards.
    private static func kind(of intent: PresentationIntent?) -> MarkdownBlock {
        guard let intent else { return .paragraph(AttributedString()) }
        var ordinal: Int? = nil
        var inList = false
        for component in intent.components {
            switch component.kind {
            case .codeBlock: return .code("")
            case .header(let level): return .heading(level: level, AttributedString())
            case .blockQuote: return .quote(AttributedString())
            case .listItem(let n): ordinal = n
            case .orderedList: inList = true
            case .unorderedList: ordinal = nil; inList = true
            case .table, .tableRow, .tableCell, .tableHeaderRow: return .plain("")
            default: break
            }
        }
        if inList || ordinal != nil {
            // An ordered list carries the ordinal; a bulleted one has none.
            let ordered = intent.components.contains { if case .orderedList = $0.kind { return true } else { return false } }
            return .listItem(ordinal: ordered ? ordinal : nil, AttributedString())
        }
        return .paragraph(AttributedString())
    }
}

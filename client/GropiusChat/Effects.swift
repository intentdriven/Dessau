// Replies come alive on certain words: a small, fixed list of words that earn
// an effect, and the renderer that draws it once, in place, on those words
// alone. The list is one constant, named on the Settings page; the switch is
// one setting. Drawing only: the text is never changed.
//
// This file is the one named exception to the client's no-styling rule — a
// TextRenderer is client-owned drawing by definition — and the architecture
// test that holds the rule names it as such.

import SwiftUI

/// What a matched word does.
enum EffectKind: String, CaseIterable {
    case shimmer, shake, bounce
}

/// The words, whole and case-insensitive, and what each earns. Six words:
/// small enough to stay a surprise.
enum EffectWords {
    static let list: [(word: String, kind: EffectKind)] = [
        ("congratulations", .shimmer),
        ("well done", .shimmer),
        ("warning", .shake),
        ("careful", .shake),
        ("wow", .bounce),
        ("amazing", .bounce),
    ]

    /// The sentence the Settings page shows beside the switch.
    static var settingsSentence: String {
        "Words such as " + list.map { "“\($0.word)”" }.joined(separator: ", ") + " animate once when a reply arrives."
    }

    /// One match in a block's plain text.
    struct Match: Equatable {
        let range: Range<String.Index>
        let kind: EffectKind
    }

    /// Whole-word, case-insensitive matches in one block of text: "Warning:"
    /// matches, "forewarning" does not.
    static func matches(in text: String) -> [Match] {
        var out: [Match] = []
        for (word, kind) in list {
            var search = text.startIndex
            while let r = text.range(of: word, options: [.caseInsensitive], range: search..<text.endIndex) {
                let before = r.lowerBound == text.startIndex ? nil : text[text.index(before: r.lowerBound)]
                let after = r.upperBound == text.endIndex ? nil : text[r.upperBound]
                if before.map({ !$0.isLetter }) ?? true, after.map({ !$0.isLetter }) ?? true {
                    out.append(Match(range: r, kind: kind))
                }
                search = r.upperBound
            }
        }
        return out.sorted { $0.range.lowerBound < $1.range.lowerBound }
    }
}

/// Tags the glyph runs the renderer animates.
struct EffectAttribute: TextAttribute {
    let kind: EffectKind
}

/// Draws a block of text, animating only the runs tagged with EffectAttribute.
/// `progress` runs 0 to 1 over the effect's duration and then the renderer is
/// removed, so the block goes back to a plain, selectable Text.
struct EffectRenderer: TextRenderer {
    var progress: Double

    var animatableData: Double {
        get { progress }
        set { progress = newValue }
    }

    func draw(layout: Text.Layout, in ctx: inout GraphicsContext) {
        for line in layout {
            for run in line {
                guard let effect = run[EffectAttribute.self] else {
                    ctx.draw(run)
                    continue
                }
                switch effect.kind {
                case .shimmer:
                    // A brightening sweep across the glyphs, left to right.
                    for (i, slice) in run.enumerated() {
                        let phase = Double(i) / Double(max(run.count, 1))
                        // The band clears the run before progress reaches 1,
                        // so nothing pops when the renderer is removed.
                        let d = abs(progress * 1.6 - 0.3 - phase)
                        var copy = ctx
                        copy.opacity = progress >= 0.95 || d < 0.15 ? 1 : 0.55 + 0.45 * min(1, d)
                        copy.draw(slice)
                    }
                case .shake:
                    let amplitude = 3.0 * sin(progress * .pi)
                    var copy = ctx
                    copy.translateBy(x: amplitude * sin(progress * .pi * 8), y: 0)
                    copy.draw(run)
                case .bounce:
                    for (i, slice) in run.enumerated() {
                        let phase = Double(i) * 0.12
                        let lift = max(0, sin((progress - phase) * .pi * 2)) * 6 * (1 - progress)
                        var copy = ctx
                        copy.translateBy(x: 0, y: -lift)
                        copy.draw(slice)
                    }
                }
            }
        }
    }
}

/// A block of a reply with its matched words animated once.
struct EffectText: View {
    let text: AttributedString
    let matches: [EffectWords.Match]
    @State private var progress: Double = 0
    @State private var finished = false

    var body: some View {
        if finished || matches.isEmpty {
            Text(text).textSelection(.enabled)
        } else {
            tagged
                .textRenderer(EffectRenderer(progress: progress))
                // The animated block is a rebuilt Text and would otherwise be
                // unselectable for the effect's second: selection is the
                // reply's, whatever is being drawn over it.
                .textSelection(.enabled)
                .onAppear {
                    withAnimation(.linear(duration: 1.2)) { progress = 1 }
                    Task {
                        try? await Task.sleep(for: .seconds(1.3))
                        finished = true
                    }
                }
        }
    }

    /// The block rebuilt as concatenated Texts so the matched words carry the
    /// attribute the renderer looks for.
    private var tagged: Text {
        let plain = String(text.characters)
        var pieces: [Text] = []
        var cursor = plain.startIndex
        for m in matches where m.range.lowerBound >= cursor {
            pieces.append(Text(slice(cursor..<m.range.lowerBound)))
            pieces.append(Text(slice(m.range)).customAttribute(EffectAttribute(kind: m.kind)))
            cursor = m.range.upperBound
        }
        pieces.append(Text(slice(cursor..<plain.endIndex)))
        return pieces.reduce(Text("")) { Text("\($0)\($1)") }
    }

    private func slice(_ r: Range<String.Index>) -> AttributedString {
        let lower = AttributedString.Index(r.lowerBound, within: text) ?? text.startIndex
        let upper = AttributedString.Index(r.upperBound, within: text) ?? text.endIndex
        return AttributedString(text[lower..<upper])
    }
}

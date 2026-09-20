#if os(macOS)
import AppKit
#endif
import SwiftUI

/// The shape of the composer's field.
///
/// Messages and WhatsApp give the composer a capsule about 34 points tall at
/// the standard text size, with a clear gap before the first glyph. This
/// client's field was 24 points tall with about 7 points of gap, so the
/// placeholder sat against the capsule's curve (iss-2609190004092322).
///
/// SwiftUI's own bordered capsule offers no way to inset its text: padding,
/// safe-area padding and content margins all move the capsule along with the
/// text, and an extra-large control size changes neither. So the capsule is
/// drawn here, in the system's own material and shape, which makes this file
/// the no-styling rule's third named exception — one file wide, like the
/// transcript's bubbles. A plain field draws no focus effect, so the ring a
/// bordered field shows for itself is drawn here too (iss-2609190034161350).
enum ComposerMetrics {
    /// The capsule's height with one line of text, at the standard text size.
    static let fieldMinHeight: CGFloat = 34
    /// The gap between the capsule's edge and the first glyph.
    static let textInset: CGFloat = 12
    /// The gap above and below the text, which is where a second line grows
    /// from.
    static let textVerticalInset: CGFloat = 8
    /// The thickness of the ring the system strokes around a bordered field
    /// that has keyboard focus. It is a constant, not a scaled metric: the
    /// system's ring is the same width whatever the text size.
    static let focusRingWidth: CGFloat = 3
}

/// The colour the system draws keyboard focus in.
///
/// Both systems supply one, so the client never picks a focus colour: macOS
/// has a named focus-indicator colour that follows the highlight setting, and
/// on iPadOS the accent colour is what the system tints focus with.
private var systemFocusRingColor: Color {
    #if os(macOS)
    Color(nsColor: .keyboardFocusIndicatorColor)
    #else
    Color.accentColor
    #endif
}

/// The composer's field: a capsule of `ComposerMetrics.fieldMinHeight` at the
/// smallest, its text inset by `ComposerMetrics.textInset`, growing with the
/// lines it is given. Every metric is scaled, so the text-size setting grows
/// the field and not only the glyphs in it.
struct ComposerFieldCapsule: ViewModifier {
    @ScaledMetric(relativeTo: .body) private var minHeight = ComposerMetrics.fieldMinHeight
    @ScaledMetric(relativeTo: .body) private var horizontal = ComposerMetrics.textInset
    @ScaledMetric(relativeTo: .body) private var vertical = ComposerMetrics.textVerticalInset
    /// Whether the field this capsule is drawn around has keyboard focus. The
    /// state lives here rather than in the chat view because the capsule is
    /// the only thing that reads it, and the exception stays one file wide.
    @FocusState private var focused: Bool
    /// Whether focus effects may be drawn at all. The system's focus-ring
    /// preference and any enclosing `focusEffectDisabled()` both arrive here,
    /// which is how a view that draws its own ring still obeys them.
    @Environment(\.isFocusEffectEnabled) private var focusEffectEnabled

    func body(content: Content) -> some View {
        content
            .textFieldStyle(.plain)
            .focused($focused)
            .padding(.horizontal, horizontal)
            .padding(.vertical, vertical)
            .frame(minHeight: minHeight)
            .background(.quaternary, in: .capsule)
            .overlay {
                if focused && focusEffectEnabled {
                    Capsule().strokeBorder(systemFocusRingColor, lineWidth: ComposerMetrics.focusRingWidth)
                }
            }
    }
}

/// The send button's circle, given the field's own height so that it reads as
/// centred beside a single line and stays at the foot of a field that has
/// grown to several.
struct ComposerButtonCircle: ViewModifier {
    @ScaledMetric(relativeTo: .body) private var diameter = ComposerMetrics.fieldMinHeight

    func body(content: Content) -> some View {
        content.frame(width: diameter, height: diameter)
    }
}

extension View {
    /// Draws this field as the composer's capsule.
    func composerFieldCapsule() -> some View { modifier(ComposerFieldCapsule()) }

    /// Sizes this button to the composer field's height.
    func composerButtonCircle() -> some View { modifier(ComposerButtonCircle()) }
}

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
/// transcript's bubbles.
enum ComposerMetrics {
    /// The capsule's height with one line of text, at the standard text size.
    static let fieldMinHeight: CGFloat = 34
    /// The gap between the capsule's edge and the first glyph.
    static let textInset: CGFloat = 12
    /// The gap above and below the text, which is where a second line grows
    /// from.
    static let textVerticalInset: CGFloat = 8
}

/// The composer's field: a capsule of `ComposerMetrics.fieldMinHeight` at the
/// smallest, its text inset by `ComposerMetrics.textInset`, growing with the
/// lines it is given. Every metric is scaled, so the text-size setting grows
/// the field and not only the glyphs in it.
struct ComposerFieldCapsule: ViewModifier {
    @ScaledMetric(relativeTo: .body) private var minHeight = ComposerMetrics.fieldMinHeight
    @ScaledMetric(relativeTo: .body) private var horizontal = ComposerMetrics.textInset
    @ScaledMetric(relativeTo: .body) private var vertical = ComposerMetrics.textVerticalInset

    func body(content: Content) -> some View {
        content
            .textFieldStyle(.plain)
            .padding(.horizontal, horizontal)
            .padding(.vertical, vertical)
            .frame(minHeight: minHeight)
            .background(.quaternary, in: .capsule)
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

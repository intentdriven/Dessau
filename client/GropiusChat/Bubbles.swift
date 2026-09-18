// Speech bubbles, the way Messages draws them: the person's message in the
// system's accent colour, the model's in grey, both changeable in Settings.
//
// A bubble is a filled shape behind the text, which is drawing of the client's
// own; this file is the no-styling rule's second named exception (the first is
// the text-effects renderer), kept one file wide, and the architecture test
// that holds the rule names it. Every default is a colour the system supplies.

import SwiftUI

/// The two bubble colours, stored as hex so a picker's choice survives a
/// relaunch; empty means the default.
struct BubbleColors {
    @AppStorage("bubbleColorUser") var user: String = ""
    @AppStorage("bubbleColorModel") var model: String = ""

    static let defaultUser = Color.accentColor
    static let defaultModel = Color.gray

    var userColor: Color { Color(hex: user) ?? Self.defaultUser }
    var modelColor: Color { Color(hex: model) ?? Self.defaultModel }
}

extension Color {
    /// "#RRGGBB" or "#RRGGBBAA"; nil for anything else, which is the default.
    init?(hex: String) {
        var s = hex.trimmingCharacters(in: .whitespaces)
        if s.hasPrefix("#") { s.removeFirst() }
        guard s.count == 6 || s.count == 8, let v = UInt64(s, radix: 16) else { return nil }
        let a = s.count == 8 ? Double(v & 0xFF) / 255 : 1
        let shift: UInt64 = s.count == 8 ? 8 : 0
        self.init(red: Double((v >> (16 + shift)) & 0xFF) / 255,
                  green: Double((v >> (8 + shift)) & 0xFF) / 255,
                  blue: Double((v >> shift) & 0xFF) / 255,
                  opacity: a)
    }

    /// The colour's sRGB components as it resolves on this system now.
    private var components: (r: CGFloat, g: CGFloat, b: CGFloat, a: CGFloat) {
        #if os(macOS)
        let native = NSColor(self).usingColorSpace(.sRGB) ?? NSColor(self)
        return (native.redComponent, native.greenComponent, native.blueComponent, native.alphaComponent)
        #else
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        UIColor(self).getRed(&r, green: &g, blue: &b, alpha: &a)
        return (r, g, b, a)
        #endif
    }

    /// The colour as "#RRGGBBAA", for storing a picker's choice.
    var hexString: String {
        let (r, g, b, a) = components
        return String(format: "#%02X%02X%02X%02X", Int(r * 255), Int(g * 255), Int(b * 255), Int(a * 255))
    }

    /// Whether white text reads on this colour: the standard sRGB luminance,
    /// against the mid point every system control uses to make the same call.
    var isDark: Bool {
        let (r, g, b, _) = components
        return 0.299 * r + 0.587 * g + 0.114 * b < 0.6
    }
}

/// A Messages-style bubble: a continuous rounded rectangle in the speaker's
/// colour behind the text, with the text in the colour that reads on it.
struct BubbleModifier: ViewModifier {
    let color: Color
    let isUser: Bool

    func body(content: Content) -> some View {
        content
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            // White reads on a dark fill and disappears on a light one, and a
            // bubble colour is the person's to choose: the text follows the
            // bubble it sits on. The model's bubble is drawn at a quarter's
            // opacity over the window, where the system's own text colour is
            // the one that reads in either appearance.
            .foregroundStyle(isUser && color.isDark ? Color.white : Color.primary)
            .background(color.opacity(isUser ? 1 : 0.28), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
    }
}

extension View {
    func bubble(_ color: Color, isUser: Bool) -> some View {
        modifier(BubbleModifier(color: color, isUser: isUser))
    }
}

/// A colour picker bound to a stored bubble colour, with a way back to the
/// default.
struct BubbleColorRow: View {
    let title: String
    @Binding var stored: String
    let fallback: Color

    var body: some View {
        HStack {
            ColorPicker(title, selection: Binding(
                get: { Color(hex: stored) ?? fallback },
                set: { stored = $0.hexString }))
            if !stored.isEmpty {
                Button("Default") { stored = "" }
            }
        }
    }
}

// Speech bubbles, the way Messages draws them: the person's message in the
// system's accent colour, the model's in the system's secondary fill, both
// changeable in Settings.
//
// A bubble is a filled shape behind the text, which is drawing of the client's
// own; this file is the no-styling rule's second named exception (the first is
// the text-effects renderer), kept one file wide, and the architecture test
// that holds the rule names it. Every default is a colour the system supplies,
// and every colour drawn here follows the appearance: a default because it is
// semantic, a picked colour because it is given a face per appearance, and the
// text on a bubble because it is the system's label colour read in whichever
// appearance the fill under it reads as.

import SwiftUI

/// The two bubble colours, stored as hex so a picker's choice survives a
/// relaunch; empty means the default.
struct BubbleColors {
    @AppStorage("bubbleColorUser") var user: String = ""
    @AppStorage("bubbleColorModel") var model: String = ""

    /// Semantic system colours: the accent the person chose for this Mac, and
    /// the system's secondary fill. Both resolve themselves in Light and in
    /// Dark, so a default bubble needs no adapting of ours.
    static let defaultUser = Color.accentColor
    static let defaultModel = Color.secondary

    var userColor: Color { Color(hex: user)?.appearanceAware ?? Self.defaultUser }
    var modelColor: Color { Color(hex: model)?.appearanceAware ?? Self.defaultModel }
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

    /// A stored colour drawn the way a system colour is: one face per
    /// appearance. The hue and the saturation are the person's pick, untouched;
    /// only the brightness moves, and only as far as it must to keep the bubble
    /// apart from the window it is drawn on — a pick that already reads there
    /// comes back exactly as it was chosen. This is why a colour can be kept as
    /// a single hex and still follow Light and Dark.
    var appearanceAware: Color {
        let (h, s, level, a) = hsb
        #if os(macOS)
        return Color(nsColor: NSColor(name: nil, dynamicProvider: { appearance in
            let dark = appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua
            let (r, g, b) = Color.srgb(h: h, s: s, b: Color.readable(level, saturation: s, inDark: dark))
            return NSColor(srgbRed: CGFloat(r), green: CGFloat(g), blue: CGFloat(b), alpha: CGFloat(a))
        }))
        #else
        return Color(uiColor: UIColor { traits in
            let dark = traits.userInterfaceStyle == .dark
            let (r, g, b) = Color.srgb(h: h, s: s, b: Color.readable(level, saturation: s, inDark: dark))
            return UIColor(red: CGFloat(r), green: CGFloat(g), blue: CGFloat(b), alpha: CGFloat(a))
        })
        #endif
    }

    /// The colour as hue, saturation, brightness and alpha: the form in which
    /// a brightness can be moved without disturbing the colour that was picked.
    private var hsb: (h: Double, s: Double, b: Double, a: Double) {
        let (r, g, b, a) = components
        let high = max(r, max(g, b)), low = min(r, min(g, b))
        let spread = high - low
        var hue: CGFloat = 0
        if spread > 0 {
            switch high {
            case r: hue = (g - b) / spread + (g < b ? 6 : 0)
            case g: hue = (b - r) / spread + 2
            default: hue = (r - g) / spread + 4
            }
            hue /= 6
        }
        return (Double(hue), high == 0 ? 0 : Double(spread / high), Double(high), Double(a))
    }

    /// The sRGB triple for a hue, a saturation and a brightness.
    nonisolated static func srgb(h: Double, s: Double, b: Double) -> (Double, Double, Double) {
        let sector = (h * 6).truncatingRemainder(dividingBy: 6)
        let index = Int(sector), fraction = sector - Double(index)
        let p = b * (1 - s), q = b * (1 - fraction * s), t = b * (1 - (1 - fraction) * s)
        switch index {
        case 0: return (b, t, p)
        case 1: return (q, b, p)
        case 2: return (p, b, t)
        case 3: return (p, q, b)
        case 4: return (t, p, b)
        default: return (b, p, q)
        }
    }

    /// A picked brightness moved only as far as it must to stay apart from the
    /// window it is drawn on: in Dark not so black that it sinks into the
    /// window, in Light not so white that it disappears into it. A saturated
    /// colour is never taken for the window, so the bands close on grey alone
    /// and a picked colour that reads in both appearances is left as picked.
    nonisolated static func readable(_ level: Double, saturation: Double, inDark dark: Bool) -> Double {
        let grey = 1 - saturation
        return dark ? max(level, 0.22 * grey) : min(level, 1 - 0.10 * grey)
    }
}

extension Color.Resolved {
    /// Whether the system's Dark-appearance label reads on this fill: the
    /// standard sRGB luminance, against the mid point every system control
    /// uses to make the same call.
    var isDark: Bool {
        0.299 * Double(red) + 0.587 * Double(green) + 0.114 * Double(blue) < 0.6
    }
}

/// A Messages-style bubble: a continuous rounded rectangle in the speaker's
/// colour behind the text, with the text in the colour that reads on it.
struct BubbleModifier: ViewModifier {
    @Environment(\.self) private var environment
    let color: Color
    let isUser: Bool

    func body(content: Content) -> some View {
        // The text is the system's own label colour throughout — never a fixed
        // white — read in the appearance the fill under it reads as: a dark
        // bubble is a Dark surface whatever the window is, and a light one a
        // Light surface. The model's bubble is drawn at a quarter's opacity
        // over the window, so its text stays in the window's own appearance.
        let fill = color.resolve(in: environment)
        let surface: ColorScheme = isUser ? (fill.isDark ? .dark : .light) : environment.colorScheme
        return content
            .foregroundStyle(Color.primary)
            .environment(\.colorScheme, surface)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(color.opacity(isUser ? 1 : 0.28), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
    }
}

extension View {
    func bubble(_ color: Color, isUser: Bool) -> some View {
        modifier(BubbleModifier(color: color, isUser: isUser))
    }
}

/// A colour picker bound to a stored bubble colour, with a way back to the
/// default. The swatch shows the value as it was picked; the bubble draws that
/// value through its face for the appearance the window is in.
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

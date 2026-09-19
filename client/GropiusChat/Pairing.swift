import Foundation
import Security
import CryptoKit

// MARK: - Pairing
//
// Pairing gives this client an identity of its own on one Gropius server
// (adr-2609182357322050). The client makes a keypair the private half of which
// never leaves the device, the server signs it a certificate, and from then on
// every request proves itself with that key over TLS. The API key is never
// typed on a paired client.
//
// Three things here are load-bearing and were measured before they were
// written (the spike of 2026-09-19, recorded in the shipping spec):
//
//  1. `SecKeyCopyExternalRepresentation` returns an X9.63 point, NOT a
//     SubjectPublicKeyInfo. The server hashes the SPKI, so the fixed 26-byte
//     P-256 header is prepended on this side. Without it pairing appears to
//     succeed and every later connection fails.
//  2. The identity is found by `kSecAttrApplicationLabel` — the key's own
//     public-key hash — and never by a label we chose: the macOS file keychain
//     overwrites a certificate's `kSecAttrLabel` with its common name, so a
//     chosen label does not survive `SecItemAdd`.
//  3. The Secure Enclave is attempted and not assumed. Under the signature
//     this app ships with, `SecKeyCreateRandomKey` with
//     `kSecAttrTokenIDSecureEnclave` fails with `errSecMissingEntitlement`
//     (-34018), so the fallback is a permanent Keychain key. A build signed
//     with a provisioning profile takes the Enclave path without any other
//     change.

/// The fixed DER header of a P-256 SubjectPublicKeyInfo, which turns the X9.63
/// point the Security framework hands back into the structure the server hashes.
nonisolated private let p256SPKIHeader: [UInt8] = [
    0x30, 0x59, 0x30, 0x13, 0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02,
    0x01, 0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07, 0x03,
    0x42, 0x00,
]

/// `base64(sha256(DER SubjectPublicKeyInfo))` — RFC 7469, and the one
/// identifier the client and the server both name a key by.
nonisolated func spkiFingerprint(_ spki: Data) -> String {
    Data(SHA256.hash(data: spki)).base64EncodedString()
}

/// The SubjectPublicKeyInfo of a public key the Security framework holds.
nonisolated func spkiOf(_ key: SecKey) -> Data? {
    guard let point = SecKeyCopyExternalRepresentation(key, nil) as Data? else { return nil }
    var spki = Data(p256SPKIHeader)
    spki.append(point)
    return spki
}

/// What this client knows about a server it has paired with.
///
/// The pinned fingerprint is the one the TLS handshake presented, never the one
/// the pairing answer or a Bonjour record carried: those arrive over channels
/// anything on the network can answer on, and a pin taken from one pins
/// whatever the loudest answer said.
struct PairedServer: Codable, Equatable {
    /// The plain origin this client paired through, which is how a stored
    /// pairing is matched to the server in front of the user.
    var origin: String
    /// The port paired requests go to.
    var tlsPort: Int
    /// The server's key, as this client saw it at a handshake it made.
    var pinnedSPKI: String
    /// The name this client gave itself, shown beside the fingerprint so it can
    /// be found on the server's own list.
    var name: String
    /// This client's own key, so its row on the panel can be recognised.
    var clientSPKI: String

    /// The base this client's requests go to once paired.
    var httpsBase: String? {
        guard let url = URL(string: origin), let host = url.host, tlsPort > 0 else { return nil }
        return "https://\(host):\(tlsPort)"
    }
}

/// The Keychain items pairing owns: the private key, the server-minted
/// certificate, and the record of what was paired.
///
/// The record is in the Keychain rather than in `UserDefaults` for the reason
/// the API key already is: a preferences plist is rewritable by anything
/// running as the user, and a pin anything can rewrite is not a pin.
enum PairingStore {
    private static let service = "dev.gropius.chat"
    private static let account = "pairedServer"
    nonisolated private static let keyTag = Data("dev.gropius.chat.pairing".utf8)

    /// Whether the key this client holds is in the Secure Enclave.
    ///
    /// Recorded and reported rather than assumed: an app that says
    /// "hardware-backed" when it silently fell back is wrong in the one place
    /// this feature sells itself.
    private(set) static var keyIsInSecureEnclave = false

    // MARK: the record

    static func read() -> PairedServer? {
        var query = baseQuery
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var out: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &out) == errSecSuccess,
              let data = out as? Data,
              let paired = try? JSONDecoder().decode(PairedServer.self, from: data)
        else { return nil }
        return paired
    }

    static func write(_ paired: PairedServer) {
        guard let data = try? JSONEncoder().encode(paired) else { return }
        let attrs: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        if SecItemUpdate(baseQuery as CFDictionary, attrs as CFDictionary) == errSecItemNotFound {
            var add = baseQuery
            add.merge(attrs) { _, new in new }
            SecItemAdd(add as CFDictionary, nil)
        }
    }

    /// Forget a pairing: the record, the certificate and the key.
    static func forget() {
        SecItemDelete(baseQuery as CFDictionary)
        SecItemDelete([
            kSecClass as String: kSecClassIdentity,
            kSecAttrApplicationTag as String: keyTag,
        ] as CFDictionary)
        SecItemDelete([
            kSecClass as String: kSecClassKey,
            kSecAttrApplicationTag as String: keyTag,
        ] as CFDictionary)
    }

    private static var baseQuery: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
    }

    // MARK: the keypair

    /// Make this client's keypair, attempting the Secure Enclave first.
    ///
    /// One code path and two attempt dictionaries, so a build signed with a
    /// provisioning profile takes the Enclave without a second design and this
    /// one still works without it.
    static func makeKey() -> SecKey? {
        // Start clean: a key left from an earlier pairing would be found by the
        // identity query and presented instead of the new one.
        SecItemDelete([kSecClass as String: kSecClassKey, kSecAttrApplicationTag as String: keyTag] as CFDictionary)

        var privateAttrs: [String: Any] = [
            kSecAttrIsPermanent as String: true,
            kSecAttrApplicationTag as String: keyTag,
        ]
        if let access = SecAccessControlCreateWithFlags(
            nil, kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly, [.privateKeyUsage], nil) {
            // .privateKeyUsage alone, deliberately. A presence flag would put a
            // biometric prompt inside a TLS handshake running on URLSession's
            // own queue, with no reason string and nowhere to draw it.
            privateAttrs[kSecAttrAccessControl as String] = access
            let enclave: [String: Any] = [
                kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
                kSecAttrKeySizeInBits as String: 256,
                kSecAttrTokenID as String: kSecAttrTokenIDSecureEnclave,
                kSecPrivateKeyAttrs as String: privateAttrs,
            ]
            if let key = SecKeyCreateRandomKey(enclave as CFDictionary, nil) {
                keyIsInSecureEnclave = true
                return key
            }
        }
        keyIsInSecureEnclave = false
        privateAttrs.removeValue(forKey: kSecAttrAccessControl as String)
        let ordinary: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits as String: 256,
            kSecPrivateKeyAttrs as String: privateAttrs,
        ]
        return SecKeyCreateRandomKey(ordinary as CFDictionary, nil)
    }

    /// Store the certificate the server minted so the Keychain can form the
    /// identity from it and the key already there.
    static func store(leaf der: Data) -> Bool {
        guard let cert = SecCertificateCreateWithData(nil, der as CFData) else { return false }
        let status = SecItemAdd([
            kSecClass as String: kSecClassCertificate,
            kSecValueRef as String: cert,
        ] as CFDictionary, nil)
        return status == errSecSuccess || status == errSecDuplicateItem
    }

    /// This client's identity, if it has one.
    ///
    /// Found through the key's own `kSecAttrApplicationLabel`, which is the
    /// hash of its public key: a label we chose does not survive a certificate
    /// add on the macOS file keychain, where the common name replaces it.
    nonisolated static func identity() -> SecIdentity? {
        var attrs: CFTypeRef?
        guard SecItemCopyMatching([
            kSecClass as String: kSecClassKey,
            kSecAttrApplicationTag as String: keyTag,
            kSecReturnAttributes as String: true,
        ] as CFDictionary, &attrs) == errSecSuccess,
            let applicationLabel = (attrs as? [String: Any])?[kSecAttrApplicationLabel as String] as? Data
        else { return nil }

        var out: CFTypeRef?
        guard SecItemCopyMatching([
            kSecClass as String: kSecClassIdentity,
            kSecAttrApplicationLabel as String: applicationLabel,
            kSecReturnRef as String: true,
        ] as CFDictionary, &out) == errSecSuccess, let found = out else { return nil }
        return (found as! SecIdentity)
    }
}

/// The URLSession delegate that makes a paired connection what it claims to be.
///
/// Two challenges, and the first is the one App Transport Security cannot
/// answer. ATS pinning cannot LOOSEN trust, so `NSPinnedDomains` can never
/// accept a self-signed leaf; the delegate route is the sanctioned one, and the
/// SPKI comparison is the gate. The system's own trust evaluation is never
/// consulted — it would refuse this certificate in any case, and a design that
/// asked it and then ignored the answer would be a check nobody could read.
/// The two fields are read and written from URLSession's own queue as well as
/// from the app, so they are behind a lock rather than on an actor: the
/// delegate has to answer synchronously, and a hop to the main actor inside a
/// TLS handshake is a hang waiting for a busy app.
nonisolated final class PinningDelegate: NSObject, URLSessionDelegate, @unchecked Sendable {
    private let lock = NSLock()
    private var _pinnedSPKI: String?
    private var _lastPresentedSPKI: String?

    /// What this client pins, or nil while it is learning it — which is the one
    /// moment of trust in this design, and the moment the docs page is about.
    var pinnedSPKI: String? {
        get { lock.withLock { _pinnedSPKI } }
        set { lock.withLock { _pinnedSPKI = newValue } }
    }

    /// The fingerprint the last handshake actually presented, so pairing can
    /// record it and the user can compare it with the panel.
    var lastPresentedSPKI: String? { lock.withLock { _lastPresentedSPKI } }

    func urlSession(
        _ session: URLSession,
        didReceive challenge: URLAuthenticationChallenge,
        completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
    ) {
        switch challenge.protectionSpace.authenticationMethod {
        case NSURLAuthenticationMethodServerTrust:
            guard let trust = challenge.protectionSpace.serverTrust,
                  let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate],
                  // The far end chooses what it sends, including nothing.
                  let leaf = chain.first,
                  let key = SecCertificateCopyKey(leaf),
                  let spki = spkiOf(key)
            else {
                completionHandler(.cancelAuthenticationChallenge, nil)
                return
            }
            let presented = spkiFingerprint(spki)
            lock.withLock { _lastPresentedSPKI = presented }
            guard let pinned = pinnedSPKI else {
                // Pairing: this handshake is where the pin comes from.
                completionHandler(.useCredential, URLCredential(trust: trust))
                return
            }
            // A certificate that is not the pinned one is a refusal with a
            // reason, and never an offer to pair again: re-pairing on a
            // mismatch is the attacker's own recovery path.
            if presented == pinned {
                completionHandler(.useCredential, URLCredential(trust: trust))
            } else {
                completionHandler(.cancelAuthenticationChallenge, nil)
            }

        case NSURLAuthenticationMethodClientCertificate:
            guard let identity = PairingStore.identity() else {
                completionHandler(.cancelAuthenticationChallenge, nil)
                return
            }
            completionHandler(.useCredential,
                              URLCredential(identity: identity, certificates: nil, persistence: .forSession))

        default:
            completionHandler(.performDefaultHandling, nil)
        }
    }
}

/// What the pairing call sends and what comes back.
struct PairRequest: Encodable {
    let name: String
    let public_key: String
}

struct PairAnswer: Decodable {
    let leaf: String
    /// The server's own fingerprint, FOR DISPLAY. It arrives over the plain
    /// port, so it is not what this client pins.
    let fingerprint: String
    let tls_port: Int
    let name: String
}

enum PairingError: LocalizedError {
    case noKey
    case badAnswer
    case refused(String)
    case noHandshake

    var errorDescription: String? {
        switch self {
        case .noKey: return "This device would not make a key for pairing."
        case .badAnswer: return "That server answered the pairing request with something this client cannot use."
        case .refused(let why): return why
        case .noHandshake: return "This client could not reach that server over its secure port."
        }
    }
}

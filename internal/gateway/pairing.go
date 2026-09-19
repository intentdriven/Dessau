package gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/pairing"
	"github.com/intentdriven/Gropius/internal/runtime"
)

// maxPairBodyBytes caps the pairing request. A name and a P-256 public key in
// base64 are about 250 bytes; the endpoint is unauthenticated by design, so the
// bound is on the reader and not on a length header a caller writes.
const maxPairBodyBytes = 4 << 10

// pairRequest is what a client sends to pair. The public key is the raw DER
// SubjectPublicKeyInfo, base64-encoded — on the client's side that is the
// X9.63 point the Security framework hands back with the fixed 26-byte P-256
// header prepended, which is the one place the two languages have to agree.
type pairRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// pairAnswer is what comes back: the client's own certificate, and the
// server's fingerprint FOR DISPLAY ONLY.
//
// The fingerprint here is not what the client pins, and it must not be. It
// arrives over the plain port, where anything on the network can answer, so a
// pin taken from it pins whatever the loudest answer said — and pinning an
// attacker's key is not a detectable failure, it is the pin working against the
// wrong key. The client pins the certificate presented at a TLS handshake it
// made itself, and shows this value so it can be compared with the one on the
// panel. That comparison is the check.
type pairAnswer struct {
	Leaf        string `json:"leaf"`
	Fingerprint string `json:"fingerprint"`
	TLSPort     int    `json:"tls_port"`
	Name        string `json:"name"`
	// clientSPKI is the pairing's own fingerprint, for the log line. Unexported
	// so it does not travel: the client computed it and does not need it back.
	clientSPKI string
	// unchanged says the stored row would be identical, so there is nothing to
	// save. Unexported for the same reason.
	unchanged bool
}

// PairHandler is POST /pair, and it is mounted on the PLAIN listener only.
//
// It has to be: a client that has not paired has no certificate, and the TLS
// listener refuses a handshake without one. Nothing secret crosses here — the
// client's PUBLIC key goes out and a certificate comes back, and a certificate
// is worthless without the private key that never leaves the device. What an
// attacker on the network gets out of watching is nothing; what they get out of
// ANSWERING is the first-come window this design accepts and the docs page
// states (adr-2609182357322050, decision 4).
func (c *Control) PairHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "pairing is a POST")
			return
		}
		// Everything the request carries is read and checked BEFORE the lock is
		// taken. The body arrives over a socket this server does not control,
		// and the plain listener bounds the headers and not the body — so
		// decoding under the lock let one connection that sent headers and then
		// stalled wedge every future pairing and every settings save the
		// operator makes, for as long as it stayed open (iss-2609190110110353).
		req, err := readPairRequest(r)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errPairBodyTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			writeError(w, status, err.Error())
			return
		}
		// The same lock the settings handler takes, in the same order
		// (adr-2609091239058072, rule 1): pairing is a read of the settings in
		// force, a change to a COPY and a SetConfig, which is exactly the shape
		// that lock serialises. Taking it here is what stops a pairing and a
		// save from losing each other's work.
		c.settingsMu.Lock()
		defer c.settingsMu.Unlock()
		// Clone, and the reason is not tidiness. App.Config() returns the
		// configuration by value, and a struct copy SHARES its maps — so
		// writing cfg.Clients wrote the running paired set, which the registry
		// reads on every handshake and every request and the panel iterates on
		// every poll. A Go map is not concurrency-safe and that is a fatal
		// error rather than a race report, reachable from an endpoint nobody
		// has to authenticate to (iss-2609190110117982). Cloning also keeps a
		// pairing the save then refuses out of the live set, so a 500 here is
		// not silent admission (iss-2609190110110994).
		cfg := c.App.Config().Clone()
		answer, ok := c.pairInto(&cfg, req, w)
		if !ok {
			return
		}
		if answer.unchanged {
			// Nothing to write. Without this an unauthenticated endpoint is a
			// remote fsync loop on the operator's settings file, since a client
			// re-pairing under a key it already holds skips the ceiling too
			// (iss-2609190110241925).
			writeJSON(w, http.StatusOK, answer)
			return
		}
		// The answer is written only once the pairing is stored. A client takes
		// the certificate in it as proof that it is paired, so answering before
		// the save would hand it one for a pairing this server does not hold —
		// and every request it then made would be refused as unpaired, with
		// nothing on either side to explain it.
		if err := c.App.SetConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "this server could not record the pairing")
			return
		}
		if c.Log != nil {
			c.Log.Info("client paired", "client", answer.Name, "fingerprint", shortFingerprint(answer.clientSPKI))
		}
		writeJSON(w, http.StatusOK, answer)
	})
}

// readPairRequest reads and shapes the request, and nothing else.
//
// Two guards here are about the CALLER rather than the content, and they are
// what keeps this from being a form a web page can post. A POST carrying a
// CORS-simple content type sends no preflight, so without them any page
// somebody on this network visits could enrol a key of the attacker's choosing
// — and the attacker would never need to read the answer, because what admits
// a client is the key and they can sign their own certificate for it. That is
// a wider window than adr-2609182357322050 accepted, which is hosts on the
// network rather than every website anybody on it visits
// (iss-2609190110118690).
func readPairRequest(r *http.Request) (pairRequest, error) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		return pairRequest{}, errors.New(`pairing is posted as "application/json"`)
	}
	if r.Header.Get("Origin") != "" {
		return pairRequest{}, errors.New("pairing is not something a web page may ask for")
	}
	// One byte past the bound, so that reaching it is a fact about the REQUEST
	// and not about how much this server felt like reading. A streaming decoder
	// over a LimitReader stops at the first complete JSON value, so a body whose
	// opening bytes are a well-formed pairing and whose remainder is anything at
	// all decoded and paired — the read was bounded, the acceptance was not
	// (iss-2609190200099532). Unmarshal over the whole body also refuses trailing
	// content INSIDE the bound, which is the same fault at a smaller size.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPairBodyBytes+1))
	if err != nil {
		return pairRequest{}, errors.New("the pairing request could not be read")
	}
	if len(body) > maxPairBodyBytes {
		return pairRequest{}, errPairBodyTooLarge
	}
	var req pairRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return pairRequest{}, errors.New("the pairing request is not JSON this server reads")
	}
	return req, nil
}

// errPairBodyTooLarge is the one refusal from readPairRequest that is not a
// 400: the caller is told the size is the problem, because a client that is
// told "not JSON" about a body that is perfectly good JSON has nothing to act
// on.
var errPairBodyTooLarge = errors.New("the pairing request is larger than this server accepts")

// pairInto validates a pairing request and writes it into cfg, returning the
// answer for the caller to send once the save has succeeded.
//
// It refuses into w and it never ANSWERS into w: a refusal is final, while an
// answer depends on a save this function does not make. It is separate from
// the saving so that everything it refuses is testable without a whole App
// behind it, and so that nothing malformed ever reaches the settings file.
//
// cfg must be a copy. Writing the running configuration's map is the fault
// iss-2609190110117982 records.
func (c *Control) pairInto(cfg *config.Config, req pairRequest, w http.ResponseWriter) (pairAnswer, bool) {
	if err := config.ValidClientName(req.Name); err != nil {
		writeError(w, http.StatusBadRequest, "that name is not one this server will record: "+err.Error())
		return pairAnswer{}, false
	}
	der, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "the public key is not base64")
		return pairAnswer{}, false
	}
	pub, err := pairing.PublicKeyFromSPKI(der)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return pairAnswer{}, false
	}
	fingerprint := pairing.Fingerprint(der)
	if _, already := cfg.Clients[fingerprint]; !already && len(cfg.Clients) >= config.MaxClients {
		// The ceiling is what stops an endpoint nobody has to authenticate to
		// from growing config.json until the next start cannot read it.
		writeError(w, http.StatusConflict,
			"this server is already paired with as many clients as it holds; revoke one on the control panel first")
		return pairAnswer{}, false
	}
	if c.Identity == nil {
		writeError(w, http.StatusServiceUnavailable, "this server has no certificate of its own, so it cannot pair")
		return pairAnswer{}, false
	}
	leaf, err := c.Identity.MintLeaf(pub, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "this server could not sign a certificate for that key")
		return pairAnswer{}, false
	}
	if cfg.Clients == nil {
		cfg.Clients = map[string]config.Client{}
	}
	pairedAt := time.Now().Unix()
	unchanged := false
	if was, already := cfg.Clients[fingerprint]; already {
		// The same key pairing again is the same client, renamed or reinstalled
		// — not a second one nobody can tell apart on the panel.
		pairedAt = was.PairedAt
		unchanged = was.Name == req.Name
	}
	cfg.Clients[fingerprint] = config.Client{Name: req.Name, SPKI: fingerprint, PairedAt: pairedAt}
	return pairAnswer{
		unchanged:   unchanged,
		Leaf:        base64.StdEncoding.EncodeToString(leaf),
		Fingerprint: c.Identity.Fingerprint(),
		TLSPort:     cfg.EffectiveTLSPort(),
		Name:        req.Name,
		clientSPKI:  fingerprint,
	}, true
}

// handleRevoke takes a client off the paired set. Its own route rather than a
// settings save: a settings body that could write this map would make the
// loopback-only settings endpoint a second pairing endpoint, asking for no
// credential, through which any other account on this Mac could write itself
// into the list of who may connect.
func (c *Control) handleRevoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxPairBodyBytes)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "that is not a revocation this server reads")
		return
	}
	c.settingsMu.Lock()
	defer c.settingsMu.Unlock()
	// Cloned for the reason PairHandler clones: App.Config() shares its maps
	// with the running configuration, and deleting from it would delete from
	// under the registry and the panel (iss-2609190110117982).
	cfg := c.App.Config().Clone()
	was, ok := cfg.Clients[req.Fingerprint]
	if !ok {
		writeError(w, http.StatusNotFound, "no client is paired under that fingerprint")
		return
	}
	delete(cfg.Clients, req.Fingerprint)
	if len(cfg.Clients) == 0 {
		cfg.Clients = nil
	}
	if err := c.App.SetConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "this server could not record the revocation")
		return
	}
	if c.Clients != nil {
		c.Clients.Forget(req.Fingerprint)
	}
	if c.Log != nil {
		c.Log.Info("client revoked", "client", was.Name, "fingerprint", shortFingerprint(req.Fingerprint))
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "revoked"})
}

// PairedClient is one row of the Clients pane.
//
// LastSeen is a Unix second and zero means "not since this server started",
// which is the honest answer rather than a time nobody recorded: the sighting
// is held in memory, because writing it would fsync the settings file once per
// request.
type PairedClient struct {
	Name        string `json:"name"`
	SPKI        string `json:"spki"`
	PairedAt    int64  `json:"paired_at"`
	LastSeen    int64  `json:"last_seen,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// pairedClients is the paired set as the panel reads it, in one order, with the
// sightings this run has folded in.
func pairedClients(cfg config.Config, reg *pairing.Registry) []PairedClient {
	out := make([]PairedClient, 0, len(cfg.Clients))
	for _, id := range cfg.ClientIDs() {
		c := cfg.Clients[id]
		row := PairedClient{Name: c.Name, SPKI: c.SPKI, PairedAt: c.PairedAt, Fingerprint: shortFingerprint(c.SPKI)}
		if reg != nil {
			if t, ok := reg.LastSeen(id); ok {
				row.LastSeen = t.Unix()
			}
		}
		out = append(out, row)
	}
	return out
}

// shortFingerprint is the first eight characters, for a log line and for a
// column narrow enough to read. Two clients may choose one name — that is
// precisely what an impostor does — so the name alone can never be what
// tells a reader which client they are looking at.
func shortFingerprint(spki string) string {
	if len(spki) <= 8 {
		return spki
	}
	return spki[:8]
}

// pairedOnly admits a request that arrived over the TLS listener under a key
// this server has paired, and refuses every other.
//
// This is the whole of the API-key exemption for a paired client, and it is
// keyed on the presented certificate's key and on NOTHING else — never a
// header, never the remote address, never a context value the plain handler
// could also carry. The plain port's withAuth is not touched by any of this.
//
// It is also where "refused on its next request" is made true. The handshake
// checks too, in VerifyConnection, but every handshake-level hook is per
// connection: a keep-alive or an HTTP/2 stream outlives it for as long as the
// client keeps the socket open, which a chat client does because it streams.
func pairedOnly(reg *pairing.Registry, next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			writeError(w, http.StatusForbidden, "this port serves paired clients over TLS")
			return
		}
		spki, ok := pairing.PeerFingerprint(r.TLS.PeerCertificates)
		if !ok {
			writeError(w, http.StatusForbidden, errors.New("this client presented no certificate").Error())
			return
		}
		client, paired := reg.Sight(spki)
		if !paired {
			writeError(w, http.StatusForbidden, pairing.ErrNotPaired.Error())
			return
		}
		if log != nil {
			log.Debug("paired request", "client", client.Name, "fingerprint", shortFingerprint(spki),
				"method", r.Method, "path", r.URL.Path)
		}
		// A paired client is admitted the way a keyed one is: it has proved
		// itself, so it is told what a keyed client is told, and it gets its
		// own place in the pool's queue rather than sharing the anonymous one.
		r = withAdmittedKeyed(r, true)
		next.ServeHTTP(w, r.WithContext(runtime.WithSource(r.Context(), "client:"+spki)))
	})
}

package hub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// checkHub is a fake Hub for the update check: one repository, whose commit
// endpoint answers anonymously or only with a token, and which records the
// Authorization header of every request.
type checkHub struct {
	commit  string
	private bool // refuse an anonymous request with 401
	files   []File
	small   map[string]string
	ratelim string // the RateLimit header to send, if any

	mu    sync.Mutex
	auths []string
	paths []string
}

func (h *checkHub) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.auths = append(h.auths, r.Header.Get("Authorization"))
		h.paths = append(h.paths, r.URL.Path)
		h.mu.Unlock()
		if h.ratelim != "" {
			w.Header().Set("RateLimit", h.ratelim)
		}
		if h.private && r.Header.Get("Authorization") == "" {
			http.Error(w, `{"error":"Repository not found"}`, http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/models/org/repo":
			fmt.Fprintf(w, `{"id":"org/repo","sha":%q}`, h.commit)
		case r.URL.Path == "/api/models/org/repo/tree/"+h.commit:
			json.NewEncoder(w).Encode(h.files)
		case strings.HasPrefix(r.URL.Path, "/org/repo/resolve/"+h.commit+"/"):
			body, ok := h.small[strings.TrimPrefix(r.URL.Path, "/org/repo/resolve/"+h.commit+"/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (h *checkHub) authorizations() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.auths...)
}

// A check sends no token to a public repository, whatever Settings holds: the
// token ties the list of models on this Mac to the operator's account.
func TestACheckOfAPublicRepositorySendsNoToken(t *testing.T) {
	h := &checkHub{commit: testCommit, files: []File{{Path: "config.json", OID: testCommit}}}
	srv := h.server(t)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.SetToken("hf_secret")

	up, err := c.Latest(context.Background(), "org/repo")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if up.Commit != testCommit {
		t.Errorf("Commit = %q", up.Commit)
	}
	if _, err := c.FilesAt(context.Background(), up); err != nil {
		t.Fatalf("FilesAt: %v", err)
	}
	for i, a := range h.authorizations() {
		if a != "" {
			t.Errorf("request %d carried %q to a public repository", i, a)
		}
	}
}

// A private or gated repository refuses the anonymous request; only then is
// the request made again with the token, and what follows for that
// repository carries it too.
func TestACheckSendsTheTokenOnlyAfterARefusal(t *testing.T) {
	h := &checkHub{commit: testCommit, private: true, files: []File{{Path: "config.json"}}}
	srv := h.server(t)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.SetToken("hf_secret")

	up, err := c.Latest(context.Background(), "org/repo")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if _, err := c.FilesAt(context.Background(), up); err != nil {
		t.Fatalf("FilesAt: %v", err)
	}
	got := h.authorizations()
	if len(got) != 3 || got[0] != "" || got[1] != "Bearer hf_secret" || got[2] != "Bearer hf_secret" {
		t.Errorf("authorizations = %q, want an anonymous request, then the token for it and for the listing", got)
	}
}

// With no token in Settings a refused repository is one the check cannot
// read, and it says so without a second request.
func TestACheckWithNoTokenStopsAtTheRefusal(t *testing.T) {
	h := &checkHub{commit: testCommit, private: true}
	srv := h.server(t)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}

	_, err := c.Latest(context.Background(), "org/repo")
	if !IsAuthRequired(err) {
		t.Fatalf("err = %v, want the refusal", err)
	}
	if n := len(h.authorizations()); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

// A missing repository answers an anonymous request with 401, the same as a
// gated one, so the refusal must not send an operator to add a token that
// cannot help: anonymously it says missing, private or gated; with a token
// sent it says access was refused (iss-2610030913177383).
func TestARefusalSaysWhatItMeans(t *testing.T) {
	anon := (&APIError{StatusCode: http.StatusUnauthorized, URL: "u"}).Error()
	for _, want := range []string{"missing", "private", "gated"} {
		if !strings.Contains(anon, want) {
			t.Errorf("an anonymous 401 says %q, which does not say %q", anon, want)
		}
	}
	if strings.Contains(anon, "may be gated; add an access token") {
		t.Errorf("an anonymous 401 still says only that the repo may be gated: %q", anon)
	}
	authed := (&APIError{StatusCode: http.StatusUnauthorized, URL: "u", TokenSent: true}).Error()
	if !strings.Contains(authed, "refused") || !strings.Contains(authed, "token") {
		t.Errorf("a 401 to a request with a token says %q, want that the token was refused", authed)
	}

	// And the flag is set from the request that was actually sent.
	h := &checkHub{commit: testCommit, private: true}
	srv := h.server(t)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Latest(context.Background(), "org/repo")
	var ae *APIError
	if !errors.As(err, &ae) || ae.TokenSent {
		t.Errorf("an anonymous refusal reads as one with a token: %#v", err)
	}
	c.SetToken("hf_wrong")
	h.private = true
	h2 := &checkHub{commit: testCommit, private: true}
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h2.mu.Lock()
		h2.auths = append(h2.auths, r.Header.Get("Authorization"))
		h2.mu.Unlock()
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer srv2.Close()
	c2 := &Client{BaseURL: srv2.URL, HTTP: srv2.Client()}
	c2.SetToken("hf_wrong")
	_, err = c2.Latest(context.Background(), "org/repo")
	if !errors.As(err, &ae) || !ae.TokenSent {
		t.Errorf("a refusal of the token reads as an anonymous one: %#v", err)
	}
}

// The small file a check reads — the new config.json — is fetched at the
// commit, from the Hub's own origin, under the same token choice, and
// bounded.
func TestACheckReadsASmallFileAtTheCommit(t *testing.T) {
	h := &checkHub{commit: testCommit, small: map[string]string{"config.json": `{"model_file":"x.py"}`}}
	srv := h.server(t)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	up, err := c.Latest(context.Background(), "org/repo")
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.SmallFileAt(context.Background(), up, File{Path: "config.json"}, 1<<20)
	if err != nil {
		t.Fatalf("SmallFileAt: %v", err)
	}
	if string(b) != `{"model_file":"x.py"}` {
		t.Errorf("body = %q", b)
	}
	if _, err := c.SmallFileAt(context.Background(), up, File{Path: "config.json"}, 4); !errors.Is(err, ErrOversizedBody) {
		t.Errorf("a body past the bound: err = %v, want ErrOversizedBody", err)
	}
}

// lfsSmallFileHub is a Hub that hands config.json to a content CDN, as it
// does for a file a repository keeps in LFS; the CDN answers with body and
// records the Authorization header it was sent.
func lfsSmallFileHub(t *testing.T, body string) (c *Client, cdnAuth func() []string) {
	t.Helper()
	var mu sync.Mutex
	var auths []string
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auths = append(auths, r.Header.Get("Authorization"))
		mu.Unlock()
		fmt.Fprint(w, body)
	}))
	t.Cleanup(cdn.Close)
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/org/repo/resolve/"+testCommit+"/config.json" {
			http.Redirect(w, r, cdn.URL+"/blob", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(hub.Close)
	return &Client{BaseURL: hub.URL, HTTP: hub.Client()}, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), auths...)
	}
}

func lfsFile(path, body string, size int64) File {
	sum := sha256.Sum256([]byte(body))
	f := File{Path: path, Type: "file", Size: size}
	f.LFS = &struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	}{OID: hex.EncodeToString(sum[:]), Size: size}
	return f
}

// A small file the listing says the repository keeps in LFS is read where
// the Hub hands it, its content CDN, as a download reads it: held to the
// sha256 the listing gives and to the bound, and never sent the token off
// the Hub's origin (iss-2610042101439623).
func TestACheckReadsASmallLFSFileFromTheContentCDN(t *testing.T) {
	const body = `{"model_file":"x.py"}`
	c, cdnAuth := lfsSmallFileHub(t, body)
	up := Upstream{RepoID: "org/repo", Commit: testCommit, Authed: true, token: "hf_secret"}
	b, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", body, int64(len(body))), 1<<20)
	if err != nil {
		t.Fatalf("SmallFileAt: %v", err)
	}
	if string(b) != body {
		t.Errorf("body = %q", b)
	}
	for _, a := range cdnAuth() {
		if a != "" {
			t.Errorf("the content CDN was sent Authorization %q", a)
		}
	}
	if _, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", `{"model_type":"qwen3"}`, int64(len(body))), 1<<20); !errors.Is(err, ErrContentMismatch) {
		t.Errorf("bytes that are not the listed ones: err = %v, want ErrContentMismatch", err)
	}
	if _, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", body, int64(len(body))), 4); !errors.Is(err, ErrOversizedBody) {
		t.Errorf("a body past the bound: err = %v, want ErrOversizedBody", err)
	}
	before := len(cdnAuth())
	if _, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", body, 1<<30), 1<<20); !errors.Is(err, ErrOversizedBody) {
		t.Errorf("a file listed past the bound: err = %v, want ErrOversizedBody", err)
	}
	if len(cdnAuth()) != before {
		t.Error("a file listed past the bound was fetched")
	}
}

// A file the listing keeps in git has no sha256 to hold the CDN's bytes to,
// so one the Hub hands off its origin is still refused.
func TestASmallGitFileHandedOffTheHubIsRefused(t *testing.T) {
	c, cdnAuth := lfsSmallFileHub(t, `{}`)
	up := Upstream{RepoID: "org/repo", Commit: testCommit}
	_, err := c.SmallFileAt(context.Background(), up, File{Path: "config.json", OID: testCommit}, 1<<20)
	if !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want ErrCrossOrigin", err)
	}
	if len(cdnAuth()) != 0 {
		t.Error("the content CDN was asked for a file the listing keeps in git")
	}
}

// The Hub's rate-limit header is read off every answer, so a round of checks
// can stop before it spends what is left.
func TestTheRateLimitHeaderIsRead(t *testing.T) {
	h := &checkHub{commit: testCommit, ratelim: `"api";r=7;t=120`}
	srv := h.server(t)
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, ok := c.RateRemaining(); ok {
		t.Error("a budget is known before any answer")
	}
	if _, err := c.Latest(context.Background(), "org/repo"); err != nil {
		t.Fatal(err)
	}
	if n, ok := c.RateRemaining(); !ok || n != 7 {
		t.Errorf("RateRemaining = %d, %v; want 7", n, ok)
	}
	for header, want := range map[string]int{
		`"api";r=0;t=30`:                    0,
		`limit=500, remaining=42, reset=10`: 42,
	} {
		if got, ok := parseRateRemaining(http.Header{"Ratelimit": {header}}); !ok || got != want {
			t.Errorf("parseRateRemaining(%q) = %d, %v; want %d", header, got, ok, want)
		}
	}
	if got, ok := parseRateRemaining(http.Header{"X-Ratelimit-Remaining": {"5"}}); !ok || got != 5 {
		t.Errorf("X-RateLimit-Remaining read as %d, %v", got, ok)
	}
	if _, ok := parseRateRemaining(http.Header{"Ratelimit": {"nonsense"}}); ok {
		t.Error("a header with no remaining figure was read as one")
	}
}

// An Upstream formatted any way never spells out the token it carries.
func TestAnUpstreamNeverFormatsItsToken(t *testing.T) {
	up := Upstream{RepoID: "org/repo", Commit: testCommit, Authed: true, token: "hf_secret"}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		if got := fmt.Sprintf(verb, up); strings.Contains(got, "hf_secret") {
			t.Errorf("%s formats the token: %s", verb, got)
		}
	}
}

// Once a content fetch's redirect chain has left the Hub's origin, the token
// stays off every later hop, including one that returns to the Hub: a host
// off the origin chooses the next hop, and must not be able to pick an
// authenticated Hub request (iss-2610071200187548).
func TestAContentFetchSendsNoTokenBackToTheHubAfterLeavingIt(t *testing.T) {
	const body = `{"model_type":"qwen3"}`
	var mu sync.Mutex
	var cdnAuth, returnAuth []string
	var hubURL string
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		cdnAuth = append(cdnAuth, r.Header.Get("Authorization"))
		mu.Unlock()
		http.Redirect(w, r, hubURL+"/returned", http.StatusFound)
	}))
	t.Cleanup(cdn.Close)
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/org/repo/resolve/" + testCommit + "/config.json":
			http.Redirect(w, r, cdn.URL+"/blob", http.StatusFound)
		case "/returned":
			mu.Lock()
			returnAuth = append(returnAuth, r.Header.Get("Authorization"))
			mu.Unlock()
			fmt.Fprint(w, body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(hub.Close)
	hubURL = hub.URL
	c := &Client{BaseURL: hub.URL, HTTP: hub.Client()}
	up := Upstream{RepoID: "org/repo", Commit: testCommit, Authed: true, token: "hf_secret"}
	if _, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", body, int64(len(body))), 1<<20); err != nil {
		t.Fatalf("SmallFileAt: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(cdnAuth) != 1 || len(returnAuth) != 1 {
		t.Fatalf("hops made: CDN %d, back to the Hub %d; want 1 each", len(cdnAuth), len(returnAuth))
	}
	if cdnAuth[0] != "" {
		t.Errorf("the content CDN was sent Authorization %q", cdnAuth[0])
	}
	if returnAuth[0] != "" {
		t.Errorf("the hop back to the Hub after leaving it was sent Authorization %q", returnAuth[0])
	}
}

// A content fetch from an https Hub refuses a hop to plain http rather than
// carry a gated repository's file unencrypted (iss-2610071200187548).
func TestAContentFetchRefusesARedirectFromHTTPSToHTTP(t *testing.T) {
	const body = `{"model_type":"qwen3"}`
	var plainHits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plainHits.Add(1)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(plain.Close)
	hub := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/blob", http.StatusFound)
	}))
	t.Cleanup(hub.Close)
	c := &Client{BaseURL: hub.URL, HTTP: hub.Client()}
	up := Upstream{RepoID: "org/repo", Commit: testCommit}
	if _, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", body, int64(len(body))), 1<<20); err == nil {
		t.Error("SmallFileAt followed a redirect from https to http")
	}
	if n := plainHits.Load(); n != 0 {
		t.Errorf("the plain-http host was asked %d times", n)
	}
}

// A refusal from the content CDN names the CDN and gives no advice about the
// access token, which the CDN was never sent; one from the Hub's own origin
// keeps it (iss-2610071200183905).
func TestACDNRefusalIsNotWordedAsTheHubRefusingTheToken(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	t.Cleanup(cdn.Close)
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/org/repo/resolve/"+testCommit+"/config.json" {
			http.Redirect(w, r, cdn.URL+"/blob", http.StatusFound)
			return
		}
		http.Error(w, "denied", http.StatusForbidden)
	}))
	t.Cleanup(hub.Close)
	c := &Client{BaseURL: hub.URL, HTTP: hub.Client()}
	up := Upstream{RepoID: "org/repo", Commit: testCommit, Authed: true, token: "hf_secret"}

	_, err := c.SmallFileAt(context.Background(), up, lfsFile("config.json", "{}", 2), 1<<20)
	if err == nil {
		t.Fatal("a 403 from the content CDN was read as a file")
	}
	msg := err.Error()
	if strings.Contains(msg, "token") {
		t.Errorf("a CDN refusal gives token advice: %s", msg)
	}
	cdnHost := strings.TrimPrefix(cdn.URL, "http://")
	if !strings.Contains(msg, "content CDN") || !strings.Contains(msg, cdnHost) {
		t.Errorf("a CDN refusal does not name the CDN at %s: %s", cdnHost, msg)
	}

	_, err = c.SmallFileAt(context.Background(), up, File{Path: "other.json"}, 1<<20)
	if err == nil || !strings.Contains(err.Error(), "access token") {
		t.Errorf("a refusal from the Hub's origin lost its token advice: %v", err)
	}
}

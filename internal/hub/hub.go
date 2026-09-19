// Package hub is a pure-Go HuggingFace Hub client: search, file listing, and a
// resumable downloader.
//
// It deliberately does not reproduce huggingface_hub's blobs/snapshots/symlinks
// cache format. mlx-lm loads a model from any plain directory of files, so
// Gropius downloads each repo into <models>/<org>/<name>/ and hands that path
// to the server process. That keeps the on-disk result inspectable and means a
// half-finished download can never masquerade as a valid cache entry.
package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the public HuggingFace Hub.
const DefaultBaseURL = "https://huggingface.co"

// Client talks to the HuggingFace Hub HTTP API.
type Client struct {
	BaseURL string
	HTTP    *http.Client

	// mu guards token. The access token is written by whoever saves the
	// settings and read by every request this client builds — a search the
	// operator typed, and each of the hundreds a multi-file download issues —
	// so it is not a field a caller may touch directly. Nothing else in this
	// package takes a lock, so it orders against nothing.
	mu    sync.RWMutex
	token string
}

// SetToken sets the access token sent with every subsequent request. It is
// optional, and required only for gated repos.
//
// A download already in flight is not affected: Download pins the token it
// starts with and uses that one for the whole repo, so a token saved (or
// cleared) halfway through cannot turn the second half of a download into a
// run of 401s against a gated repo.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// Token is the access token currently in force.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// New returns a Client pointed at the public Hub.
func New() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP: &http.Client{
			// No overall timeout: model downloads legitimately take many
			// minutes. Cancellation comes from the caller's context. The
			// transport timeouts below cover only the handshake and response
			// headers; a dead connection mid-body surfaces as a read error
			// via TCP keepalives (the zero net.Dialer enables them), and a
			// live-but-silent peer is recovered by the user's Cancel, after
			// which the .part resumes on retry.
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				MaxIdleConnsPerHost:   8,
				ResponseHeaderTimeout: 30 * time.Second,
				TLSHandshakeTimeout:   15 * time.Second,
			},
		},
	}
}

// Model is a search result / repo summary.
type Model struct {
	ID           string    `json:"id"`
	Downloads    int       `json:"downloads"`
	Likes        int       `json:"likes"`
	Tags         []string  `json:"tags"`
	PipelineTag  string    `json:"pipeline_tag"`
	LastModified time.Time `json:"lastModified"`
}

// Org returns the account that owns the repo ("mlx-community/Foo" -> "mlx-community").
func (m Model) Org() string {
	if i := strings.Index(m.ID, "/"); i >= 0 {
		return m.ID[:i]
	}
	return ""
}

// Name returns the repo name without the owner.
func (m Model) Name() string {
	if i := strings.Index(m.ID, "/"); i >= 0 {
		return m.ID[i+1:]
	}
	return m.ID
}

// Quantization extracts the quant suffix from a model name ("...-4bit" -> "4bit").
// Returns "" when the name carries no recognizable quantization marker.
func (m Model) Quantization() string {
	name := strings.ToLower(m.Name())
	// Ordered longest-first so "-4bit-dwq" wins over "-4bit".
	for _, q := range []string{
		"8bit-dwq", "6bit-dwq", "4bit-dwq", "3bit-dwq",
		"8bit", "6bit", "5bit", "4bit", "3bit", "2bit",
		"bf16", "fp16", "float16", "fp32",
	} {
		if strings.HasSuffix(name, "-"+q) || strings.Contains(name, "-"+q+"-") {
			return q
		}
	}
	return ""
}

// IsMLX reports whether the Hub says this repo holds an MLX model, i.e. one
// this server could actually load.
//
// The Hub says it twice. A conversion that declares its library carries the
// "mlx" tag, which is the Hub's own answer and the reliable one. A conversion
// that declares nothing still names itself: the convention on the Hub is an
// "mlx" marker in the repo name, and reading that as the second answer is what
// lets a conversion published under somebody's own account be recognised at
// all. Both are words the Hub holds; neither is a guess about the weights, and
// a repo that says neither is not offered as an MLX model.
func (m Model) IsMLX() bool {
	for _, t := range m.Tags {
		if strings.EqualFold(t, "mlx") {
			return true
		}
	}
	name := strings.ToLower(m.Name())
	return strings.HasPrefix(name, "mlx-") ||
		strings.HasSuffix(name, "-mlx") ||
		strings.Contains(name, "-mlx-")
}

// File is one entry in a repo's file tree.
type File struct {
	Path string `json:"path"`
	// Type is "file" or "directory" in the Hub tree listing. Directory entries
	// must be dropped before download: they are not fetchable and a GET of one
	// 404s, which would abort the whole repo download.
	Type string `json:"type"`
	Size int64  `json:"size"`
	OID  string `json:"oid"`
	LFS  *struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	} `json:"lfs,omitempty"`
}

// SearchQuery parameterises a model search.
type SearchQuery struct {
	// Search is a free-text term matched against the repo name.
	Search string
	// Author restricts results to one org, e.g. "mlx-community".
	Author string
	// Limit caps the number of results (the Hub's own default is small).
	Limit int
	// Sort is a Hub sort key, e.g. "downloads", "lastModified", "likes".
	Sort string
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// newRequest builds a request carrying whatever token is in force right now.
// A caller that must not see the token change under it — a download, which
// issues one request per file — reads it once and calls newTokenRequest.
func (c *Client) newRequest(ctx context.Context, method, u string) (*http.Request, error) {
	return c.newTokenRequest(ctx, method, u, c.Token())
}

func (c *Client) newTokenRequest(ctx context.Context, method, u, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("User-Agent", "gropius/1.0 (+https://github.com/intentdriven/Gropius)")
	return req, nil
}

// maxRedirects is the hop limit refuseOffOrigin re-imposes. Setting
// CheckRedirect replaces net/http's own default of 10, so it is spelled here
// rather than inherited.
const maxRedirects = 10

// refuseOffOrigin is the redirect policy every API request runs under: a hop
// that would leave the Hub's origin is refused before it is made.
//
// Before, not after, for two reasons. The Authorization header travels on a
// redirect whenever the target hostname is the Hub's or a subdomain of it —
// net/http strips it only for a different domain — so a hop to
// "<sub>.huggingface.co", or to the same host on another port, would carry the
// access token with it. And a host that answers one hop chooses the next: a
// detour that returns to the Hub's own origin still lets it pick which Hub URL
// we end up decoding, e.g. answering a lookup of one repo with another repo's
// authentic metadata.
func (c *Client) refuseOffOrigin(req *http.Request, via []*http.Request) error {
	if !sameOrigin(c.baseURL(), req.URL.String()) {
		return fmt.Errorf("redirected to %s://%s: %w", req.URL.Scheme, req.URL.Host, ErrCrossOrigin)
	}
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	return nil
}

// do issues an API request under refuseOffOrigin and refuses an answer that
// did not come from the Hub's own origin.
//
// Every path in this package that decodes what the Hub *says* — Search,
// RepoInfo, and each page of Files — goes through here, so all three answer
// the same way about where a response may come from. The policy is installed
// on a shallow copy of the client, which shares its transport and so its
// connection pool: the copy is what keeps the rule off the file-download path
// below. The check on the answer that arrives is a backstop for the copy not
// being in force, and an origin it cannot determine is refused rather than
// allowed. Files keeps its own check in addition, on a rel="next" URL rather
// than on a response, because that URL is followed by a fresh request the
// token is attached to and no redirect policy sees it.
//
// A file download deliberately does not come through here: the Hub answers a
// /resolve/ GET for an LFS object with a redirect to its content CDN, which is
// another host by design, and those bytes are anchored by the sha256 this same
// API stated for them. Small files the repo stores in git carry no such hash
// and are checked on length alone, so for those the exception is wider than
// that justification — iss-2609190151179403 holds the question of narrowing it
// to files the tree gave a hash for.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	hc := *c.httpClient()
	hc.CheckRedirect = c.refuseOffOrigin
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	// resp.Request is the last request in the redirect chain.
	final := resp.Request
	if final == nil || final.URL == nil {
		resp.Body.Close()
		return nil, fmt.Errorf("%s came back with no record of what answered it: %w", req.URL.Path, ErrCrossOrigin)
	}
	if !sameOrigin(c.baseURL(), final.URL.String()) {
		resp.Body.Close()
		return nil, fmt.Errorf("%s was answered by %s://%s — refusing to read it: %w",
			req.URL.Path, final.URL.Scheme, final.URL.Host, ErrCrossOrigin)
	}
	return resp, nil
}

// APIError is a non-2xx response from the Hub.
type APIError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *APIError) Error() string {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Sprintf("huggingface denied access to %s (HTTP %d) — the repo may be gated; add an access token in Settings", e.URL, e.StatusCode)
	case http.StatusNotFound:
		return fmt.Sprintf("huggingface has no such repo or file: %s", e.URL)
	case http.StatusTooManyRequests:
		return fmt.Sprintf("huggingface rate-limited the request to %s (HTTP 429)", e.URL)
	}
	return fmt.Sprintf("huggingface returned HTTP %d for %s: %s", e.StatusCode, e.URL, e.Body)
}

// IsNotFound reports whether err is a 404 from the Hub.
func IsNotFound(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == http.StatusNotFound
	}
	return false
}

// IsAuthRequired reports whether err is the Hub refusing access, which for a
// model repo almost always means it is gated and needs a token.
func IsAuthRequired(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == http.StatusUnauthorized || ae.StatusCode == http.StatusForbidden
	}
	return false
}

// Search finds models on the Hub.
func (c *Client) Search(ctx context.Context, q SearchQuery) ([]Model, error) {
	v := url.Values{}
	if q.Search != "" {
		v.Set("search", q.Search)
	}
	if q.Author != "" {
		v.Set("author", q.Author)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 30
	}
	v.Set("limit", strconv.Itoa(limit))
	if q.Sort != "" {
		v.Set("sort", q.Sort)
		v.Set("direction", "-1")
	}
	// full=false keeps the payload small; we only need summary fields here.
	base, err := c.hubURL(segment("api"), segment("models"))
	if err != nil {
		return nil, err
	}
	u := base + "?" + v.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, u)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("search huggingface: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp, u)
	}
	var models []Model
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONBody)).Decode(&models); err != nil {
		return nil, fmt.Errorf("decode search results: %w", err)
	}
	return models, nil
}

// RepoInfo returns one repo's summary, in the same shape a search result
// carries: what the Hub says this model is.
//
// It exists for the download path, which needs the repo's pipeline tag and tags
// and cannot get them any other way — a search result is not in hand when a
// download is started by name, and nothing in the downloaded files says what
// kind of model they are. One request, to the host the download is already
// talking to — and answered by that host or not at all, because it goes
// through do — bounded by the same maxJSONBody every other decode here is.
//
// The caller decides what a failure means. For a download it means no category,
// which is the same state as a repo the Hub does not tag; it never means the
// model is unusable.
func (c *Client) RepoInfo(ctx context.Context, repoID string) (Model, error) {
	u, err := c.hubURL(segment("api"), segment("models"), repoPath(repoID))
	if err != nil {
		return Model{}, fmt.Errorf("repo info for %q: %w", repoID, err)
	}
	req, err := c.newRequest(ctx, http.MethodGet, u)
	if err != nil {
		return Model{}, err
	}
	resp, err := c.do(req)
	if err != nil {
		return Model{}, fmt.Errorf("read repo info for %s: %w", repoID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Model{}, apiError(resp, u)
	}
	var m Model
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONBody)).Decode(&m); err != nil {
		return Model{}, fmt.Errorf("decode repo info for %s: %w", repoID, err)
	}
	return m, nil
}

// maxJSONBody caps how large a Hub JSON response we will buffer/decode. Search
// results and a single tree page are at most a few MB; a body near this limit is
// a broken or hostile endpoint, not a real repo listing.
const maxJSONBody = 32 << 20

// maxTreePages bounds how many pagination hops Files will follow. The tree
// endpoint pages at 1000 entries, so this covers repos with up to ~1M files —
// far beyond any real model — while refusing an endpoint that loops forever.
const maxTreePages = 1000

// maxTreeBytes and maxTreeEntries bound a file listing as a whole, which
// maxJSONBody and maxTreePages do not: those cap one page and the number of
// hops, and their product — 32 GiB — is what a hostile Hub gets to spend,
// because every page's entries are appended to one slice. A whole listing may
// therefore cost at most what a single page may, and may name at most
// maxTreeEntries files: two orders of magnitude above the most heavily sharded
// real model, and far below anything that threatens this Mac's memory.
const (
	maxTreeBytes   = maxJSONBody
	maxTreeEntries = 100_000
)

// Files lists a repo's file tree at the given revision (default "main"),
// including sizes, which the downloader needs for progress reporting.
//
// The tree endpoint paginates at 1000 entries and signals more with a
// `Link: <...>; rel="next"` header. We follow it to the end: a repo with more
// than 1000 tree entries (a heavily-sharded model, say) would otherwise yield a
// silently truncated list, and the download would "succeed" while missing shards.
//
// The listing is bounded as a whole, not a page at a time: see maxTreeBytes
// and maxTreeEntries. A listing past either is refused with ErrOversizedBody.
func (c *Client) Files(ctx context.Context, repoID, revision string) ([]File, error) {
	return c.files(ctx, repoID, revision, c.Token())
}

// files is Files under a token the caller has already read, so that a download
// lists and fetches a repo under one token rather than picking up a new one
// between the listing and the files it names.
func (c *Client) files(ctx context.Context, repoID, revision, token string) ([]File, error) {
	if revision == "" {
		revision = "main"
	}
	base, err := c.hubURL(segment("api"), segment("models"), repoPath(repoID),
		segment("tree"), segment(revision))
	if err != nil {
		return nil, fmt.Errorf("list files for %q: %w", repoID, err)
	}
	u := base + "?recursive=true"

	var entries []File
	budget := int64(maxTreeBytes)
	for page := 0; u != ""; page++ {
		if page >= maxTreePages {
			return nil, fmt.Errorf("file tree for %s did not terminate after %d pages", repoID, maxTreePages)
		}
		req, err := c.newTokenRequest(ctx, http.MethodGet, u, token)
		if err != nil {
			return nil, err
		}
		resp, err := c.do(req)
		if err != nil {
			return nil, fmt.Errorf("list files for %s: %w", repoID, err)
		}
		if resp.StatusCode != http.StatusOK {
			err := apiError(resp, u)
			resp.Body.Close()
			return nil, err
		}

		// One byte past the remaining budget is read deliberately: reaching
		// it is how the page is known to have spent more than the listing had
		// left, and it is refused as oversized rather than as a decode error.
		// While the budget is larger than a page may be, the per-page cap is
		// what binds and this reads exactly as it did before.
		var pageEntries []File
		counted := &countingReader{r: io.LimitReader(resp.Body, min(int64(maxJSONBody), budget+1))}
		decodeErr := json.NewDecoder(counted).Decode(&pageEntries)
		if counted.n > budget {
			resp.Body.Close()
			return nil, fmt.Errorf("file tree for %s ran past the %d bytes one listing may cost, at page %d: %w",
				repoID, maxTreeBytes, page+1, ErrOversizedBody)
		}
		if decodeErr != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode file tree for %s: %w", repoID, decodeErr)
		}
		budget -= counted.n
		if len(entries)+len(pageEntries) > maxTreeEntries {
			resp.Body.Close()
			return nil, fmt.Errorf("file tree for %s names more than the %d files one listing may hold: %w",
				repoID, maxTreeEntries, ErrOversizedBody)
		}
		link := resp.Header.Get("Link")
		// The page this Link came off is where the request ended up, not where
		// it was sent: the Hub redirects a re-cased or renamed repo id to its
		// canonical URL, and a relative next page has to be resolved against
		// the canonical one. do guarantees resp.Request is set and that its
		// URL is on the Hub's own origin.
		pageURL := resp.Request.URL.String()
		resp.Body.Close()

		next, err := c.nextPage(pageURL, link)
		if err != nil {
			return nil, fmt.Errorf("file tree for %s %w", repoID, err)
		}

		entries = append(entries, pageEntries...)
		u = next
	}

	// LFS entries carry the true size in the nested object; the outer size is
	// the pointer file's size for some repos. Prefer the LFS size when present.
	// Directory entries are not fetchable — drop them so they never reach the
	// downloader (a GET of a directory 404s and aborts the whole download).
	out := entries[:0]
	for _, e := range entries {
		if e.Type == "directory" {
			continue
		}
		if e.LFS != nil && e.LFS.Size > 0 {
			e.Size = e.LFS.Size
			e.OID = e.LFS.OID
		}
		out = append(out, e)
	}
	return out, nil
}

// countingReader reports how many bytes were taken from the reader beneath it.
// A json.Decoder reads ahead into its own buffer, so what it consumed is not
// what the JSON it returned was worth; the count below is of bytes handed on,
// which is what the listing's budget is spent in.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// nextPage is the rel="next" URL of the page fetched from pageURL, or "" when
// there is no next page.
//
// The Link header is attacker-influenced (it comes from the Hub response).
// newTokenRequest attaches the bearer token to whatever URL we pass, so a
// "rel=next" pointing at another host would leak the HuggingFace token off to
// it. This is a fresh request rather than a redirect, so refuseOffOrigin never
// sees it: the same sameOrigin rule is applied here instead.
//
// The reference is resolved against the page that carried it before that rule
// runs. RFC 8288 permits a relative URI-reference, and the Hub's paging URLs
// are absolute only by current practice; unresolved, a relative next page has
// no host, which the origin rule can only read as a different origin — so the
// listing would stop at page one and the error would name a same-origin path
// as cross-origin. A value that will not parse is called unparseable, which is
// what it is.
//
// Resolving is what makes the origin rule insufficient on its own. Before it,
// only an absolute URL could be followed and anything else was refused for
// want of a host; after it, any string that resolves lands somewhere on the
// Hub's origin, and would be followed with the bearer token on it. So the
// resolved reference must also be a continuation of THIS listing: the same
// path, differing only in the query, which is how the Hub pages. That is what
// keeps a "next page" from splicing another repo's tree into the one that was
// asked for — the risk do names for a redirect, arriving by the one door do
// does not watch — and what refuses a Link value that is not a URL at all but
// resolves to a plausible path anyway.
//
// Userinfo is dropped, and so is any fragment. url.URL.Host excludes userinfo,
// so it passes the origin check untouched, and net/http then turns it into an
// Authorization header on a request we wrote none for.
func (c *Client) nextPage(pageURL, link string) (string, error) {
	raw := nextPageURL(link)
	if raw == "" {
		return "", nil
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("returned an unparseable next page (%s): %w", raw, err)
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("was fetched from an unparseable URL (%s): %w", pageURL, err)
	}
	next := base.ResolveReference(ref)
	next.User = nil
	next.Fragment, next.RawFragment = "", ""
	if !sameOrigin(c.baseURL(), next.String()) {
		return "", fmt.Errorf("returned a cross-origin next page (%s) — refusing to follow it: %w", next, ErrCrossOrigin)
	}
	if next.Path != base.Path {
		return "", fmt.Errorf("returned a next page onto another listing (%s, not %s) — refusing to follow it", next.Path, base.Path)
	}
	return next.String(), nil
}

// ErrCrossOrigin is what every request path in this package refuses with when
// an answer would come from, or a next page would point at, a host that is not
// the Hub's own origin. One class, so a caller asks one question of all of
// them.
var ErrCrossOrigin = errors.New("the answer did not come from the hub's origin")

// sameOrigin reports whether target has the same scheme and host as base. A
// parse failure or missing host counts as different, i.e. refuse it.
func sameOrigin(base, target string) bool {
	b, err := url.Parse(base)
	if err != nil {
		return false
	}
	t, err := url.Parse(target)
	if err != nil || t.Host == "" {
		return false
	}
	return strings.EqualFold(b.Scheme, t.Scheme) && strings.EqualFold(b.Host, t.Host)
}

// nextPageURL extracts the rel="next" URL from an RFC 8288 Link header, or "".
func nextPageURL(link string) string {
	for _, part := range strings.Split(link, ",") {
		segs := strings.Split(strings.TrimSpace(part), ";")
		if len(segs) < 2 {
			continue
		}
		isNext := false
		for _, s := range segs[1:] {
			if strings.Contains(strings.ToLower(s), `rel="next"`) {
				isNext = true
				break
			}
		}
		if !isNext {
			continue
		}
		u := strings.TrimSpace(segs[0])
		return strings.TrimSuffix(strings.TrimPrefix(u, "<"), ">")
	}
	return ""
}

// RepoSize returns the total download size, in bytes, of the MLX-relevant files
// in a repo at the default revision. This is what Gropius would actually fetch.
func (c *Client) RepoSize(ctx context.Context, repoID string) (int64, error) {
	files, err := c.Files(ctx, repoID, "main")
	if err != nil {
		return 0, err
	}
	return TotalSize(WantedFiles(files)), nil
}

// ResolveURL is the direct-download URL for one file in a repo, or the reason
// there is none.
//
// The repo id, the revision and each file-path segment are checked and escaped
// through the same hubURL every other request path here goes through: a file
// named e.g. "weights#2.safetensors" would otherwise have everything after '#'
// parsed as a URL fragment, producing a wrong request that 404s and aborts the
// download. This is the path that most needs it — a file download deliberately
// does not go through do, so nothing downstream asks where the answer came
// from.
func (c *Client) ResolveURL(repoID, revision, file string) (string, error) {
	if revision == "" {
		revision = "main"
	}
	return c.hubURL(repoPath(repoID), segment("resolve"), segment(revision), repoPath(file))
}

// urlPart is one value on its way into a Hub URL path, together with how it
// must be escaped.
type urlPart struct {
	value string
	// isPath marks a value that is a path by nature — a repo id ("org/name"),
	// a repo-relative file path — whose '/' are separators and stay
	// separators. It is unset for a value that must occupy exactly one path
	// element, such as a revision, where an unescaped '/' would silently name
	// a deeper endpoint than the caller asked for.
	isPath bool
}

// segment is a value that must occupy exactly one path element.
func segment(v string) urlPart { return urlPart{value: v} }

// repoPath is a value that is itself a '/'-separated path.
func repoPath(v string) urlPart { return urlPart{value: v, isPath: true} }

// hubURL is the one place in this package where a URL under the Hub's base is
// built, so that every value reaching a Hub path is checked and escaped the
// same way, exactly once. Escaping one path and interpolating another raw is
// how a repo id carrying '?' or '#' reached a different endpoint than the
// caller named. A caller that needs a query string appends it to the result;
// no caller-supplied value belongs in one.
//
// Escaping is what keeps a '?' or a '#' inside the element it was written in.
// Two things it cannot do anything about, so they are refused here rather than
// on whichever value someone remembered to check:
//
// A "." or a ".." is not an element at all but an instruction about the path.
// url.PathEscape leaves it alone and net/http forwards it verbatim for the
// server to resolve, so "org/../../evil" — or a revision of ".." — reaches an
// endpoint the caller never named. Nothing on the Hub is named that.
//
// An empty value would simply drop an element, and a path one element shorter
// is another valid endpoint, not an obviously broken URL.
func (c *Client) hubURL(parts ...urlPart) (string, error) {
	segs := make([]string, 0, len(parts))
	for _, p := range parts {
		if p.value == "" {
			return "", errors.New("refusing to build a hub URL: a path element is empty")
		}
		elems := []string{p.value}
		if p.isPath {
			elems = strings.Split(p.value, "/")
		}
		for _, e := range elems {
			if e == "." || e == ".." {
				return "", fmt.Errorf("refusing to build a hub URL: %q in %q is not a path element", e, p.value)
			}
		}
		if p.isPath {
			segs = append(segs, escapePathSegments(p.value))
			continue
		}
		segs = append(segs, url.PathEscape(p.value))
	}
	return c.baseURL() + "/" + strings.Join(segs, "/"), nil
}

// escapePathSegments percent-escapes each '/'-separated segment while keeping the
// separators, so a repo-relative path stays a valid, correct URL path.
func escapePathSegments(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

// wantedFile reports whether a repo file is needed to run the model.
//
// This is a deny list rather than an allow list: missing a required file breaks
// the model in ways that surface only at load time, whereas an unexpected extra
// file merely wastes a little disk. The denied extensions are weights in
// formats MLX cannot use (PyTorch, ONNX, GGUF, TF) plus repo cruft — these are
// the entries big enough to matter.
func wantedFile(p string) bool {
	if p == "" || strings.HasSuffix(p, "/") {
		return false
	}
	// Reject any path that could escape the destination directory. safeJoin in
	// the downloader is the real guard, but filtering here means such files never
	// even appear in progress totals or the file list.
	if strings.Contains(p, "..") {
		return false
	}
	base := path.Base(p)
	if base == ".gitattributes" {
		return false
	}
	// Skip hidden entries and anything under a hidden top-level folder
	// (".gitignore", ".cache/…") — repo metadata, never model content.
	if strings.HasPrefix(p, ".") {
		return false
	}
	deniedExt := []string{
		".bin", ".pth", ".pt", ".ckpt", // PyTorch weights
		".onnx", ".gguf", ".ggml", // other runtimes
		".h5", ".msgpack", ".tflite", // TF/Flax
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".mp4", // media
		".zip", ".tar", ".gz",
	}
	lower := strings.ToLower(base)
	for _, ext := range deniedExt {
		if strings.HasSuffix(lower, ext) {
			return false
		}
	}
	return true
}

// WantedFiles filters a file tree down to what MLX needs to load the model,
// de-duplicating by cleaned path. A tree that lists the same path twice (a
// malformed or hostile manifest) would otherwise spawn two goroutines racing to
// write and rename the same file, and double-count it in the size total so
// progress never reaches 100%.
func WantedFiles(files []File) []File {
	out := make([]File, 0, len(files))
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		if !wantedFile(f.Path) {
			continue
		}
		key := path.Clean(f.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	return out
}

// caseCollision reports the first pair of wanted paths that differ only by
// case. Such a pair cannot both exist on macOS's case-insensitive default
// volume: two goroutines would race over one .gropius-part and, at best, the
// download aborts. The repo is refused up front instead.
func caseCollision(files []File) (a, b string, collide bool) {
	seen := make(map[string]string, len(files))
	for _, f := range files {
		clean := path.Clean(f.Path)
		folded := strings.ToLower(clean)
		if prev, ok := seen[folded]; ok {
			if prev != clean {
				return prev, clean, true
			}
			continue
		}
		seen[folded] = clean
	}
	return "", "", false
}

// TotalSize sums the sizes of a file list.
func TotalSize(files []File) int64 {
	var n int64
	for _, f := range files {
		n += f.Size
	}
	return n
}

// HasWeights reports whether the file list contains MLX-loadable weights.
// A repo without safetensors is not a usable MLX model, and we would rather say
// so before downloading gigabytes than after.
func HasWeights(files []File) bool {
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Path), ".safetensors") {
			return true
		}
	}
	return false
}

func apiError(resp *http.Response, u string) error {
	// Read, not a single resp.Body.Read call: io.Reader may legitimately
	// return fewer bytes than the buffer even mid-body (chunked framing, TLS
	// record boundaries), which would otherwise truncate the captured error
	// message well before the intended 512-byte budget.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return &APIError{StatusCode: resp.StatusCode, URL: u, Body: strings.TrimSpace(string(body))}
}

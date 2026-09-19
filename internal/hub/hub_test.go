package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestSearchParsesHubResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id":"mlx-community/Qwen3-8B-4bit","downloads":1234,"likes":7,"tags":["mlx","safetensors"],"pipeline_tag":"text-generation"},
			{"id":"mlx-community/Llama-3.2-1B-Instruct-4bit","downloads":99,"likes":1,"tags":["mlx"]}
		]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	models, err := c.Search(context.Background(), SearchQuery{
		Search: "qwen", Author: "mlx-community", Limit: 5, Sort: "downloads",
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0].ID != "mlx-community/Qwen3-8B-4bit" {
		t.Errorf("ID = %q", models[0].ID)
	}
	if models[0].Downloads != 1234 {
		t.Errorf("Downloads = %d, want 1234", models[0].Downloads)
	}
	for _, want := range []string{"search=qwen", "author=mlx-community", "limit=5", "sort=downloads"} {
		if !containsStr(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

func TestSearchSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Search(context.Background(), SearchQuery{}); err == nil {
		t.Fatal("expected an error for HTTP 500")
	}
}

// apiError must not truncate the captured error message just because the
// response body arrives across several small reads (chunked framing, TLS
// record boundaries) — io.Reader is explicitly allowed to return fewer bytes
// than the buffer even mid-body, so a single Read call is not enough.
func TestAPIErrorReadsFullBodyAcrossShortReads(t *testing.T) {
	msg := strings.Repeat("boom ", 100) // 500 bytes, far more than one short read
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(&shortReader{data: []byte(msg)}),
	}

	err := apiError(resp, "http://example.test")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("got %T, want *APIError", err)
	}
	want := strings.TrimSpace(msg)
	if apiErr.Body != want {
		t.Errorf("Body = %q (%d bytes), want %q (%d bytes)", apiErr.Body, len(apiErr.Body), want, len(want))
	}
}

// shortReader never returns more than 5 bytes per Read, regardless of the
// caller's buffer size.
type shortReader struct{ data []byte }

func (r *shortReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(5, len(p), len(r.data))
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestModelMetadataHelpers(t *testing.T) {
	tests := []struct {
		id    string
		org   string
		name  string
		quant string
	}{
		{"mlx-community/Qwen3-8B-4bit", "mlx-community", "Qwen3-8B-4bit", "4bit"},
		{"mlx-community/Llama-3.3-70B-Instruct-8bit", "mlx-community", "Llama-3.3-70B-Instruct-8bit", "8bit"},
		{"mlx-community/Qwen3-4B-4bit-DWQ", "mlx-community", "Qwen3-4B-4bit-DWQ", "4bit-dwq"},
		{"mlx-community/Mistral-7B-bf16", "mlx-community", "Mistral-7B-bf16", "bf16"},
		{"someone/plain-model", "someone", "plain-model", ""},
	}
	for _, tt := range tests {
		m := Model{ID: tt.id}
		if got := m.Org(); got != tt.org {
			t.Errorf("%s: Org = %q, want %q", tt.id, got, tt.org)
		}
		if got := m.Name(); got != tt.name {
			t.Errorf("%s: Name = %q, want %q", tt.id, got, tt.name)
		}
		if got := m.Quantization(); got != tt.quant {
			t.Errorf("%s: Quantization = %q, want %q", tt.id, got, tt.quant)
		}
	}
}

func TestFilesPrefersLFSSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models/org/repo/tree/main" {
			t.Errorf("unexpected tree path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// The weights entry reports the LFS size in a nested object.
		fmt.Fprint(w, `[
			{"path":"config.json","size":937,"oid":"abc"},
			{"path":"model.safetensors","size":135,"oid":"ptr","lfs":{"oid":"deadbeef","size":335450584}}
		]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	weights := files[1]
	if weights.Size != 335450584 {
		t.Errorf("LFS size = %d, want the nested LFS size 335450584 (not the pointer size)", weights.Size)
	}
	if weights.OID != "deadbeef" {
		t.Errorf("OID = %q, want the LFS oid", weights.OID)
	}
}

// The recursive tree listing includes directory entries; they are not fetchable
// (a GET of one 404s and aborts the whole download), so Files must drop them.
// A tree that lists the same path twice must be de-duplicated, or two goroutines
// race to write the same file and TotalSize double-counts it.
func TestWantedFilesDeduplicates(t *testing.T) {
	files := []File{
		{Path: "model.safetensors", Size: 100},
		{Path: "config.json", Size: 10},
		{Path: "model.safetensors", Size: 100}, // duplicate
		{Path: "./config.json", Size: 10},      // duplicate via a different spelling
	}
	got := WantedFiles(files)
	if len(got) != 2 {
		t.Fatalf("WantedFiles kept %d entries, want 2 deduplicated: %+v", len(got), got)
	}
	if n := TotalSize(got); n != 110 {
		t.Errorf("TotalSize after dedup = %d, want 110 (a double-count keeps progress under 100%%)", n)
	}
}

func TestFilesDropsDirectoryEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"type":"directory","path":"assets","size":0},
			{"type":"file","path":"config.json","size":10,"oid":"a"},
			{"type":"file","path":"model.safetensors","size":20,"oid":"b"}
		]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2 (the directory entry must be dropped)", len(files))
	}
	for _, f := range files {
		if f.Path == "assets" {
			t.Error("directory entry 'assets' leaked into the file list")
		}
	}
}

// Repos with more than 1000 tree entries paginate via a Link rel="next" header;
// Files must follow it or downloads silently omit later shards.
func TestFilesFollowsPagination(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			w.Header().Set("Link", "<"+srv.URL+"/api/models/org/repo/tree/main?recursive=true&cursor=p2>; rel=\"next\"")
			fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
			return
		}
		fmt.Fprint(w, `[{"type":"file","path":"b.safetensors","size":2,"oid":"b"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2 across both pages", len(files))
	}
}

// A rel="next" pointing at a different host must not be followed: newRequest
// attaches the bearer token, so following it would leak the HuggingFace token
// to the attacker-named host.
func TestFilesRefusesCrossOriginNextPage(t *testing.T) {
	var leaked bool
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			leaked = true
		}
		fmt.Fprint(w, `[]`)
	}))
	defer evil.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", "<"+evil.URL+"/api/models/org/repo/tree/main?cursor=p2>; rel=\"next\"")
		fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.SetToken("secret-hf-token")
	_, err := c.Files(context.Background(), "org/repo", "")
	if err == nil {
		t.Fatal("Files followed a cross-origin next page; it must refuse")
	}
	if leaked {
		t.Fatal("TOKEN LEAK: the bearer token was sent to the cross-origin host")
	}
}

func TestFilesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Files(context.Background(), "org/nope", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound should be true for a 404, got err = %v", err)
	}
}

func TestGatedRepoReportsAuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gated", http.StatusForbidden)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Files(context.Background(), "meta/gated", "")
	if !IsAuthRequired(err) {
		t.Errorf("IsAuthRequired should be true for a 403, got err = %v", err)
	}
}

func TestTokenIsSentAsBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.SetToken("hf_abc123")
	if _, err := c.Search(context.Background(), SearchQuery{}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer hf_abc123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer hf_abc123")
	}
}

func TestResolveURL(t *testing.T) {
	c := &Client{BaseURL: "https://huggingface.co"}
	got, err := c.ResolveURL("mlx-community/Qwen3-0.6B-4bit", "", "model.safetensors")
	if err != nil {
		t.Fatalf("ResolveURL: %v", err)
	}
	want := "https://huggingface.co/mlx-community/Qwen3-0.6B-4bit/resolve/main/model.safetensors"
	if got != want {
		t.Errorf("ResolveURL = %q, want %q", got, want)
	}
}

// A filename with URL-significant characters must be escaped, or the '#' turns
// the rest into a fragment and the GET hits the wrong path.
func TestResolveURLEscapesSpecialChars(t *testing.T) {
	c := &Client{BaseURL: "https://huggingface.co"}
	got, err := c.ResolveURL("org/repo", "main", "weights#2.safetensors")
	if err != nil {
		t.Fatalf("ResolveURL: %v", err)
	}
	want := "https://huggingface.co/org/repo/resolve/main/weights%232.safetensors"
	if got != want {
		t.Errorf("ResolveURL = %q, want %q", got, want)
	}
	// Path separators must survive as separators, not be escaped.
	nested, err := c.ResolveURL("org/repo", "main", "sub/dir/model.json")
	if err != nil {
		t.Fatalf("ResolveURL: %v", err)
	}
	if nested != "https://huggingface.co/org/repo/resolve/main/sub/dir/model.json" {
		t.Errorf("nested path mangled: %q", nested)
	}
}

func TestWantedFilesKeepsMLXFilesAndDropsForeignWeights(t *testing.T) {
	files := []File{
		{Path: "config.json"},
		{Path: "model.safetensors"},
		{Path: "model.safetensors.index.json"},
		{Path: "tokenizer.json"},
		{Path: "tokenizer_config.json"},
		{Path: "special_tokens_map.json"},
		{Path: "merges.txt"},
		{Path: "vocab.json"},
		{Path: "chat_template.jinja"},
		{Path: "tokenizer.model"}, // sentencepiece — required by some models
		// Not needed / wrong runtime:
		{Path: ".gitattributes"},
		{Path: "pytorch_model.bin"},
		{Path: "model.onnx"},
		{Path: "model.gguf"},
		{Path: "figure.png"},
	}
	got := WantedFiles(files)

	wantKept := map[string]bool{
		"config.json": true, "model.safetensors": true,
		"model.safetensors.index.json": true, "tokenizer.json": true,
		"tokenizer_config.json": true, "special_tokens_map.json": true,
		"merges.txt": true, "vocab.json": true,
		"chat_template.jinja": true, "tokenizer.model": true,
		"README.md": true,
	}
	for _, f := range got {
		if !wantKept[f.Path] {
			t.Errorf("file %q should have been filtered out", f.Path)
		}
	}
	kept := map[string]bool{}
	for _, f := range got {
		kept[f.Path] = true
	}
	// tokenizer.model must survive: dropping it silently breaks sentencepiece models.
	if !kept["tokenizer.model"] {
		t.Error("tokenizer.model must be kept — some models cannot tokenize without it")
	}
	if !kept["model.safetensors"] {
		t.Error("weights must be kept")
	}
	if kept["pytorch_model.bin"] || kept["model.onnx"] || kept["model.gguf"] {
		t.Error("non-MLX weight formats must be filtered out to save bandwidth")
	}
}

func TestHasWeights(t *testing.T) {
	if HasWeights([]File{{Path: "config.json"}}) {
		t.Error("a repo with no safetensors has no weights")
	}
	if !HasWeights([]File{{Path: "model-00001-of-00002.safetensors"}}) {
		t.Error("sharded safetensors count as weights")
	}
}

func TestTotalSize(t *testing.T) {
	got := TotalSize([]File{{Size: 100}, {Size: 250}})
	if got != 350 {
		t.Errorf("TotalSize = %d, want 350", got)
	}
}

func TestProgressPercentAndETA(t *testing.T) {
	p := Progress{Completed: 50, Total: 200, BytesPerSec: 10}
	if got := p.Percent(); got != 25 {
		t.Errorf("Percent = %v, want 25", got)
	}
	if got := p.ETA().Seconds(); got != 15 {
		t.Errorf("ETA = %vs, want 15s", got)
	}
	// Guard against divide-by-zero on an empty repo.
	if got := (Progress{}).Percent(); got != 0 {
		t.Errorf("Percent of empty progress = %v, want 0", got)
	}
}

// fakeHub serves a tree endpoint and resolve endpoints for a set of files,
// honoring Range requests the way the real Hub CDN does.
//
// Handlers run concurrently (the downloader fetches files in parallel), so the
// bookkeeping maps are mutex-guarded.
type fakeHub struct {
	files map[string][]byte
	// ignoreRange makes the server reply 200 with the whole body even when a
	// Range was requested, which some CDNs and proxies do.
	ignoreRange bool
	// lfs maps a path to the LFS content oid (sha256) advertised in the tree.
	lfs map[string]string
	// badContentRange makes a 206 reply carry a Content-Range that starts at the
	// wrong offset, simulating a misbehaving CDN/proxy.
	badContentRange bool
	// always416 makes every resolve request fail with 416, even a plain GET
	// without a Range header, simulating a broken proxy or CDN edge.
	always416 bool

	mu sync.Mutex
	// ranges records the Range header seen per file path.
	ranges map[string]string
	hits   map[string]int
}

func newFakeHub(files map[string][]byte) *fakeHub {
	return &fakeHub{files: files, ranges: map[string]string{}, hits: map[string]int{}}
}

// rangeFor returns the Range header the server saw for a path.
func (f *fakeHub) rangeFor(path string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ranges[path]
}

// hitsFor returns how many times a path was fetched.
func (f *fakeHub) hitsFor(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits[path]
}

func (f *fakeHub) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/org/repo/tree/main", func(w http.ResponseWriter, r *http.Request) {
		var entries []File
		for p, b := range f.files {
			e := File{Path: p, Size: int64(len(b))}
			if oid, ok := f.lfs[p]; ok {
				e.LFS = &struct {
					OID  string `json:"oid"`
					Size int64  `json:"size"`
				}{OID: oid, Size: int64(len(b))}
			}
			entries = append(entries, e)
		}
		json.NewEncoder(w).Encode(entries)
	})
	mux.HandleFunc("/org/repo/resolve/main/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/org/repo/resolve/main/"):]
		body, ok := f.files[name]
		if !ok {
			http.Error(w, "no such file", http.StatusNotFound)
			return
		}
		rng := r.Header.Get("Range")
		f.mu.Lock()
		f.hits[name]++
		if rng != "" {
			f.ranges[name] = rng
		}
		f.mu.Unlock()
		if f.always416 {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if rng != "" && !f.ignoreRange {
			var start int64
			fmt.Sscanf(rng, "bytes=%d-", &start)
			if start >= int64(len(body)) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if f.badContentRange {
				// A misbehaving proxy: claims 206 but resumes at offset 0 and sends
				// the WHOLE body. Appending that onto our .part overflows the size
				// check; the client must detect the wrong range and restart instead.
				w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(body)-1, len(body)))
				w.WriteHeader(http.StatusPartialContent)
				w.Write(body)
				return
			}
			w.Header().Set("Content-Range",
				fmt.Sprintf("bytes %d-%d/%d", start, len(body)-1, len(body)))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(body[start:])
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func containsStr(hay, needle string) bool {
	return len(hay) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(hay); i++ {
			if hay[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

// RepoInfo is how a download learns what kind of model it just fetched: the
// Hub's own pipeline tag and tags, from the repo endpoint, in the same shape a
// search result carries them.
func TestRepoInfoReadsTheCategory(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"mlx-community/Qwen3-8B-4bit","downloads":12,"likes":3,
			"tags":["mlx","conversational","text-generation"],"pipeline_tag":"text-generation"}`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	m, err := c.RepoInfo(context.Background(), "mlx-community/Qwen3-8B-4bit")
	if err != nil {
		t.Fatalf("RepoInfo: %v", err)
	}
	if gotPath != "/api/models/mlx-community/Qwen3-8B-4bit" {
		t.Errorf("asked for %q, want the repo endpoint", gotPath)
	}
	if m.PipelineTag != "text-generation" {
		t.Errorf("PipelineTag = %q, want text-generation", m.PipelineTag)
	}
	if len(m.Tags) != 3 || m.Tags[1] != "conversational" {
		t.Errorf("Tags = %v, want the Hub's three", m.Tags)
	}
}

// A repo the Hub does not tag is not an error: it is a model with no category,
// which is a thing that exists and has to stay downloadable.
func TestRepoInfoOnAnUntaggedRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"org/quiet"}`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	m, err := c.RepoInfo(context.Background(), "org/quiet")
	if err != nil {
		t.Fatalf("RepoInfo: %v", err)
	}
	if m.PipelineTag != "" || len(m.Tags) != 0 {
		t.Errorf("an untagged repo returned %q/%v, want nothing", m.PipelineTag, m.Tags)
	}
}

// And a repo that is gone, or gated, is an error the caller can recognize
// rather than an empty category it would record as fact.
func TestRepoInfoSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such repo", http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.RepoInfo(context.Background(), "org/missing")
	if err == nil {
		t.Fatal("expected an error for HTTP 404")
	}
	if !IsNotFound(err) {
		t.Errorf("err = %v, want a recognizable 404", err)
	}
}

// A redirect is the other way a host that is not the Hub can end up answering
// for it: net/http follows up to ten of them, so without a rule the decoded
// answer about what a repo is comes from wherever the last hop pointed. The
// token is not the exposure here — net/http strips Authorization on a redirect
// to another domain — the body is, and RepoInfo must refuse it the way Files
// refuses a cross-origin next page.
func TestRepoInfoRefusesARedirectOffTheHubsOrigin(t *testing.T) {
	var reached bool
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		fmt.Fprint(w, `{"id":"org/repo","pipeline_tag":"text-generation","tags":["mlx"]}`)
	}))
	defer evil.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+r.URL.Path, http.StatusFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	m, err := c.RepoInfo(context.Background(), "org/repo")
	if err == nil {
		t.Fatalf("RepoInfo read a redirected answer (%+v); it must refuse", m)
	}
	if !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want the same class Files gives (ErrCrossOrigin)", err)
	}
	if m.PipelineTag != "" || len(m.Tags) != 0 {
		t.Errorf("a refused answer still carried %q/%v", m.PipelineTag, m.Tags)
	}
	_ = reached // the other host may be reached; what must not happen is believing it.
}

// The same rule, from the other request path: whatever Files refuses a
// cross-origin next page with is the class RepoInfo answers with too, so a
// caller asks one question about both.
func TestFilesRefusesACrossOriginNextPageWithTheSameClass(t *testing.T) {
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[]`)
	}))
	defer evil.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", "<"+evil.URL+"/api/models/org/repo/tree/main?cursor=p2>; rel=\"next\"")
		fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Files(context.Background(), "org/repo", "")
	if !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want ErrCrossOrigin", err)
	}
}

// And Search, the third path that decodes what the Hub says.
func TestSearchRefusesARedirectOffTheHubsOrigin(t *testing.T) {
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"evil/model","tags":["mlx"]}]`)
	}))
	defer evil.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/api/models", http.StatusFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	models, err := c.Search(context.Background(), SearchQuery{Search: "qwen"})
	if err == nil {
		t.Fatalf("Search read a redirected answer (%v); it must refuse", models)
	}
	if !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want ErrCrossOrigin", err)
	}
}

// A redirect back onto the Hub's own origin is ordinary and stays followed —
// the Hub redirects a re-cased or renamed repo id to its canonical one.
func TestRepoInfoFollowsARedirectOnTheHubsOwnOrigin(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/org/canonical", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"org/canonical","pipeline_tag":"text-generation"}`)
	})
	mux.HandleFunc("/api/models/ORG/Canonical", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/models/org/canonical", http.StatusMovedPermanently)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	m, err := c.RepoInfo(context.Background(), "ORG/Canonical")
	if err != nil {
		t.Fatalf("RepoInfo refused a same-origin redirect: %v", err)
	}
	if m.ID != "org/canonical" {
		t.Errorf("ID = %q, want the canonical id the Hub redirected to", m.ID)
	}
}

// Refusing the body after the fact is too late for the token. net/http keeps
// the Authorization header on a redirect to the same hostname — a different
// port, or a subdomain — so the rule has to be applied to the hop before it is
// made, not to the answer after it arrives.
func TestAnAPIRedirectOffTheHubsOriginIsRefusedBeforeTheTokenTravels(t *testing.T) {
	var gotAuth string
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, `{"id":"org/repo"}`)
	}))
	defer evil.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+r.URL.Path, http.StatusFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.SetToken("secret-hf-token")
	if _, err := c.RepoInfo(context.Background(), "org/repo"); !errors.Is(err, ErrCrossOrigin) {
		t.Fatalf("err = %v, want ErrCrossOrigin", err)
	}
	if gotAuth != "" {
		t.Fatalf("TOKEN LEAK: the bearer token was sent to the redirect target (%q)", gotAuth)
	}
}

// A hop off the Hub that comes back onto it is still a host that is not the
// Hub choosing which Hub answer we decode: it can point a lookup of one repo
// at another repo's authentic metadata. Checking only where the last hop
// landed accepts that; checking each hop refuses it.
func TestAnAPIDetourThroughAnotherHostIsRefused(t *testing.T) {
	var srv *httptest.Server
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/api/models/attacker/lookalike", http.StatusFound)
	}))
	defer evil.Close()

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/models/org/wanted" {
			http.Redirect(w, r, evil.URL+"/detour", http.StatusFound)
			return
		}
		fmt.Fprint(w, `{"id":"attacker/lookalike","pipeline_tag":"text-generation"}`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	m, err := c.RepoInfo(context.Background(), "org/wanted")
	if err == nil {
		t.Fatalf("RepoInfo followed a detour through another host and answered %q", m.ID)
	}
	if !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want ErrCrossOrigin", err)
	}
}

// A transport that hands back a response with no record of what was requested
// leaves the origin unknown, and an unknown origin is refused: a rule about
// where an answer may come from cannot default to allowing it.
func TestAnAnswerWithNoRecordedOriginIsRefused(t *testing.T) {
	c := &Client{
		BaseURL: "https://huggingface.co",
		HTTP:    &http.Client{Transport: originlessTransport{}},
	}
	if _, err := c.Search(context.Background(), SearchQuery{Search: "qwen"}); !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want ErrCrossOrigin", err)
	}
}

// originlessTransport answers every request without setting Response.Request,
// which is how the final URL goes missing.
type originlessTransport struct{}

func (originlessTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(`[{"id":"evil/model","tags":["mlx"]}]`)),
	}, nil
}

// RFC 8288 lets a Link header carry a relative URI-reference, and the Hub's
// paging URLs are absolute only by current practice. A relative next page is
// the Hub's own: it must be resolved against the page that carried it and
// followed, not refused — and certainly not refused as "cross-origin", which
// names a same-origin path as off-origin and sends a debugger the wrong way.
func TestFilesFollowsARelativeNextPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			w.Header().Set("Link", `</api/models/org/repo/tree/main?recursive=true&cursor=p2>; rel="next"`)
			fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
			return
		}
		fmt.Fprint(w, `[{"type":"file","path":"b.safetensors","size":2,"oid":"b"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("Files refused a relative next page: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2 across both pages", len(files))
	}
}

// Resolving a relative next page must not weaken the origin rule: a
// protocol-relative reference resolves to another host and is still refused,
// with the token never sent to it.
func TestFilesRefusesAProtocolRelativeNextPage(t *testing.T) {
	var leaked bool
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			leaked = true
		}
		fmt.Fprint(w, `[]`)
	}))
	defer evil.Close()
	evilHost := strings.TrimPrefix(evil.URL, "http://")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", "<//"+evilHost+"/api/models/org/repo/tree/main?cursor=p2>; rel=\"next\"")
		fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	c.SetToken("secret-hf-token")
	_, err := c.Files(context.Background(), "org/repo", "")
	if !errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want ErrCrossOrigin", err)
	}
	if leaked {
		t.Fatal("TOKEN LEAK: the bearer token was sent to the host a protocol-relative next page named")
	}
}

// A next page that will not parse at all is neither followed nor described as
// cross-origin: it is unparseable, and saying so is what points a debugger at
// the Link header rather than at the origin rule.
func TestFilesNamesAnUnparseableNextPageAsUnparseable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<::not-a-url>; rel="next"`)
		fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Files(context.Background(), "org/repo", "")
	if err == nil {
		t.Fatal("Files accepted an unparseable next page")
	}
	if !strings.Contains(err.Error(), "unparseable") {
		t.Errorf("err = %v, want it to say the next page is unparseable", err)
	}
	if errors.Is(err, ErrCrossOrigin) {
		t.Errorf("err = %v, want it not to be reported as cross-origin", err)
	}
}

// Every path in this package that puts a repo id into a URL must escape it the
// same way. It escaped on the RepoInfo path and interpolated raw on the tree
// and resolve paths, so a repo id carrying a URL-significant character reached
// a different Hub endpoint than the caller named: a '#' turned the rest of the
// path into a fragment that is never sent at all.
func TestEveryPathEscapesTheRepoIDTheSameWay(t *testing.T) {
	const repoID = "org/repo#1"

	var sawInfo, sawTree, sawResolve string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/tree/main") {
			sawTree = r.URL.Path
			fmt.Fprint(w, `[{"type":"file","path":"model.safetensors","size":4,"oid":"a"}]`)
			return
		}
		sawInfo = r.URL.Path
		fmt.Fprint(w, `{"id":"org/repo#1","tags":["mlx"]}`)
	})
	mux.HandleFunc("/org/repo#1/resolve/main/", func(w http.ResponseWriter, r *http.Request) {
		sawResolve = r.URL.Path
		w.Write([]byte("abcd"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.RepoInfo(context.Background(), repoID); err != nil {
		t.Fatalf("RepoInfo: %v", err)
	}
	if _, err := c.Files(context.Background(), repoID, ""); err != nil {
		t.Fatalf("Files: %v", err)
	}
	if err := c.Download(context.Background(), DownloadRequest{RepoID: repoID, Dest: t.TempDir()}); err != nil {
		t.Fatalf("Download: %v", err)
	}

	if want := "/api/models/org/repo#1"; sawInfo != want {
		t.Errorf("RepoInfo asked for %q, want %q", sawInfo, want)
	}
	if want := "/api/models/org/repo#1/tree/main"; sawTree != want {
		t.Errorf("the tree listing asked for %q, want %q", sawTree, want)
	}
	if want := "/org/repo#1/resolve/main/model.safetensors"; sawResolve != want {
		t.Errorf("the file download asked for %q, want %q", sawResolve, want)
	}
	if got, _ := c.ResolveURL(repoID, "main", "model.safetensors"); !strings.Contains(got, "org/repo%231/resolve") {
		t.Errorf("ResolveURL = %q, want the repo id percent-escaped", got)
	}
}

// A '.' or '..' is not a path element but an instruction about the path, and
// url.PathEscape leaves it alone, so escaping cannot make a repo id carrying
// one name the endpoint the caller asked for. No Hub repo id has such a
// segment, so every request path refuses it instead.
func TestARepoIDWithADotSegmentIsRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a request was made for %q; the repo id should have been refused first", r.URL.Path)
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	for _, id := range []string{"org/../../evil", "./org/repo", "org/repo/.."} {
		if _, err := c.RepoInfo(context.Background(), id); err == nil {
			t.Errorf("RepoInfo accepted repo id %q", id)
		}
		if _, err := c.Files(context.Background(), id, ""); err == nil {
			t.Errorf("Files accepted repo id %q", id)
		}
	}
}

// A relative next page is resolved against the page the request ended at, not
// the one it was sent to. The Hub redirects a re-cased or renamed repo id to
// its canonical URL, so the two differ, and resolving against the wrong one
// sends the second page request somewhere the first page never was.
func TestARelativeNextPageResolvesAgainstThePageThatCarriedIt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/ORG/Repo/tree/main", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "" {
			t.Errorf("the next page was fetched from %q, the URL the first request was SENT to, not the one it ended at", r.URL.String())
			fmt.Fprint(w, `[]`)
			return
		}
		http.Redirect(w, r, "/api/models/org/repo/tree/main?"+r.URL.RawQuery, http.StatusMovedPermanently)
	})
	mux.HandleFunc("/api/models/org/repo/tree/main", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			w.Header().Set("Link", `<?recursive=true&cursor=p2>; rel="next"`)
			fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
			return
		}
		fmt.Fprint(w, `[{"type":"file","path":"b.safetensors","size":2,"oid":"b"}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "ORG/Repo", "")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2 across both pages", len(files))
	}
}

// A next page continues the listing it came from: same path, a different
// cursor. Resolving a reference against the page that carried it means a
// reference that is not really a URL at all now resolves to *something* on the
// Hub's origin, so the origin rule alone no longer decides what gets followed
// with the bearer token attached.
func TestFilesRefusesANextPageThatIsNotAContinuationOfTheListing(t *testing.T) {
	cases := []struct {
		name string
		link string
	}{
		// Another repo's tree, spliced into the one that was asked for.
		{"another listing", `</api/models/other/repo/tree/main?cursor=p2>; rel="next"`},
		// A Link header value that is not a URL. nextPageURL splits on ',',
		// so what survives is a fragment of it that resolves against the page.
		{"junk that resolves", `<data:text/plain;base64,AAAA>; rel="next"`},
	}
	for _, tc := range cases {
		var secondRequest string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("cursor") != "" || !strings.HasSuffix(r.URL.Path, "/org/repo/tree/main") {
				secondRequest = r.URL.String()
				fmt.Fprint(w, `[{"type":"file","path":"b.safetensors","size":2,"oid":"b"}]`)
				return
			}
			w.Header().Set("Link", tc.link)
			fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
		}))

		c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
		files, err := c.Files(context.Background(), "org/repo", "")
		if err == nil {
			t.Errorf("%s: Files followed it and returned %d files", tc.name, len(files))
		}
		if secondRequest != "" {
			t.Errorf("%s: a second request was made for %q", tc.name, secondRequest)
		}
		srv.Close()
	}
}

// Userinfo in a next page is not part of the origin — url.URL.Host excludes it
// — so a next page can carry credentials past the origin check, and net/http
// turns them into an Authorization header of its own on a request that had
// none. It is dropped before the reference is used.
func TestANextPageCarriesNoUserinfo(t *testing.T) {
	var secondAuth string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "" {
			secondAuth = r.Header.Get("Authorization")
			fmt.Fprint(w, `[{"type":"file","path":"b.safetensors","size":2,"oid":"b"}]`)
			return
		}
		host := strings.TrimPrefix(srv.URL, "http://")
		w.Header().Set("Link", "<http://attacker:hunter2@"+host+r.URL.Path+"?recursive=true&cursor=p2>; rel=\"next\"")
		fmt.Fprint(w, `[{"type":"file","path":"a.safetensors","size":1,"oid":"a"}]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2 across both pages", len(files))
	}
	if secondAuth != "" {
		t.Errorf("the next-page request carried %q; a next page must not be able to put a header on it", secondAuth)
	}
}

// The rule that a "." or a ".." is not a path element belongs where the URL is
// built, not on the one value that happened to be looked at. A revision goes
// into the same path as the repo id, on the tree endpoint and on the resolve
// endpoint — and the resolve endpoint is the one with no origin check at all,
// because it must follow the Hub's redirect to its content store.
func TestARevisionWithADotSegmentIsRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a request was made for %q; the revision should have been refused first", r.URL.Path)
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	for _, rev := range []string{"..", "."} {
		if _, err := c.Files(context.Background(), "org/repo", rev); err == nil {
			t.Errorf("Files accepted revision %q", rev)
		}
		if got, err := c.ResolveURL("org/repo", rev, "model.safetensors"); err == nil {
			t.Errorf("ResolveURL accepted revision %q and built %q", rev, got)
		}
	}
}

// A missing value must not quietly produce a different, valid endpoint:
// leaving a part out shortens the path by one element, which is the same
// reshaping the escaping is there to prevent.
func TestResolveURLRefusesAMissingPart(t *testing.T) {
	c := &Client{BaseURL: "https://huggingface.co"}
	if got, err := c.ResolveURL("", "main", "model.safetensors"); err == nil {
		t.Errorf("ResolveURL built %q from an empty repo id", got)
	}
	if got, err := c.ResolveURL("org/repo", "main", ""); err == nil {
		t.Errorf("ResolveURL built %q from an empty file path", got)
	}
}

// A listing is bounded as a whole, not a page at a time. maxJSONBody caps one
// page and maxTreePages caps the hops, but their product is what a hostile Hub
// gets to spend: it pages forever, each page as large as one page may be, and
// the entries all land in one slice. Both halves of the whole-listing bound are
// exercised here — the bytes decoded across every page, and the number of
// entries held.
func TestFilesRefusesAListingThatIsOversizedAsAWhole(t *testing.T) {
	t.Run("bytes across pages", func(t *testing.T) {
		// One entry per page, its path a few MiB long: no single page is near
		// maxJSONBody, but enough pages run past what a whole listing may cost.
		page := `[{"type":"file","path":"` + strings.Repeat("a", 4<<20) + `","size":1,"oid":"a"}]`
		var pages int
		var mu sync.Mutex
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			pages++
			n := pages
			mu.Unlock()
			w.Header().Set("Link", fmt.Sprintf(`<?recursive=true&cursor=p%d>; rel="next"`, n+1))
			fmt.Fprint(w, page)
		}))
		defer srv.Close()

		c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
		_, err := c.Files(context.Background(), "org/repo", "")
		if !errors.Is(err, ErrOversizedBody) {
			t.Fatalf("Files err = %v, want ErrOversizedBody for a listing past the byte budget", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if want := maxTreeBytes/len(page) + 2; pages > want {
			t.Errorf("the hub served %d pages before the refusal; the budget should have stopped it by %d", pages, want)
		}
	})

	t.Run("entries held", func(t *testing.T) {
		// Small entries, so the byte budget is not what bites: a Hub that pages
		// short listings forever still may not make the client hold an
		// unbounded number of them.
		var b strings.Builder
		b.WriteString("[")
		for i := 0; i < 1000; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, `{"type":"file","path":"f%d.bin","size":1,"oid":"o%d"}`, i, i)
		}
		b.WriteString("]")
		page := b.String()

		var pages int
		var mu sync.Mutex
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			pages++
			n := pages
			mu.Unlock()
			w.Header().Set("Link", fmt.Sprintf(`<?recursive=true&cursor=p%d>; rel="next"`, n+1))
			fmt.Fprint(w, page)
		}))
		defer srv.Close()

		c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
		_, err := c.Files(context.Background(), "org/repo", "")
		if !errors.Is(err, ErrOversizedBody) {
			t.Fatalf("Files err = %v, want ErrOversizedBody for a listing past the entry cap", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if want := maxTreeEntries/1000 + 1; pages > want {
			t.Errorf("the hub served %d pages before the refusal; the entry cap should have stopped it by %d", pages, want)
		}
	})
}

// A real listing — many pages, ordinary entries — is still followed to its end.
// The whole-listing bound must refuse the hostile case without truncating the
// heavily-sharded repo the pagination exists for.
func TestFilesStillFollowsAnOrdinaryMultiPageListing(t *testing.T) {
	const pages = 12
	var seen int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen++
		n := seen
		mu.Unlock()
		if n < pages {
			w.Header().Set("Link", fmt.Sprintf(`<?recursive=true&cursor=p%d>; rel="next"`, n+1))
		}
		fmt.Fprintf(w, `[{"type":"file","path":"shard-%05d.safetensors","size":1,"oid":"o%d"}]`, n, n)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	files, err := c.Files(context.Background(), "org/repo", "")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != pages {
		t.Fatalf("got %d files, want %d across every page", len(files), pages)
	}
}

package hub

import (
	"context"
	"crypto/sha1"
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

// testCommit is the commit every fake Hub in this package reports as current.
const testCommit = "0123456789abcdef0123456789abcdef01234567"

// atCommit answers a download's commit lookup with testCommit and serves the
// listing and the files at that commit from the routes a fake already has for
// "main", so the fakes written before downloads were pinned to a commit keep
// describing one repository. The tests below that are about the pinning look
// at the raw paths themselves.
func atCommit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if r.URL.Query().Has("expand[]") && strings.HasPrefix(p, "/api/models/") {
			fmt.Fprintf(w, `{"id":%q,"sha":%q}`, strings.TrimPrefix(p, "/api/models/"), testCommit)
			return
		}
		r2 := r.Clone(r.Context())
		for _, pair := range [][2]string{
			{"/tree/" + testCommit, "/tree/main"},
			{"/resolve/" + testCommit + "/", "/resolve/main/"},
		} {
			r2.URL.Path = strings.Replace(r2.URL.Path, pair[0], pair[1], 1)
			r2.URL.RawPath = strings.Replace(r2.URL.RawPath, pair[0], pair[1], 1)
		}
		h.ServeHTTP(w, r2)
	})
}

// A download resolves the repository's commit once, then lists and fetches
// every file at that commit, never at a branch name: a commit that lands on
// the Hub part-way through a download must not leave a model that is half one
// version and half the next. The fake moves its current commit the moment the
// first file is asked for, which is exactly that race.
func TestDownloadFetchesEveryFileAtOneResolvedCommit(t *testing.T) {
	repo := standardRepo()
	const first = "1111111111111111111111111111111111111111"
	const second = "2222222222222222222222222222222222222222"

	var current atomic.Value
	current.Store(first)
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		switch {
		case r.URL.Path == "/api/models/org/repo":
			fmt.Fprintf(w, `{"id":"org/repo","sha":%q}`, current.Load().(string))
		case strings.HasPrefix(r.URL.Path, "/api/models/org/repo/tree/"):
			var entries []File
			for p, b := range repo {
				entries = append(entries, File{Path: p, Type: "file", Size: int64(len(b)), OID: "blob-" + p})
			}
			json.NewEncoder(w).Encode(entries)
		case strings.HasPrefix(r.URL.Path, "/org/repo/resolve/"):
			current.Store(second)
			rest := strings.TrimPrefix(r.URL.Path, "/org/repo/resolve/")
			name := rest[strings.Index(rest, "/")+1:]
			w.Write(repo[name])
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	snap, err := c.Download(context.Background(), DownloadRequest{RepoID: "org/repo", Dest: t.TempDir()})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if snap.Commit != first {
		t.Errorf("the snapshot names commit %q, want the one resolved before the download began (%q)", snap.Commit, first)
	}
	mu.Lock()
	defer mu.Unlock()
	listed, fetched := 0, 0
	for _, p := range paths {
		if strings.Contains(p, "/main") {
			t.Errorf("%s was requested at a branch name; every request must name the resolved commit", p)
		}
		if strings.Contains(p, second) {
			t.Errorf("%s was requested at the commit that landed mid-download", p)
		}
		if strings.Contains(p, "/tree/"+first) {
			listed++
		}
		if strings.Contains(p, "/resolve/"+first+"/") {
			fetched++
		}
	}
	if listed != 1 {
		t.Errorf("the listing was requested %d times at the resolved commit, want 1 (requests: %v)", listed, paths)
	}
	if fetched != len(repo) {
		t.Errorf("%d files were fetched at the resolved commit, want all %d (requests: %v)", fetched, len(repo), paths)
	}
}

// The snapshot is what the registry records as the version on disk: the
// commit, and each downloaded file's hash as the tree stated it — the LFS
// sha256 for a large file, the git blob id for a small one. A file the
// downloader skipped is not part of the version Dessau holds.
func TestDownloadReportsTheCommitAndEachFilesHash(t *testing.T) {
	repo := standardRepo()
	fh := newFakeHub(repo)
	fh.lfs = map[string]string{"model.safetensors": sha256Hex(repo["model.safetensors"])}
	fh.extra = map[string][]byte{"README.md": []byte("# hello"), "pytorch_model.bin": []byte("x")}
	srv := fh.server(t)

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	snap, err := c.Download(context.Background(), DownloadRequest{RepoID: "org/repo", Dest: t.TempDir()})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if snap.Commit != testCommit {
		t.Errorf("Commit = %q, want %q", snap.Commit, testCommit)
	}
	if got, want := snap.Files["model.safetensors"], sha256Hex(repo["model.safetensors"]); got != want {
		t.Errorf("the weights are recorded as %q, want their LFS sha256 %q", got, want)
	}
	if got, want := snap.Files["config.json"], gitBlobID(repo["config.json"]); got != want {
		t.Errorf("config.json is recorded as %q, want its git blob id %q", got, want)
	}
	if _, ok := snap.Files["pytorch_model.bin"]; ok {
		t.Error("a file the downloader skipped is recorded as part of the version")
	}
	if len(snap.Files) != len(repo)+1 { // README.md is wanted: the downloader keeps Markdown
		t.Errorf("the snapshot records %d files, want %d: %v", len(snap.Files), len(repo)+1, snap.Files)
	}
}

// What the Hub names as the commit is spliced into every URL the download
// makes and recorded as the version on disk, so anything that is not a full
// commit id is refused before a single file is asked for.
func TestDownloadRefusesACommitThatIsNotACommitID(t *testing.T) {
	for _, bad := range []string{"", "main", "../../etc", "0123456789abcdef", "0123456789ABCDEF0123456789ABCDEF01234567", "0123456789abcdef0123456789abcdef0123456/"} {
		t.Run(bad, func(t *testing.T) {
			var files atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/models/org/repo" {
					fmt.Fprintf(w, `{"id":"org/repo","sha":%q}`, bad)
					return
				}
				files.Add(1)
				http.NotFound(w, r)
			}))
			defer srv.Close()
			c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
			_, err := c.Download(context.Background(), DownloadRequest{RepoID: "org/repo", Dest: t.TempDir()})
			if !errors.Is(err, ErrBadCommit) {
				t.Fatalf("err = %v, want ErrBadCommit", err)
			}
			if n := files.Load(); n != 0 {
				t.Errorf("%d further requests were made after a bad commit", n)
			}
		})
	}
}

// A caller that names the revision is held to the same shape: a branch name
// is not a version.
func TestDownloadRefusesANamedRevisionThatIsNotACommitID(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1"}
	_, err := c.Download(context.Background(), DownloadRequest{RepoID: "org/repo", Revision: "main", Dest: t.TempDir()})
	if !errors.Is(err, ErrBadCommit) {
		t.Fatalf("err = %v, want ErrBadCommit", err)
	}
}

// gitBlobID is the id git gives a file's content, which is what the Hub's tree
// reports as the oid of a file the repository stores in git.
func gitBlobID(b []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(b))
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func TestValidCommit(t *testing.T) {
	if !ValidCommit(testCommit) {
		t.Errorf("%s is a commit id", testCommit)
	}
	for _, bad := range []string{"", "main", testCommit[:39], testCommit + "0", strings.ToUpper(testCommit), "g123456789abcdef0123456789abcdef01234567"} {
		if ValidCommit(bad) {
			t.Errorf("ValidCommit(%q) = true", bad)
		}
	}
}

package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/Gropius/internal/config"
)

// A query typed as a full repository id is looked up exactly, whatever account
// owns it: the picker's search is scoped to one organisation, so a conversion
// published under somebody else's account is otherwise unfindable from the
// panel. A plain search term is not an id and must not cost that extra request.
func TestSearchFindsAModelTypedAsAFullRepoID(t *testing.T) {
	const repoID = "prism-ml/Ternary-Bonsai-2-27B-mlx-2bit"

	var mu sync.Mutex
	var asked []string
	hubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.URL.Path)
		mu.Unlock()
		switch {
		case r.URL.Path == "/api/models":
			// The author-scoped search knows nothing about this repo.
			fmt.Fprint(w, `[]`)
		case r.URL.Path == "/api/models/"+repoID:
			fmt.Fprintf(w, `{"id":%q,"tags":["mlx","safetensors"],"pipeline_tag":"text-generation"}`, repoID)
		case strings.HasPrefix(r.URL.Path, "/api/models/"+repoID+"/tree/"):
			fmt.Fprint(w, `[{"path":"model.safetensors","size":1048576}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer hubSrv.Close()

	srv, a := newTestControlApp(t, config.Default())
	a.Hub.BaseURL = hubSrv.URL

	search := func(q string) []string {
		t.Helper()
		resp, err := srv.Client().Get(srv.URL + "/api/search?q=" + url.QueryEscape(q))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
			Hidden int `json:"hidden"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out.Hidden > 0 {
			t.Fatalf("searching %q hid %d results; this Mac cannot measure the fixture", q, out.Hidden)
		}
		ids := make([]string, 0, len(out.Results))
		for _, r := range out.Results {
			ids = append(ids, r.ID)
		}
		return ids
	}

	if ids := search(repoID); len(ids) == 0 || ids[0] != repoID {
		t.Errorf("searching for the full repo id returned %v, want %s offered first", ids, repoID)
	}

	mu.Lock()
	asked = nil
	mu.Unlock()
	search("Ternary")
	mu.Lock()
	defer mu.Unlock()
	for _, p := range asked {
		if p == "/api/models/"+repoID {
			t.Errorf("a plain search term asked the Hub for %s; the exact lookup runs only for a query that is a repo id", p)
		}
	}
}

// The exact lookup adds a result or it adds nothing: a repo that is not there,
// a Hub that answers with something MLX cannot load, and a query that is not an
// id all leave the author-scoped search exactly as it was.
func TestSearchIgnoresAnExactLookupThatIsNotAnMLXModel(t *testing.T) {
	const repoID = "prism-ml/Ternary-Bonsai-2-27B"

	hubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/models":
			fmt.Fprint(w, `[]`)
		case r.URL.Path == "/api/models/"+repoID:
			// A real repo, but PyTorch weights: nothing here will load.
			fmt.Fprintf(w, `{"id":%q,"tags":["transformers","pytorch"]}`, repoID)
		default:
			http.NotFound(w, r)
		}
	}))
	defer hubSrv.Close()

	srv, a := newTestControlApp(t, config.Default())
	a.Hub.BaseURL = hubSrv.URL

	resp, err := srv.Client().Get(srv.URL + "/api/search?q=" + url.QueryEscape(repoID))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 0 {
		t.Errorf("results = %v, want none: the repo carries no MLX weights", out.Results)
	}
}

// The typed text becomes a URL path segment, so only a well-formed id is ever
// looked up. Everything else — a URL, a traversal, a term with a space — is a
// search term and nothing more.
func TestOnlyAWellFormedRepoIDIsLookedUpExactly(t *testing.T) {
	cases := []struct {
		q    string
		want bool
	}{
		{"prism-ml/Ternary-Bonsai-2-27B-mlx-2bit", true},
		{"  prism-ml/Ternary-Bonsai-2-27B-mlx-2bit  ", true},
		{"qwen3", false},
		{"author:prism-ml Ternary", false},
		{"prism-ml/Ternary Bonsai", false},
		{"prism-ml/", false},
		{"/Ternary", false},
		{"a/b/c", false},
		{"a/..", false},
		{"../../etc/passwd", false},
		{"https://huggingface.co/prism-ml/Ternary", false},
		{"prism-ml/Ternary?x=1", false},
		{"prism-ml/Tern#ary", false},
		{"", false},
	}
	for _, tc := range cases {
		got, ok := exactRepoID(tc.q)
		if ok != tc.want {
			t.Errorf("exactRepoID(%q) = (%q, %v), want ok=%v", tc.q, got, ok, tc.want)
			continue
		}
		if ok && !config.ValidRepoID(got) {
			t.Errorf("exactRepoID(%q) returned %q, which is not a valid repo id", tc.q, got)
		}
	}
}

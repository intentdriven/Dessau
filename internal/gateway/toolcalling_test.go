package gateway

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
)

// The models list carries the tool-call verdict on every ready entry as one
// of three words: an unprobed model and one whose verdict is stale both
// read unknown — never no — and the two recorded verdicts read yes and no.
// It goes to every client, keyed or not, on or off this Mac, because it
// says what a model IS rather than what this Mac is doing.
func TestModelsListCarriesTheToolCallVerdictOnEveryReadyEntry(t *testing.T) {
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m"})
	defer fake.Close()
	models := &stubModels{models: []registry.Model{
		{RepoID: "org/unprobed", State: registry.StateReady},
		{RepoID: "org/stale", State: registry.StateReady,
			ToolCalling: &registry.ToolCalling{Can: true, At: 1, Runtime: "0.30.0", Stale: registry.StaleRuntime}},
		{RepoID: "org/yes", State: registry.StateReady,
			ToolCalling: &registry.ToolCalling{Can: true, At: 1, Runtime: "0.31.3"}},
		{RepoID: "org/no", State: registry.StateReady,
			ToolCalling: &registry.ToolCalling{Can: false, At: 1, Runtime: "0.31.3"}},
		{RepoID: "org/downloading", State: registry.StateDownloading,
			ToolCalling: &registry.ToolCalling{Can: true, At: 1, Runtime: "0.31.3"}},
	}}
	g := New(Options{Config: config.Default(), Pool: &stubPool{srv: fake}, Models: models})

	// From off this Mac with no key: the listing an open server serves the
	// network, which withholds residency and nothing else.
	entries, body := listModelsEntriesFrom(t, g.Handler(), "", "203.0.113.50:9999")
	assertNoResidency(t, "a network client", body)
	byID := map[string]map[string]any{}
	for _, e := range entries {
		byID[e["id"].(string)] = e
	}
	if len(byID) != 4 {
		t.Fatalf("listed %v, want the four ready models", byID)
	}
	for id, want := range map[string]string{
		"org/unprobed": "unknown", "org/stale": "unknown", "org/yes": "yes", "org/no": "no",
	} {
		if got := byID[id]["tool_calling"]; got != want {
			t.Errorf("%s: tool_calling = %v, want %q", id, got, want)
		}
	}
	// The verdict's own fields stay on the registry: the wire carries the
	// word and nothing of when or under what.
	if strings.Contains(body, `"can"`) || strings.Contains(body, "0.31.3") {
		t.Errorf("the verdict's record reached the wire: %s", body)
	}
}

// The field is informative and nothing else: a request carrying tools for a
// model recorded as unable is relayed unchanged and answered.
func TestARequestWithToolsForAModelRecordedUnableIsRelayedUnchanged(t *testing.T) {
	const modelPath = "/models/org/no"
	fake := mlxtest.Start(mlxtest.Options{ModelArg: modelPath, Reply: "DESSAU OK"})
	defer fake.Close()
	models := &stubModels{models: []registry.Model{{
		RepoID: "org/no", Path: modelPath, State: registry.StateReady,
		ToolCalling: &registry.ToolCalling{Can: false, At: 1, Runtime: "0.31.3"},
	}}}
	g := New(Options{Config: config.Default(), Pool: &stubPool{srv: fake}, Models: models})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()

	if entry := firstModelEntry(t, srv); entry["tool_calling"] != "no" {
		t.Fatalf("tool_calling = %v, want no — this test is about a model recorded as unable", entry["tool_calling"])
	}
	tools := []any{map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":       "lookup",
			"parameters": map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}},
		},
	}}
	resp := post(t, srv, "/v1/chat/completions", map[string]any{
		"model":       "org/no",
		"messages":    []any{map[string]string{"role": "user", "content": "hi"}},
		"tools":       tools,
		"tool_choice": "auto",
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a request with tools for a model recorded as unable was refused: status %d", resp.StatusCode)
	}
	body := fake.LastBody()
	got, _ := body["tools"].([]any)
	if len(got) != 1 {
		t.Fatalf("upstream saw tools = %v, want the client's one tool relayed", body["tools"])
	}
	fn, _ := got[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "lookup" || body["tool_choice"] != "auto" {
		t.Errorf("upstream saw tools %v and tool_choice %v, want the client's relayed unchanged", got, body["tool_choice"])
	}
	if got := fake.LastModelField(); got != modelPath {
		t.Errorf("upstream saw model=%q, want the backend path %q", got, modelPath)
	}
}

// Nothing on the completions path reads the field: the relay does not
// consult it, refuse on it or rewrite tools because of it. Held by a scan:
// tool_calling, in either spelling, is named nowhere in this package's
// shipping source outside handleListModels.
func TestTheToolCallVerdictIsNamedNowhereOutsideTheModelsList(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	spellings := []string{"tool_calling", "ToolCalling"}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		// The one admitted range: the body of handleListModels, in this file
		// or any other of the package.
		var admitted [2]int
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Name.Name != "handleListModels" || fd.Body == nil {
				continue
			}
			admitted = [2]int{fset.Position(fd.Body.Pos()).Offset, fset.Position(fd.Body.End()).Offset}
		}
		for _, spelling := range spellings {
			for at := 0; ; {
				i := strings.Index(string(src[at:]), spelling)
				if i < 0 {
					break
				}
				at += i
				if at < admitted[0] || at >= admitted[1] {
					t.Errorf("%s names %s at %s, outside handleListModels — the field is informative, and nothing on the completions path may read it",
						filepath.Base(name), spelling, fset.Position(f.Pos()+token.Pos(at)))
				}
				at += len(spelling)
			}
		}
	}
}

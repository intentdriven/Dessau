package archtest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// Nothing reachable from /v1 can delete a model (adr-2610031153127219: "No
// DELETE is registered on /v1, and nothing reachable from /v1 can delete a
// model's files"). The /v1 handlers reach the pool and the registry only
// through the gateway's own Pool and Models interfaces, so the rule is held
// on those: neither may carry a method that removes a model, and the /v1
// route table registers no DELETE. The control plane's delete is the App's,
// on its own loopback-only routes, and is not reachable through either.
func TestNothingReachableFromV1CanDeleteAModel(t *testing.T) {
	path := filepath.Join("..", "gateway", "gateway.go")
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	removing := map[string]bool{"Remove": true, "Delete": true, "RemoveAll": true, "Purge": true}
	seen := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			it, ok := x.Type.(*ast.InterfaceType)
			if !ok || (x.Name.Name != "Pool" && x.Name.Name != "Models") {
				return true
			}
			seen[x.Name.Name] = true
			for _, m := range it.Methods.List {
				for _, name := range m.Names {
					if removing[name.Name] {
						t.Errorf("the gateway's %s interface carries %s, which /v1 could then reach", x.Name.Name, name.Name)
					}
				}
			}
		case *ast.FuncDecl:
			if x.Name.Name != "routes" || x.Recv == nil {
				return true
			}
			ast.Inspect(x.Body, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if ok && lit.Kind == token.STRING && strings.HasPrefix(strings.Trim(lit.Value, "\"`"), "DELETE") {
					t.Errorf("the /v1 route table registers %s", lit.Value)
				}
				return true
			})
		}
		return true
	})
	if !seen["Pool"] || !seen["Models"] {
		t.Fatal("the gateway's Pool and Models interfaces were not found, so nothing is held")
	}
}

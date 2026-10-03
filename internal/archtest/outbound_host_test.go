package archtest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// outboundHosts is every host a non-test Go file in this repository names in
// a URL it can dial, with the reason it is there. adr-2609201008476813 names
// "the architecture test that refuses any outbound host other than the ones
// the product needs"; this is that test (iss-2610030913170591). A new host is
// a new party this Mac talks to, so adding one has to be deliberate: here,
// with its reason, beside a record that says why.
var outboundHosts = map[string]string{
	"huggingface.co":     "model downloads and the opt-in update check (internal/hub)",
	"github.com":         "the pinned uv release at first-run provisioning, and Dessau's own release at an update the operator starts",
	"discord.com":        "the Discord bridge's REST calls, opt-in (adr-2609181004167097)",
	"gateway.discord.gg": "the Discord bridge's gateway connection, opt-in (adr-2609181004167097)",
	"raw.githubusercontent.com": "the install command printed for the person to run (internal/lifecycle); " +
		"Dessau itself never fetches it",
}

// localAuthority is an authority that never leaves this Mac or names no real
// host at all: loopback, the unspecified address written into a sample
// address, a reserved .invalid name a socket transport answers under, the
// site generator's placeholder, and a bare scheme prefix being matched.
var localAuthority = regexp.MustCompile(`^(|…|0|localhost|127\.0\.0\.1|\[+::1\]+|[a-z0-9.-]+\.invalid)(:[^/]*)?$`)

// urlLiteral finds a URL's authority in a string literal's value; only string
// literals are read, so a URL in a comment is prose, not something dialled.
var urlLiteral = regexp.MustCompile(`(?:https?|wss?)://([^"/?#\s]*)`)

func TestEveryOutboundHostIsOneTheProductNeeds(t *testing.T) {
	root := repoRootDir(t)
	used := map[string]bool{}
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			var literals []string
			ast.Inspect(file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if v, err := strconv.Unquote(lit.Value); err == nil {
						literals = append(literals, v)
					}
				}
				return true
			})
			for _, m := range urlLiteral.FindAllStringSubmatch(strings.Join(literals, "\n"), -1) {
				authority := m[1]
				if localAuthority.MatchString(authority) {
					continue
				}
				host := authority
				if i := strings.LastIndex(host, ":"); i >= 0 && !strings.Contains(host, "]") {
					host = host[:i]
				}
				if _, ok := outboundHosts[host]; !ok {
					t.Errorf("%s names the outbound host %q, which is not one the product needs: "+
						"add it to outboundHosts with its reason only beside a record that says why "+
						"(adr-2609201008476813)", rel, host)
					continue
				}
				used[host] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for host := range outboundHosts {
		if !used[host] {
			t.Errorf("outboundHosts lists %q, which no file names any more: remove it, so the list stays the product's", host)
		}
	}
}

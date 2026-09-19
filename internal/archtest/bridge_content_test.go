package archtest_test

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// bridgePackage is the one bridge there is. Its source is walked rather than
// its behaviour exercised, because what is being held is a property of every
// line the package will ever write, including the ones a future change adds.
const bridgePackage = "internal/bridge/discord"

// bridgeLogKeys is every attribute key a bridge log line may carry.
//
// adr-2609181004167097 condition 4 decides this list: the log records the
// FACT of a bridged request and never its content — the bridge's name, the
// platform's channel and user identifiers as opaque numbers, the model, the
// sizes and the timing. The reasons a connection failed are the operator's
// too, and are here for the same reason the gateway's refusal log is: an
// operator diagnosing a bridge that will not connect has nowhere else to
// look.
//
// AN ENTRY HERE COSTS SOMETHING. A key added to this list is a new thing the
// bridge may write down about a conversation it is relaying, and the whole
// point of the list is that adding one is a line in a diff somebody reads.
var bridgeLogKeys = map[string]string{
	"bridge":       "which bridge wrote the line; one constant",
	"channel":      "the platform's channel identifier, as an opaque number",
	"user":         "the platform's user identifier, as an opaque number",
	"model":        "the repo id this Mac resolved, never a name a stranger chose",
	"prompt_bytes": "the size of the encoded request — a count, not content",
	"answer_bytes": "the size of the answer — a count, not content",
	"duration_ms":  "how long the request took",
	"reason":       "why the bridge stopped, in the bridge's own words",
	"err":          "a transport or protocol failure, for the operator",
	"in":           "how long until the next reconnect attempt",
}

// bridgeContentNames are the identifiers in that package that hold a message
// or an answer. A log value that names one of them is content on its way into
// a file.
//
// The list is the package's own vocabulary, so it is short and it is checked
// for liveness below: a name that no longer exists exempts nothing and would
// leave this scan matching a word the package stopped using.
var bridgeContentNames = []string{
	"Content", "Messages", "text", "answer", "delta", "turns", "pending", "sent",
}

// Every log line the bridge writes carries counts, classes and identifiers,
// and nothing of the message it is relaying or the answer it is posting.
//
// This is the second half of the grant in adr-2609181004167097 condition 4.
// The first half — that the bridge READS both sides — is admitted by name in
// promptContentReaders. What it may do with what it reads is this: nothing
// reaches a log line, which is the same rule every other record in this
// project follows.
func TestTheBridgeWritesNoMessageContent(t *testing.T) {
	fset := token.NewFileSet()
	files := bridgeSourceFiles(t)
	if len(files) < 4 {
		t.Fatalf("the scan found %d source files in %s, so it is reading the wrong place", len(files), bridgePackage)
	}
	lines := 0
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		rel := filepath.ToSlash(filepath.Join(bridgePackage, filepath.Base(path)))
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isLogCall(call) {
				return true
			}
			lines++
			where := rel + ":" + strconv.Itoa(fset.Position(call.Pos()).Line)
			if len(call.Args) == 0 {
				return true
			}
			if _, ok := call.Args[0].(*ast.BasicLit); !ok {
				t.Errorf("%s writes a log line whose message is not a literal, so what it says is "+
					"decided somewhere this scan cannot see", where)
				return true
			}
			args := call.Args[1:]
			if len(args)%2 != 0 {
				t.Errorf("%s writes an odd number of attributes, so a value is being read as a key", where)
				return true
			}
			for i := 0; i < len(args); i += 2 {
				key, ok := stringLit(args[i])
				if !ok {
					t.Errorf("%s writes an attribute whose key is not a literal", where)
					continue
				}
				if _, allowed := bridgeLogKeys[key]; !allowed {
					t.Errorf("%s writes the attribute %q, which is not one the bridge's log may carry. "+
						"adr-2609181004167097 condition 4 holds the log to the fact of a bridged request "+
						"and never its content; if this is genuinely a new fact about a request, add it to "+
						"bridgeLogKeys with the reason", where, key)
				}
				if name := namesContent(args[i], args[i+1]); name != "" {
					t.Errorf("%s logs %s under %q, which is the message being relayed or the answer being "+
						"posted. The bridge reads both and writes down neither (adr-2609181004167097 "+
						"condition 4)", where, name, key)
				}
			}
			return true
		})
	}
	if lines < 5 {
		t.Fatalf("the scan found %d log lines in the bridge, so it is not reading its log calls", lines)
	}
}

// And the one line the ADR asks for, positively: a bridged request is recorded
// with the bridge, the identifiers, the model, the sizes and the timing. A
// scan that only forbids things passes a bridge that logs nothing at all.
func TestTheBridgeRecordsTheFactOfABridgedRequest(t *testing.T) {
	var src strings.Builder
	for _, path := range bridgeSourceFiles(t) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	body := src.String()
	if !strings.Contains(body, `"bridged a request"`) {
		t.Fatal("the bridge writes no line for a bridged request; the log is how an operator sees that " +
			"a conversation is leaving this Mac (adr-2609181004167097 condition 4)")
	}
	line := body[strings.Index(body, `"bridged a request"`):]
	if end := strings.Index(line, "\n\n"); end > 0 {
		line = line[:end]
	}
	for _, key := range []string{"bridge", "channel", "user", "model", "prompt_bytes", "answer_bytes", "duration_ms"} {
		if !strings.Contains(line, `"`+key+`"`) {
			t.Errorf("the bridged-request line does not carry %q", key)
		}
	}
}

// The lists above are only a boundary while every entry on one is live: a key
// nothing writes and a name nothing holds both exempt nothing, and a renamed
// field would leave the exemption behind for the next one to inherit. This is
// the rule TestStatisticsSwitchReadersAllExist keeps for its own list.
func TestTheBridgesContentNamesAllExist(t *testing.T) {
	var src strings.Builder
	for _, path := range bridgeSourceFiles(t) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	body := src.String()
	for _, name := range bridgeContentNames {
		if !strings.Contains(body, name) {
			t.Errorf("bridgeContentNames watches for %q, which %s no longer holds — remove it rather "+
				"than leaving a scan that matches nothing", name, bridgePackage)
		}
	}
}

// isLogCall reports whether a call is a structured log call.
func isLogCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Debug", "Info", "Warn", "Error":
	default:
		return false
	}
	// A method on something called log — the bridge reaches its logger as
	// b.log, s.bridge.log or r.log, and nothing else in the package is spelled
	// that way.
	return strings.Contains(exprString(sel.X), "log")
}

// namesContent reports the content-bearing name a log value reaches for, or
// "" when it reaches for none.
//
// A length is not content and is allowed: len(body) is how big a request was,
// which is exactly the kind of fact the log is for. Everything else that names
// one of the package's content variables is refused.
func namesContent(key, value ast.Expr) string {
	_ = key
	var found string
	ast.Inspect(value, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "len" {
				return false // a count of it, not it
			}
		}
		var name string
		switch v := n.(type) {
		case *ast.Ident:
			name = v.Name
		case *ast.SelectorExpr:
			name = v.Sel.Name
		default:
			return true
		}
		for _, banned := range bridgeContentNames {
			if name == banned {
				found = name
				return false
			}
		}
		return true
	})
	return found
}

func exprString(e ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, token.NewFileSet(), e)
	return b.String()
}

// bridgeSourceFiles is the bridge's shipping Go files. Test files are
// excluded: a test may compose whatever conversation it needs, and may log it.
func bridgeSourceFiles(t *testing.T) []string {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(bridgePackage)))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, filepath.Join(repoRoot, filepath.FromSlash(bridgePackage), name))
	}
	return out
}

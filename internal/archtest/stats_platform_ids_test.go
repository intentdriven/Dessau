package archtest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"testing"

	"github.com/intentdriven/Dessau/internal/gateway"
	"github.com/intentdriven/Dessau/internal/stats"
)

// statsRecordFields is every field a request record carries, and where its
// value comes from. statsEventFields is the same for a load, removal or
// footprint event.
//
// The maintainer's decision of 2026-09-20 (iss-2609201007355356) holds the
// statistics store free of every identifier a platform supplied: a bridged
// request's channel and user identifiers go to the bridge's log line only
// (cond-2609181018498110), and the record says only that the request was
// bridged, in its source. A Discord identifier is a number, so a field of
// any type could carry one; the list is therefore of every field, not of the
// text-bearing ones.
//
// AN ENTRY HERE COSTS SOMETHING. A field added to either list is a new thing
// written down about every request, kept on disk and read by other people's
// tools, and the point of the list is that adding one is a line in a diff
// somebody reads with this rule beside it.
var statsRecordFields = map[string]string{
	"model":                   "the repo id the request resolved to on this Mac, never a name a client sent",
	"at":                      "when the request arrived",
	"class":                   "how it ended, one of the fixed classes",
	"source":                  "how it reached this Mac, one of the fixed sources; a bridged request says bridge and nothing about where",
	"streamed":                "whether the client asked for a stream",
	"prompt_tokens":           "the model server's count",
	"completion_tokens":       "the model server's count",
	"first_token_ms":          "a time",
	"duration_ms":             "a time",
	"queue_wait_ms":           "a time",
	"load_wait_ms":            "a time",
	"declared_context":        "the model's declared window",
	"served_context":          "the model's served window",
	"estimated_prompt_tokens": "the gateway's estimate of the prompt's size",
	"requested_tokens":        "the figure the served-window check judged",
	"in_flight":               "how many requests the model already had",
	"overrides":               "names from the closed SamplingParameters set, filtered at the recorder",
	"footprint_bytes":         "the model server's sampled footprint",
}

var statsEventFields = map[string]string{
	"at":          "when it happened",
	"model":       "the repo id on this Mac",
	"kind":        "one of the fixed event kinds",
	"reason":      "one of the pool's fixed removal reasons",
	"by":          "one of the fixed caller kinds, never a name",
	"duration_ms": "how long a load took",
	"failed":      "whether a load never became ready",
	"sampling":    "the launch's sampling values by parameter name",
	"bytes":       "the sampled footprint",
}

func TestTheStatisticsStoreCarriesNoPlatformIdentifier(t *testing.T) {
	for _, c := range []struct {
		what    string
		fields  []string
		allowed map[string]string
	}{
		{"request record", stats.RecordFields(), statsRecordFields},
		{"event", stats.EventFields(), statsEventFields},
	} {
		seen := map[string]bool{}
		for _, f := range c.fields {
			seen[f] = true
			if _, ok := c.allowed[f]; !ok {
				t.Errorf("a statistics %s carries %q, which this test does not know. If it could hold "+
					"an identifier a platform supplied — a channel, user, server or message id — it "+
					"belongs on the bridge's log line instead (iss-2609201007355356); otherwise add it "+
					"here with where its value comes from", c.what, f)
			}
		}
		for f := range c.allowed {
			if !seen[f] {
				t.Errorf("the list for a statistics %s names %q, which it no longer carries; drop it so "+
					"the list stays the whole of what is written", c.what, f)
			}
		}
	}
}

// askRequestFields is every field an in-process request carries into the
// gateway, which is the only way a bridge reaches the statistics store. The
// conversation goes in Body, which the recorder never reads; nothing else a
// bridge holds has a field to travel in.
var askRequestFields = []string{"Body", "Model", "OnEvent", "Source"}

func TestABridgedRequestHasNoFieldForAPlatformIdentifier(t *testing.T) {
	typ := reflect.TypeOf(gateway.AskRequest{})
	var got []string
	for i := 0; i < typ.NumField(); i++ {
		got = append(got, typ.Field(i).Name)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, askRequestFields) {
		t.Errorf("gateway.AskRequest has fields %v, want %v. A field added here travels from a bridge "+
			"to the request's record; a platform's identifiers go to the bridge's log line only "+
			"(iss-2609201007355356)", got, askRequestFields)
	}
	// And the bridge sets only those, by name.
	fset := token.NewFileSet()
	found := false
	for _, path := range bridgeSourceFiles(t) {
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || exprString(lit.Type) != "gateway.AskRequest" {
				return true
			}
			found = true
			for _, el := range lit.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					t.Errorf("%s: an AskRequest built positionally; name its fields so this test can read them",
						fset.Position(lit.Pos()))
					continue
				}
				if k := exprString(kv.Key); k == "Source" && exprString(kv.Value) != "stats.SourceBridge" {
					t.Errorf("%s: the bridge's request names its source as %s, want stats.SourceBridge",
						fset.Position(kv.Pos()), exprString(kv.Value))
				}
			}
			return true
		})
	}
	if !found {
		t.Error("the bridge builds no gateway.AskRequest; this test then holds nothing")
	}
}

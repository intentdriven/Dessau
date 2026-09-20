package archtest_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// The tool-call probe appears in no request statistic and in no request log
// line, and it does so by construction rather than by exemption: it is not a
// client of the gateway, and it counts nothing itself. The first half is an
// import rule — the package sees neither the gateway nor the statistics
// recorder — and the second is where its one request goes: to the model
// server's own address, the one the pool handed it, never to this Mac's own
// endpoint (spc-2609201451069488 criterion 5).
const toolProbePkg = "github.com/intentdriven/Dessau/internal/toolprobe"

func TestTheToolCallProbeIsNotAClientOfTheGatewayAndCountsNothing(t *testing.T) {
	for _, dep := range depsOf(t, toolProbePkg) {
		switch dep {
		case "github.com/intentdriven/Dessau/internal/gateway":
			t.Errorf("%s depends on the gateway — its request would then be a request, counted and logged like a client's", toolProbePkg)
		case "github.com/intentdriven/Dessau/internal/stats":
			t.Errorf("%s depends on the statistics recorder — nothing on the probe's path may count", toolProbePkg)
		}
	}

	root := repoRootDir(t)
	src := readRepoFile(t, root, filepath.Join("internal", "toolprobe", "probe.go"))
	// The request is built against the upstream's own base URL, which is
	// the address the pool handed the probe for that model's server.
	if !strings.Contains(src, `up.BaseURL+"/v1/chat/completions"`) {
		t.Error("internal/toolprobe/probe.go does not build its request against the upstream's own BaseURL — the one address that reaches the model server without passing the gateway")
	}
	// And nothing in it reaches for this Mac's own endpoint: no bind port,
	// no API key, no Authorization header.
	for _, spelling := range []string{"BindPort", "APIKey", "Authorization", "Endpoint("} {
		if strings.Contains(src, spelling) {
			t.Errorf("internal/toolprobe/probe.go names %s, which is how a request would reach Dessau's own endpoint rather than the model server", spelling)
		}
	}
}

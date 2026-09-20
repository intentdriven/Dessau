package app

import (
	"context"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/toolprobe"
)

// toolProbeSource is the identity the tool-call probe's hold carries
// (runtime.WithSource): its own, never a client's. The probe never loads, so
// it never joins the load-waiter queue the tag shares out; the tag is what
// names the hold in the pool's own view.
const toolProbeSource = "dessau-tool-probe"

// toolProbeSources is what the tool-call probe sees of the app. Like
// selfTestServer it holds none of the app's locks across a call
// (adr-2609091239058072): each method takes the pool's or the registry's own
// lock for one read.
type toolProbeSources struct{ a *App }

// Resident reads the pool's view of the model: loaded and answering, and how
// many requests it has in flight. A model the pool is not holding, or one
// still loading, is not resident to the probe.
func (s toolProbeSources) Resident(repoID string) (bool, int) {
	key := config.FoldRepoID(repoID)
	for _, r := range s.a.Pool.Resident() {
		if config.FoldRepoID(r.RepoID) == key {
			return r.State == runtime.ResidencyLoaded, r.InFlight
		}
	}
	return false, 0
}

// Acquire is the pool's ordinary Acquire under the probe's identity and a
// soft hold: a client whose load needs the memory takes the model, and the
// pool cancels the probe's request to get it — the same acquisition the
// self-test takes, so nothing is evicted underneath the probe and the probe
// evicts nothing. The probe asks Resident first; a model that is not loaded
// is never acquired, since the pool's Acquire would load it.
func (s toolProbeSources) Acquire(ctx context.Context, repoID string) (toolprobe.Upstream, func(), error) {
	ctx, yield := context.WithCancel(ctx)
	poolCtx := runtime.WithSoftHold(runtime.WithSource(ctx, toolProbeSource), yield)
	up, release, err := s.a.Pool.Acquire(poolCtx, repoID)
	if err != nil {
		yield()
		return toolprobe.Upstream{}, nil, err
	}
	return toolprobe.Upstream{BaseURL: up.BaseURL, ModelArg: up.ModelArg}, func() {
		release()
		yield()
	}, nil
}

func (s toolProbeSources) Runtime() string { return runtime.MLXLMVersion() }

func (s toolProbeSources) Save(repoID string, tc *registry.ToolCalling) error {
	return s.a.Registry.SetToolCalling(repoID, tc)
}

// modelLoaded is what the pool's observer tells the app when a model server
// reaches loaded: a model with no current tool-call verdict — none, or one
// taken under another runtime — is queued for one probe, after the request
// that loaded it has been served. A model with a current verdict is asked
// nothing (itd-2609201445423499).
func (a *App) modelLoaded(repoID string) {
	m, err := a.Registry.Get(repoID)
	if err != nil {
		return
	}
	if m.ToolCalling.Current() {
		return
	}
	a.ToolProbe.Enqueue(m.RepoID)
}

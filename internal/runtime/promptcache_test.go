package runtime

import (
	"context"
	"strconv"
	"testing"
)

// The model server keeps prompts' caches after their answers are sent, and
// left to its own defaults it keeps ten of them with no limit on their size,
// so a model serving several long conversations held several windows of cache
// its charge does not count (iss-2610071035130302). The maintainer's bound is
// one served window of real, uncharged cache for one sequence: the charge
// already reserves a charged window per sequence, so the kept caches sit
// inside it and the charge is not raised.
//
// The pool hands the launcher both figures, and the launcher passes the
// product as Dessau's own flag — which serve.py reads and hands to the
// prompt cache's constructor — with the entry count spelled out rather than
// left to the server's default.
func TestThePromptCacheIsBoundedToOneServedWindowOfRealCache(t *testing.T) {
	const (
		window   = 32768
		perToken = 24576 // real bytes a token; the charge is five times this
	)
	l := newFakeLauncher()
	src := &fakeSource{
		models: map[string]int64{"org/a": 1 << 20},
		facts: map[string]ResolvedModel{"org/a": {
			ServedContext: window, KVChargePerToken: perToken * 5, KVBytesPerToken: perToken,
		}},
	}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 40})
	_, release, err := p.Acquire(context.Background(), "org/a")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	release()

	spec := l.specFor("org/a")
	if spec.KVBytesPerToken != perToken || spec.ServedContext != window {
		t.Fatalf("the launch carries %d bytes a token at a window of %d, want %d at %d",
			spec.KVBytesPerToken, spec.ServedContext, perToken, window)
	}
	argv := launchArgs(spec)
	if got, ok := flagValue(argv, promptCacheBytesFlag); !ok || got != strconv.Itoa(perToken*window) {
		t.Errorf("%s = %q (present=%v), want %d — one served window of real cache; argv %v",
			promptCacheBytesFlag, got, ok, perToken*window, argv)
	}
	if got, ok := flagValue(argv, "--prompt-cache-size"); !ok || got != "10" {
		t.Errorf("--prompt-cache-size = %q (present=%v), want an explicit 10; argv %v", got, ok, argv)
	}
}

// serve.py takes its own flags by position, before anything it hands the
// server's argument parser, so the bound is always where it looks for it —
// and it is always there: a launch with no bound is one it refuses.
func TestThePromptCacheBoundFollowsTheSocketInEveryLaunch(t *testing.T) {
	for name, spec := range map[string]Spec{
		"known":          {ModelPath: "/models/org/a", Socket: "/s/m1", KVBytesPerToken: 10, ServedContext: 20},
		"no real figure": {ModelPath: "/models/org/a", Socket: "/s/m1", ServedContext: 20},
		"no window":      {ModelPath: "/models/org/a", Socket: "/s/m1", KVBytesPerToken: 10},
		"neither":        {ModelPath: "/models/org/a", Socket: "/s/m1"},
	} {
		argv := launchArgs(spec)
		i := indexOf(argv, "--dessau-socket")
		if i < 0 || i+3 >= len(argv) || argv[i+2] != promptCacheBytesFlag {
			t.Errorf("%s: %s does not follow the socket: %v", name, promptCacheBytesFlag, argv)
		}
		if n := countArg(argv, promptCacheBytesFlag); n != 1 {
			t.Errorf("%s: %s appears %d times, want once: %v", name, promptCacheBytesFlag, n, argv)
		}
	}
}

// A model whose real figure is not known — its configuration says nothing
// about its cache, or the window it is served at is not known — is charged
// the flat figure, which reserves nothing for a cache. Its prompt cache is
// bounded within that charge too: at nothing, so the server keeps no prompt
// cache between requests rather than one with no limit. A guess would be
// worse, as it is for the charge (capability.LoadCostOf).
func TestAModelWithNoRealCacheFigureKeepsNoPromptCache(t *testing.T) {
	for name, m := range map[string]ResolvedModel{
		"no real figure":   {ServedContext: 32768, KVChargePerToken: 100},
		"no window":        {KVChargePerToken: 100, KVBytesPerToken: 20},
		"nothing known":    {},
		"negative figures": {ServedContext: -1, KVBytesPerToken: -5},
	} {
		t.Run(name, func(t *testing.T) {
			l := newFakeLauncher()
			src := &fakeSource{models: map[string]int64{"org/a": 1 << 20}, facts: map[string]ResolvedModel{"org/a": m}}
			p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 40})
			_, release, err := p.Acquire(context.Background(), "org/a")
			if err != nil {
				t.Fatalf("Acquire: %v", err)
			}
			release()
			argv := launchArgs(l.specFor("org/a"))
			if got, ok := flagValue(argv, promptCacheBytesFlag); !ok || got != "0" {
				t.Errorf("%s = %q (present=%v), want 0; argv %v", promptCacheBytesFlag, got, ok, argv)
			}
		})
	}
}

func indexOf(argv []string, s string) int {
	for i, a := range argv {
		if a == s {
			return i
		}
	}
	return -1
}

package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/contextprobe"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// A pin saved in the instant before an idle job's unload reaches the pool is
// honoured: the model stays loaded and pinned, and the job is refused with its
// own ErrPinned. The pin check and the stop are one hold of the pool's lock,
// so no save can land between them (iss-2610042033419572).
func TestAnIdleJobsUnloadHonoursAPinSavedJustBeforeTheStop(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unload func(*App, string) error
		want   error
	}{
		{"self-test", func(a *App, id string) error { return selfTestServer{a}.Unload(id) }, selftest.ErrPinned},
		{"context probe", func(a *App, id string) error { return probeSources{a}.Unload(id) }, contextprobe.ErrPinned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newSelfTestPinApp(t, "org/m")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, release, err := a.Pool.Acquire(ctx, "org/m")
			if err != nil {
				t.Fatal(err)
			}
			release()

			previous := beforeIdleUnload
			beforeIdleUnload = func(string) { pin(t, a, "ORG/M") }
			t.Cleanup(func() { beforeIdleUnload = previous })

			if err := tc.unload(a, "org/m"); !errors.Is(err, tc.want) {
				t.Errorf("the unload of a model pinned just before the stop = %v, want %v", err, tc.want)
			}
			if !a.isPinned("org/m") {
				t.Error("the model is no longer pinned")
			}
			if !residentIn(a, "org/m") {
				t.Error("the idle job unloaded a model pinned just before the stop")
			}
		})
	}
}

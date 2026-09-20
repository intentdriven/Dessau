package archtest_test

import (
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/stats"
)

// The statistics package bounds a record's windows with its own constant,
// because it imports nothing of ours; this holds it to the registry's bound
// so the two cannot drift apart.
func TestTheStatisticsWindowBoundIsTheRegistrys(t *testing.T) {
	if stats.MaxContext != config.MaxContextLength {
		t.Errorf("stats.MaxContext = %d, config.MaxContextLength = %d; one figure", stats.MaxContext, config.MaxContextLength)
	}
}

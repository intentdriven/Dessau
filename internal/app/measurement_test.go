package app

import (
	"reflect"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// The launch flags reach the load event by name, only the ones set: the
// statistics package keeps a wire shape and imports nothing of ours, so the
// mapping is here, and this holds it to the five parameters.
func TestALoadEventCarriesTheLaunchSampling(t *testing.T) {
	temp, topP, topK, minP, maxT := 0.7, 0.9, 40, 0.05, 512
	got := samplingValues(config.Sampling{Temperature: &temp, TopP: &topP, TopK: &topK, MinP: &minP, MaxTokens: &maxT})
	want := map[string]float64{"temperature": 0.7, "top_p": 0.9, "top_k": 40, "min_p": 0.05, "max_tokens": 512}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("samplingValues = %v, want %v", got, want)
	}
	if got := samplingValues(config.Sampling{}); got != nil {
		t.Errorf("no flags set gave %v, want nil", got)
	}
	if got := samplingValues(config.Sampling{TopK: &topK}); !reflect.DeepEqual(got, map[string]float64{"top_k": 40}) {
		t.Errorf("one flag set gave %v", got)
	}
}

package stats

import (
	"context"
	"reflect"
	"testing"
	"time"
)

type captureStore struct {
	requests []Record
	events   []Event
}

func (s *captureStore) AppendRequest(r Record) error  { s.requests = append(s.requests, r); return nil }
func (s *captureStore) AppendEvent(e Event) error     { s.events = append(s.events, e); return nil }
func (s *captureStore) AppendSettings(Settings) error { return nil }

// A record carries the windows it was judged against, the in-flight count at
// admission, the estimate, the override names and the footprint; and each is
// bounded at the recorder: an implausible window or count is dropped, an
// override outside the closed set is dropped, never repaired.
func TestARecordCarriesTheMeasurementFieldsAndTheyAreBounded(t *testing.T) {
	store := &captureStore{}
	r := New(Options{Store: store})
	r.SetEnabled(true)
	r.Add(Record{Model: "org/m", Class: ClassOK, At: 100,
		DeclaredContext: 131072, ServedContext: 65536, EstimatedPromptTokens: 4000, InFlight: 2,
		Overrides: []string{"temperature", "vibes", "max_tokens"}, FootprintBytes: 48 << 30})
	r.Add(Record{Model: "org/m", Class: ClassOK, At: 101,
		DeclaredContext: 1 << 40, ServedContext: -1, EstimatedPromptTokens: -5, InFlight: 1 << 30,
		Overrides: []string{"nope"}, FootprintBytes: -1})
	if len(store.requests) != 2 {
		t.Fatalf("%d records written", len(store.requests))
	}
	got := store.requests[0]
	if got.DeclaredContext != 131072 || got.ServedContext != 65536 || got.EstimatedPromptTokens != 4000 ||
		got.InFlight != 2 || got.FootprintBytes != 48<<30 || !reflect.DeepEqual(got.Overrides, []string{"temperature", "max_tokens"}) {
		t.Errorf("record = %+v", got)
	}
	bad := store.requests[1]
	if bad.DeclaredContext != 0 || bad.ServedContext != 0 || bad.EstimatedPromptTokens != 0 ||
		bad.InFlight != 0 || bad.FootprintBytes != 0 || bad.Overrides != nil {
		t.Errorf("implausible figures survived: %+v", bad)
	}
}

// A footprint sample is an event of its own kind, recorded only while the
// switch is on, and the newest reading is on the model's counters.
func TestAFootprintSampleIsAnEvent(t *testing.T) {
	store := &captureStore{}
	r := New(Options{Store: store})
	r.FootprintSampled("org/m", 1<<30)
	if len(store.events) != 0 {
		t.Fatal("a sample was recorded with the switch off")
	}
	r.SetEnabled(true)
	r.FootprintSampled("org/m", 2<<30)
	r.FootprintSampled("org/m", 0)
	if len(store.events) != 1 || store.events[0].Kind != EventFootprint || store.events[0].Bytes != 2<<30 {
		t.Errorf("events = %+v, want one footprint of 2 GiB", store.events)
	}
}

// The load event carries the sampling the server was launched with, only
// the parameters in the closed set, and nothing once it has been carried out.
func TestALoadEventCarriesTheLaunchSampling(t *testing.T) {
	store := &captureStore{}
	r := New(Options{Store: store})
	r.SetEnabled(true)
	r.LoadStarted("org/m")
	r.LoadFinished("org/m", time.Second, nil, map[string]float64{"temperature": 0.7, "top_k": 40, "vibes": 1})
	if len(store.events) != 1 {
		t.Fatalf("%d events", len(store.events))
	}
	if got := store.events[0].Sampling; !reflect.DeepEqual(got, map[string]float64{"temperature": 0.7, "top_k": 40}) {
		t.Errorf("sampling = %v", got)
	}
	r.LoadFinished("org/m", time.Second, nil, nil)
	if store.events[1].Sampling != nil {
		t.Errorf("a load with no sampling carried some: %v", store.events[1].Sampling)
	}
}

// memSource feeds the aggregator lines the way the store does: newest first.
type memSource struct{ lines []Line }

func (m memSource) Read(_ context.Context, _ ReadOptions, fn func(Line) bool) (ReadStats, error) {
	for i := len(m.lines) - 1; i >= 0; i-- {
		if !fn(m.lines[i]) {
			break
		}
	}
	return ReadStats{}, nil
}

func sizedRequest(at int64, model string, est int, served, declared int64, overrides ...string) Line {
	return Line{Kind: KindRequest, At: at, Request: Record{Model: model, At: at, Class: ClassOK,
		EstimatedPromptTokens: est, RequestedTokens: est, ServedContext: served, DeclaredContext: declared, Overrides: overrides}}
}

// Prompts are bucketed by their share of the served window, the refused ones
// counted apart, and the windows carried beside the counts.
func TestPromptSizesAreBucketedAgainstBothWindows(t *testing.T) {
	src := memSource{lines: []Line{
		sizedRequest(1000, "org/m", 1000, 16384, 131072),  // ≤ 1/4
		sizedRequest(1001, "org/m", 6000, 16384, 131072),  // ≤ 1/2
		sizedRequest(1002, "org/m", 12000, 16384, 131072), // ≤ 3/4
		sizedRequest(1003, "org/m", 16000, 16384, 131072), // ≤ 1
		sizedRequest(1004, "org/m", 20000, 16384, 131072), // over: refused
	}}
	h, err := Aggregate(context.Background(), src, time.Unix(0, 0), time.Unix(2000, 0), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.PromptSizes) != 1 {
		t.Fatalf("prompt sizes = %+v", h.PromptSizes)
	}
	row := h.PromptSizes[0]
	if row.Requests != 5 || row.Buckets != [5]int{1, 1, 1, 1, 1} || row.Refused != 1 ||
		row.ServedContext != 16384 || row.DeclaredContext != 131072 || row.LargestEstimate != 20000 {
		t.Errorf("row = %+v", row)
	}
}

// The override rate is per parameter, over every request the model served,
// with every parameter present so a zero is a zero and not an absence.
func TestOverrideRatesPerParameter(t *testing.T) {
	src := memSource{lines: []Line{
		sizedRequest(1000, "org/m", 10, 100, 100, "temperature"),
		sizedRequest(1001, "org/m", 10, 100, 100, "temperature", "max_tokens"),
		sizedRequest(1002, "org/m", 10, 100, 100),
	}}
	h, err := Aggregate(context.Background(), src, time.Unix(0, 0), time.Unix(2000, 0), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Overrides) != 1 {
		t.Fatalf("overrides = %+v", h.Overrides)
	}
	row := h.Overrides[0]
	want := map[string]int{"temperature": 2, "top_p": 0, "top_k": 0, "min_p": 0, "max_tokens": 1}
	if row.Requests != 3 || !reflect.DeepEqual(row.ByParameter, want) {
		t.Errorf("row = %+v, want %v over 3", row, want)
	}
}

// The footprint series is the samples averaged into at most FootprintPoints
// slices of the range, stamped at each slice's start.
func TestFootprintSeriesIsDownsampledToTheRange(t *testing.T) {
	var lines []Line
	for at := int64(0); at < 20000; at += 10 {
		// Alternating readings, so the mean is exercised: each slice holds
		// ten samples, five of each.
		b := int64(1 << 30)
		if (at/10)%2 == 1 {
			b = 3 << 30
		}
		lines = append(lines, Line{Kind: KindFootprint, At: at, Event: Event{Model: "org/m", At: at, Kind: EventFootprint, Bytes: b}})
	}
	h, err := Aggregate(context.Background(), memSource{lines}, time.Unix(0, 0), time.Unix(20000, 0), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Footprints) != 1 {
		t.Fatalf("footprints = %+v", h.Footprints)
	}
	pts := h.Footprints[0].Points
	if len(pts) == 0 || len(pts) > FootprintPoints {
		t.Fatalf("%d points from 2000 samples, want at most %d", len(pts), FootprintPoints)
	}
	for _, p := range pts {
		if p.Bytes != 2<<30 {
			t.Errorf("a point averaged to %d, want the mean of 1 and 3 GiB", p.Bytes)
		}
	}
	if pts[0].At != 0 || pts[len(pts)-1].At >= 20000 {
		t.Errorf("points run from %d to %d", pts[0].At, pts[len(pts)-1].At)
	}
}

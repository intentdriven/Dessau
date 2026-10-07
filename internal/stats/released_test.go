package stats

import "testing"

// A model a program released is recorded under its own reason with the kind
// of caller that asked and nothing else about the caller: never a key, and
// never which client — the store says who no more than any record does
// (itd-2610031024247803; adversarial review of step 2).
func TestAReleaseRecordsTheKindOfCaller(t *testing.T) {
	store := &captureStore{}
	r := New(Options{Store: store})
	r.SetEnabled(true)

	r.Released("org/a", CallerThisMac)
	r.Released("org/a", CallerAPIKey)
	r.Released("org/a", CallerPairedClient)
	r.Released("org/a", "sk-a-key-in-the-wrong-place")

	want := []Event{
		{Model: "org/a", Kind: EventRemoved, Reason: ReasonReleased, By: CallerThisMac},
		{Model: "org/a", Kind: EventRemoved, Reason: ReasonReleased, By: CallerAPIKey},
		{Model: "org/a", Kind: EventRemoved, Reason: ReasonReleased, By: CallerPairedClient},
		{Model: "org/a", Kind: EventRemoved, Reason: ReasonReleased},
	}
	if len(store.events) != len(want) {
		t.Fatalf("%d events written, want %d", len(store.events), len(want))
	}
	for i, ev := range store.events {
		ev.At = 0
		if ev.Model != want[i].Model || ev.Kind != want[i].Kind || ev.Reason != want[i].Reason ||
			ev.By != want[i].By {
			t.Errorf("event %d = %+v, want %+v", i, ev, want[i])
		}
	}
	if got := r.Summary()[0].Evictions; got != 0 {
		t.Errorf("a release counted as %d evictions: only making room for another model is one", got)
	}
}

// The per-model counters the Statistics tab draws count each release by the
// kind of caller that asked, beside the evictions and apart from them, so the
// panel can show what Go records (iss-2610042100405045).
func TestTheCountersCountReleasesByKindOfCaller(t *testing.T) {
	r := New(Options{})
	r.SetEnabled(true)

	r.Released("org/a", CallerThisMac)
	r.Released("org/a", CallerThisMac)
	r.Released("org/a", CallerPairedClient)
	r.Released("org/a", "sk-a-key-in-the-wrong-place")
	r.Removed("org/a", ReasonEvicted)
	r.Removed("org/a", ReasonUnloaded)

	got := r.Summary()[0]
	want := map[string]int{CallerThisMac: 2, CallerPairedClient: 1, "": 1}
	if len(got.Released) != len(want) {
		t.Fatalf("Released = %v, want %v", got.Released, want)
	}
	for k, n := range want {
		if got.Released[k] != n {
			t.Errorf("Released[%q] = %d, want %d", k, got.Released[k], n)
		}
	}
	if got.Evictions != 1 {
		t.Errorf("Evictions = %d, want the one eviction alone", got.Evictions)
	}
	// What Summary hands out is a copy: the caller cannot reach the counts.
	got.Released[CallerThisMac] = 99
	if again := r.Summary()[0].Released[CallerThisMac]; again != 2 {
		t.Errorf("a caller's write reached the recorder's counts: %d", again)
	}
}

//go:build !race

package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// The dashboard's promise is that the panel stays responsive on a full store,
// so the bound is checked rather than asserted in prose: a store at the
// default 200 MB size cap is aggregated, and the pass has to finish inside
// aggregateBound.
//
// It is excluded from the race build, because the race detector multiplies
// wall-clock by something that varies with the machine and measures nothing
// about this code; `make test` runs with -race, so the bound is checked by the
// plain `go test ./...` that continuous integration also runs. It skips under
// -short, because it writes and reads two hundred megabytes.
//
// The bound is not a number of seconds. A store of 881,521 records in
// 209,723,172 bytes aggregates — read, grouped by day, model and hour, and
// reduced to percentiles — in 1.4 s on an idle Apple M4 Max, and in 12.3 s on
// a loaded one (iss-2609111104112814): a fixed wall-clock bound set from the
// first figure measures which of those two machines the suite is running on,
// which is not a property of this code.
//
// What is a property of this code is how much the pass costs OVER the work it
// cannot avoid. Every record has to be decoded whatever the aggregation does
// with it, so the decode is the yardstick: it is measured on this machine, at
// this moment, over this fixture's own records, and the pass is allowed a
// multiple of it. A machine having a bad afternoon slows both halves together
// and the multiple stands; a change that makes the pass an order of magnitude
// slower — a second decode, a read that is no longer bounded — raises the
// multiple on any machine, which is the change this is here to catch.
//
// The multiple is loose against the measurement it was set from: the pass
// costs a little over twice its decode on an idle Mac. Four is the room a
// grouping pass is allowed to take without anyone having to look at it.
const aggregateOverhead = 4

// aggregateFloor keeps a fixture that decoded implausibly fast from producing
// a bound no correct implementation could meet. It is well under the multiple
// on any machine this has been run on, so it decides nothing unless the
// yardstick itself is noise.
const aggregateFloor = 2 * time.Second

// aggregateCeiling is the second guard, and it is the one the multiple cannot
// give: a regression INSIDE the decode moves the yardstick and the bound
// together and would go unseen, however slow it made the panel
// (iss-2609181119340884). It is set where no machine this has run on comes
// near it — a minute against the 1.4 s an idle Apple M4 Max takes and the 4 s
// a loaded one does — so it fails a store that has stopped being aggregable at
// all, and nothing else.
const aggregateCeiling = 60 * time.Second

func TestAStoreAtTheSizeCapAggregatesWithinTheBound(t *testing.T) {
	if testing.Short() {
		t.Skip("fills the default size cap; run it deliberately")
	}
	dir := t.TempDir()
	end := time.Now().UTC().Truncate(time.Hour)
	records := writeCapSizedStore(t, dir, end)

	s := NewStore(dir, StoreOptions{Months: 1200, MaxBytes: defaultMaxBytes})
	t.Cleanup(func() { s.Close() })

	perRecord := decodeCost(t, dir)

	started := time.Now()
	h, err := Aggregate(t.Context(), s, end.AddDate(0, 0, -30), end, time.Local)
	took := time.Since(started)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("aggregated %d records of %d written in %v (%v per record decoded, %.2fx)",
		h.Records, records, took, perRecord, float64(took)/float64(perRecord*time.Duration(h.Records)))

	if h.Records < records {
		t.Errorf("the pass read %d of the %d records written", h.Records, records)
	}
	if h.Truncated {
		t.Errorf("a store at the default cap holds %d records, which is past the bound of %d",
			h.Records, MaxHistoryRecords)
	}
	bound := max(perRecord*time.Duration(h.Records)*aggregateOverhead, aggregateFloor)
	if took > bound {
		t.Errorf("aggregating a store at the default size cap took %v, which is more than %d times what decoding its %d records costs on this machine (%v)",
			took, aggregateOverhead, h.Records, bound)
	}
	if took > aggregateCeiling {
		t.Errorf("aggregating a store at the default size cap took %v, want under %v whatever the decode costs — the panel polls this",
			took, aggregateCeiling)
	}
}

// decodeCost measures what one record of this fixture costs to decode on this
// machine, now.
//
// It is the store's own line parser over the store's own bytes, so it is the
// irreducible half of the aggregating pass rather than a stand-in for it. It is
// taken over a few of the fixture's files rather than all forty: a yardstick
// that cost as much as the thing it measures would double what this test
// spends to say the same thing, and a sample this size is long enough that a
// scheduler hiccup does not decide it.
func decodeCost(t *testing.T, dir string) time.Duration {
	t.Helper()
	const sample = 4
	names, err := filepath.Glob(filepath.Join(dir, "stats-*.jsonl"))
	if err != nil || len(names) == 0 {
		t.Fatalf("no fixture files to measure the decode against: %v", err)
	}
	sort.Strings(names)
	if len(names) > sample {
		names = names[:sample]
	}
	files := make([][][]byte, 0, len(names))
	total := 0
	for _, name := range names {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		lines := bytes.Split(bytes.TrimRight(b, "\n"), []byte("\n"))
		files = append(files, lines)
		total += len(lines)
	}

	started := time.Now()
	parsed := 0
	for _, lines := range files {
		for _, line := range lines {
			if _, ok := parseLine(line); ok {
				parsed++
			}
		}
	}
	took := time.Since(started)
	if parsed != total {
		t.Fatalf("parsed %d of the %d lines sampled; the yardstick is not over the same records the pass reads", parsed, total)
	}
	return took / time.Duration(parsed)
}

// writeCapSizedStore fills a directory with the store's own files until they
// come to the default size cap, and returns how many records it wrote.
//
// The files are written directly rather than through the store's writer: what
// is being measured is the reading, and driving a million records through a
// buffered channel to measure a read would spend most of the test on the half
// that is not under test.
//
// The shape it makes is not quite a store's. Arrival times descend as each file
// grows while the file names ascend, so the whole fixture reads back oldest
// first and spans about ten days rather than the thirty the range asks for.
// That is the worst case for the timing — every record is decoded and every one
// of them is folded in — which is what this is here to measure; the ordering
// itself is asserted by the tests in dashboard_test.go, against fixtures
// written through the store.
func writeCapSizedStore(t *testing.T, dir string, end time.Time) int {
	t.Helper()
	models := []string{
		"mlx-community/Qwen3-8B-4bit",
		"mlx-community/Llama-3.2-3B-Instruct-4bit",
		"mlx-community/Mistral-7B-Instruct-v0.3-8bit",
	}
	written, num, total := 0, 0, int64(0)
	for total < defaultMaxBytes {
		num++
		var buf []byte
		for int64(len(buf)) < defaultRotateBytes {
			// Spread over the thirty days the default range covers, so the
			// day table has its rows and the range filter does real work.
			at := end.Add(-time.Duration(written%(30*24*60*60)) * time.Second).Unix()
			var line []byte
			var err error
			switch written % 500 {
			case 0:
				line, err = json.Marshal(eventLine{V: SchemaVersion, Kind: KindLoad, Event: Event{
					At: at, Model: models[written%len(models)], Kind: EventLoad, DurationMS: 4200,
				}})
			case 1:
				line, err = json.Marshal(eventLine{V: SchemaVersion, Kind: KindRemoved, Event: Event{
					At: at, Model: models[written%len(models)], Kind: EventRemoved, Reason: ReasonEvicted,
				}})
			default:
				line, err = json.Marshal(requestLine{V: SchemaVersion, Kind: KindRequest, Record: Record{
					Model: models[written%len(models)], At: at, Class: ClassOK, Streamed: true,
					PromptTokens: 1234 + written%97, CompletionTokens: 567 + written%53,
					FirstTokenMS: int64(120 + written%900), DurationMS: int64(2000 + written%3000),
					QueueWaitMS: int64(written % 40),
				}})
			}
			if err != nil {
				t.Fatal(err)
			}
			buf = append(append(buf, line...), '\n')
			written++
		}
		name := fmt.Sprintf("stats-%s-%03d.jsonl", end.Format("20060102"), num)
		if err := os.WriteFile(filepath.Join(dir, name), buf, 0o600); err != nil {
			t.Fatal(err)
		}
		total += int64(len(buf))
	}
	t.Logf("fixture store: %d bytes in %d files, %d records", total, num, written)
	return written
}

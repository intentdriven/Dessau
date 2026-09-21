package gateway

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// The panel rebuilds the whole per-model map from the snapshot it last
// loaded, so a field written into config.json by hand after that load is
// absent from the posted body. A save that REPLACED the map would discard it
// in silence (itd-2609091715089488): the other per-model settings fail
// towards a slower model, and the transcript exception fails towards a
// transcript that was promised not to exist. So the save merges, for every
// field on ModelSettings, and this test is written against the surviving
// value rather than against the merge — a later rewrite of the save path
// cannot satisfy it by accident.
//
// plantedPerModel is one non-zero value for every field on
// config.ModelSettings. It is keyed by the Go field name and the test walks
// the struct, so a field added later fails here until it is planted too.
var plantedPerModel = map[string]config.ModelSettings{
	"MergeSystemMessages": {MergeSystemMessages: true},
	"Pinned":              {Pinned: true},
	"Sampling":            {Sampling: config.Sampling{Temperature: floatPtr(0.2)}},
	"ServedContext":       {ServedContext: 4096},
	"NoTranscript":        {NoTranscript: true},
}

func floatPtr(f float64) *float64 { return &f }

func TestAPerModelFieldThePanelDidNotRenderSurvivesASave(t *testing.T) {
	const id = "mlx-community/Qwen3-8B-4bit"
	rt := reflect.TypeOf(config.ModelSettings{})
	for i := range rt.NumField() {
		field := rt.Field(i)
		planted, ok := plantedPerModel[field.Name]
		if !ok {
			t.Errorf("config.ModelSettings has a field %s this test does not plant; add it to plantedPerModel", field.Name)
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			// The panel's snapshot names the model with one setting the form
			// owns — a different one from the field under test — so the body
			// it posts names the model and says nothing about that field.
			snapshot := config.Default()
			anchor := config.ModelSettings{MergeSystemMessages: true}
			if field.Name == "MergeSystemMessages" {
				anchor = config.ModelSettings{Pinned: true}
			}
			snapshot.Models = map[string]config.ModelSettings{id: anchor}
			srv, a := newTestControlApp(t, snapshot)
			body := uneditedFormBody(t, snapshot)

			// Alice hand-edits config.json after the panel loaded: the field
			// under test is in force, and the panel does not know it.
			edited := snapshot.Clone()
			entry := edited.Models[id]
			reflect.ValueOf(&entry).Elem().Field(i).Set(reflect.ValueOf(planted).Field(i))
			edited.Models[id] = entry
			if err := a.SetConfig(edited); err != nil {
				t.Fatalf("planting %s: %v", field.Name, err)
			}

			resp := postJSON(t, srv, "/api/settings", body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d: %s", resp.StatusCode, refusalText(t, resp))
			}

			got := a.Config().Models[id]
			want := reflect.ValueOf(planted).Field(i).Interface()
			if have := reflect.ValueOf(got).Field(i).Interface(); !reflect.DeepEqual(have, want) {
				t.Errorf("models[%s].%s = %+v after a save that never named it, want %+v — the save dropped a field the panel did not render",
					id, field.Name, have, want)
			}
			// And the setting the body did carry is still what it said.
			if !reflect.DeepEqual(stripField(got, i), anchor) {
				t.Errorf("the rest of models[%s] = %+v after the save, want %+v", id, stripField(got, i), anchor)
			}
		})
	}
}

// stripField is a ModelSettings with one field zeroed, for comparing the
// rest of an entry against what the body posted.
func stripField(m config.ModelSettings, i int) config.ModelSettings {
	v := reflect.ValueOf(&m).Elem()
	v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
	return m
}

// A posted key replaces its field whole, and only a key the body never names
// survives. This is the direction that keeps the panel's own clearing honest:
// a box the form drew and left empty is posted as its zero value, and the
// merge must not bring the stored value back — a nested object included,
// which is why the merge overwrites encoded keys rather than decoding the
// posted object into the stored struct.
func TestAPostedPerModelKeyReplacesTheStoredFieldWhole(t *testing.T) {
	const id = "mlx-community/Qwen3-8B-4bit"
	stored := config.Default()
	stored.Models = map[string]config.ModelSettings{id: {
		Pinned:        true,
		ServedContext: 4096,
		Sampling:      config.Sampling{Temperature: floatPtr(0.2), TopP: floatPtr(0.9)},
	}}
	srv, a := newTestControlApp(t, stored)

	resp := postJSON(t, srv, "/api/settings",
		`{"host":"0.0.0.0","port":11535,"api_key":"","decode_concurrency":1,"idle_timeout_sec":0,`+
			`"models":{"`+id+`":{"pinned":false,"served_context":0,"sampling":{"top_p":0.5},"no_transcript":true}}}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	got := a.Config().Models[id]
	if got.Pinned {
		t.Error("a posted pinned:false left the stored pin in place")
	}
	if got.ServedContext != 0 {
		t.Errorf("a posted served_context:0 left the stored window at %d", got.ServedContext)
	}
	if got.Sampling.Temperature != nil {
		t.Errorf("a posted sampling object without a temperature brought the stored one back: %v", *got.Sampling.Temperature)
	}
	if got.Sampling.TopP == nil || *got.Sampling.TopP != 0.5 {
		t.Errorf("sampling.top_p = %v, want the posted 0.5", got.Sampling.TopP)
	}
	if !got.NoTranscript {
		t.Error("the posted exception was not stored")
	}
}

// The merged map holds exactly the keys the body names, under whatever
// spelling the body uses: a model whose every box is cleared is left out of
// the body and so removed, and a stored entry under another spelling of a
// posted id is the entry merged into rather than a second one kept beside it.
func TestTheMergedPerModelMapHoldsExactlyTheKeysTheBodyNames(t *testing.T) {
	stored := config.Default()
	stored.Models = map[string]config.ModelSettings{
		"mlx-community/Qwen3-8B-4bit": {NoTranscript: true},
		"mlx-community/Gone-4bit":     {Pinned: true, NoTranscript: true},
	}
	srv, a := newTestControlApp(t, stored)

	resp := postJSON(t, srv, "/api/settings",
		`{"host":"0.0.0.0","port":11535,"api_key":"","decode_concurrency":1,"idle_timeout_sec":0,`+
			`"models":{"MLX-Community/Qwen3-8B-4bit":{"pinned":true}}}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	got := a.Config().Models
	if len(got) != 1 {
		t.Fatalf("models = %+v, want exactly the one the body named", got)
	}
	for key, entry := range got {
		if !strings.EqualFold(key, "mlx-community/Qwen3-8B-4bit") {
			t.Errorf("the stored key is %q, want the one model the body named", key)
		}
		if !entry.Pinned || !entry.NoTranscript {
			t.Errorf("models[%s] = %+v, want the posted pin merged over the stored exception", key, entry)
		}
	}
}

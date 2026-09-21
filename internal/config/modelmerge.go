package config

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MergeModelSettings is the per-model half of a settings save: the stored
// map, merged field by field with what the body posted under "models".
//
// The result holds EXACTLY the keys the body names — a model the body leaves
// out is removed, which is how clearing every box for a model removes it —
// and for each of those keys, the field the posted object names replace the
// stored field whole, while a field it never names keeps its stored value.
// The stored entry is found under the posted key first and then under any
// spelling that folds alike, so a hand-typed spelling merges into the
// registry's rather than standing beside it.
//
// Merged at the encoded-object level rather than by decoding the posted
// object into the stored struct: encoding/json decodes into an existing
// struct in place, so a posted `"sampling": {}` decoded into a stored entry
// would merge the nested object too and an override the operator had just
// cleared would come back. Overwriting whole keys means the panel's own
// clearing stays honest — a box the form drew and left empty is posted as
// its zero value and wins — and only a key the body never names survives:
// a field written into config.json by hand after the panel loaded its
// snapshot, which a replaced map discarded in silence
// (itd-2609091715089488, the 2026-09-20 decision to merge every per-model
// field rather than one).
//
// A posted value that is null names the model and says nothing about any
// field, so the stored entry survives untouched; anything else that is not
// an object is refused, as the struct decode refused it before.
func MergeModelSettings(stored map[string]ModelSettings, posted map[string]json.RawMessage) (map[string]ModelSettings, error) {
	if posted == nil {
		return nil, nil
	}
	out := make(map[string]ModelSettings, len(posted))
	for key, raw := range posted {
		var fields map[string]json.RawMessage
		if !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			if err := json.Unmarshal(raw, &fields); err != nil {
				return nil, fmt.Errorf("settings for model %q are not an object", key)
			}
		}
		base, err := encodedModelSettings(storedEntry(stored, key))
		if err != nil {
			return nil, err
		}
		for name, value := range fields {
			base[name] = value
		}
		merged, err := json.Marshal(base)
		if err != nil {
			return nil, err
		}
		var entry ModelSettings
		if err := json.Unmarshal(merged, &entry); err != nil {
			return nil, fmt.Errorf("settings for model %q: %w", key, err)
		}
		out[key] = entry
	}
	return out, nil
}

// storedEntry is the stored settings for a posted key: under the key as
// posted, else under the first spelling in sorted order that folds alike,
// else nothing. Sorted, so that a map holding two spellings — which the
// settings path refuses and the file path drops — still merges the same one
// on every save.
func storedEntry(stored map[string]ModelSettings, key string) ModelSettings {
	if ms, ok := stored[key]; ok {
		return ms
	}
	folded := FoldRepoID(key)
	for _, id := range modelKeys(stored) {
		if FoldRepoID(id) == folded {
			return stored[id]
		}
	}
	return ModelSettings{}
}

// encodedModelSettings is one model's settings as the keys config.json
// would carry, each still encoded, so a posted key can replace one whole.
func encodedModelSettings(ms ModelSettings) (map[string]json.RawMessage, error) {
	b, err := json.Marshal(ms)
	if err != nil {
		return nil, err
	}
	out := map[string]json.RawMessage{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

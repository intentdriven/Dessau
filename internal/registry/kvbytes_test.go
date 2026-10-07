package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// hybridConfig is Qwen3-Coder-Next's shape: 12 full-attention layers of 48,
// 24,576 bytes a token of real cache, charged five times that.
const hybridConfig = `{"model_type":"qwen3_next","max_position_embeddings":262144,"num_hidden_layers":48,
  "full_attention_interval":4,"num_key_value_heads":2,"head_dim":256}`

// The real cache a token costs is recorded beside what it is charged, from
// the same scan, so the launcher can bound the model server's prompt cache
// by it (iss-2610071035130302). A model the index already lists gains it at
// the next rescan, as it gains every other figure the scan re-derives.
func TestRescanRecordsTheRealCacheFigure(t *testing.T) {
	r, dir := newTestRegistry(t)
	if err := r.Put(Model{RepoID: "org/hybrid", State: StateReady, KVChargePerToken: 24576 * 5}); err != nil {
		t.Fatal(err)
	}
	writeModelDirWithConfig(t, dir, "org", "hybrid", hybridConfig, 1024)
	if err := r.Rescan(dir); err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	m, err := r.Get("org/hybrid")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.KVBytesPerToken != 24576 {
		t.Errorf("KVBytesPerToken = %d, want 24576", m.KVBytesPerToken)
	}
	if m.KVChargePerToken != 24576*5 {
		t.Errorf("KVChargePerToken = %d, want %d", m.KVChargePerToken, 24576*5)
	}
}

// A configuration whose charge is past the plausible ceiling records no real
// figure either: a model charged the flat figure has no cache in its charge
// for the prompt cache to be bounded within.
func TestAnImplausibleConfigurationRecordsNoRealCacheFigure(t *testing.T) {
	// 64 layers of 64 heads of 1,024: 16 MiB a token of real cache, under
	// the ceiling on its own and five times past it once charged.
	const hostile = `{"model_type":"test","num_hidden_layers":64,"num_key_value_heads":64,"head_dim":1024}`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(hostile), 0o644); err != nil {
		t.Fatal(err)
	}
	facts := ReadModelFacts(dir)
	if facts.KVChargePerToken != 0 {
		t.Fatalf("KVChargePerToken = %d, want 0 for a charge past the ceiling", facts.KVChargePerToken)
	}
	if facts.KVBytesPerToken != 0 {
		t.Errorf("KVBytesPerToken = %d for a model charged no cache, want 0", facts.KVBytesPerToken)
	}
}

// The figure is persisted, and it is what the model server's prompt cache is
// allowed to hold, so it survives a reopen as written and is held, when read
// back, to the bound the charge beside it is held to — and to the charge
// itself, which a real figure is never above.
func TestTheRealCacheFigureRoundTripsAndIsBoundedOnRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(Model{RepoID: "org/kept", State: StateReady, KVChargePerToken: 24576 * 5, KVBytesPerToken: 24576}); err != nil {
		t.Fatal(err)
	}
	r, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := r.Get("org/kept"); m.KVBytesPerToken != 24576 {
		t.Errorf("after a reopen KVBytesPerToken = %d, want 24576", m.KVBytesPerToken)
	}

	planted := fmt.Sprintf(`[
	  {"repo_id":"org/huge","state":"ready","kv_charge_per_token":%d,"kv_bytes_per_token":%d},
	  {"repo_id":"org/negative","state":"ready","kv_charge_per_token":100,"kv_bytes_per_token":-1},
	  {"repo_id":"org/above","state":"ready","kv_charge_per_token":100,"kv_bytes_per_token":101},
	  {"repo_id":"org/uncharged","state":"ready","kv_bytes_per_token":20},
	  {"repo_id":"org/fine","state":"ready","kv_charge_per_token":100,"kv_bytes_per_token":20}]`,
		int64(MaxKVChargePerToken), int64(MaxKVChargePerToken)+1)
	if err := os.WriteFile(path, []byte(planted), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]int64{
		"org/huge": 0, "org/negative": 0, "org/above": 0, "org/uncharged": 0, "org/fine": 20,
	} {
		if m, _ := r.Get(id); m.KVBytesPerToken != want {
			t.Errorf("%s: KVBytesPerToken = %d, want %d", id, m.KVBytesPerToken, want)
		}
	}
}

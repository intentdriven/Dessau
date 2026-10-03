package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/stats"
)

// The merge setting is read the way every per-model setting is: folded, so a
// setting stored under another spelling of the model's id than the one a
// request resolves to still applies (iss-2609201015464436).
func TestTheMergeSettingAppliesWhicheverWayItsIdIsSpelled(t *testing.T) {
	cfg := config.Default()
	cfg.Models = map[string]config.ModelSettings{
		"MLX-COMMUNITY/QWEN3-8B-4BIT": {MergeSystemMessages: true},
	}
	srv, _, fake := newTestGateway(t, cfg)
	body := `{"model":"mlx-community/Qwen3-8B-4bit","messages":[` +
		`{"role":"system","content":"one"},{"role":"user","content":"hi"},{"role":"system","content":"two"}]}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+chatCompletionsPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	msgs, _ := fake.LastBody()["messages"].([]any)
	systems := 0
	for _, m := range msgs {
		if mm, _ := m.(map[string]any); mm["role"] == "system" {
			systems++
		}
	}
	if systems != 1 {
		t.Errorf("the model server got %d system messages, want them merged into one: %v", systems, msgs)
	}
}

// The bridge's path reads the same setting the same way: a conversation the
// bridge asks for is merged under any spelling the setting is stored under.
func TestAskMergesWhicheverWayTheSettingsIdIsSpelled(t *testing.T) {
	const modelPath = "/models/" + testModelID
	fake := mlxtest.Start(mlxtest.Options{ModelArg: modelPath, Reply: "ok"})
	t.Cleanup(fake.Close)
	cfg := config.Default()
	cfg.Models = map[string]config.ModelSettings{
		strings.ToUpper(testModelID): {MergeSystemMessages: true},
	}
	g := New(Options{
		Config: cfg,
		Pool:   &stubPool{srv: fake},
		Models: &stubModels{models: []registry.Model{{ChatTemplate: true,
			RepoID: testModelID, Path: modelPath, State: registry.StateReady, ContextLength: 131072,
		}}},
		ServedWindow: settingOrDeclared(cfg),
	})
	body, err := json.Marshal(map[string]any{
		"model": testModelID,
		"messages": []map[string]string{
			{"role": "system", "content": "one"}, {"role": "user", "content": "hi"}, {"role": "system", "content": "two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Ask(context.Background(), AskRequest{
		Model: testModelID, Body: body, Source: stats.SourceBridge, OnEvent: func([]byte) {},
	}); err != nil {
		t.Fatal(err)
	}
	msgs, _ := fake.LastBody()["messages"].([]any)
	systems := 0
	for _, m := range msgs {
		if mm, _ := m.(map[string]any); mm["role"] == "system" {
			systems++
		}
	}
	if systems != 1 {
		t.Errorf("the model server got %d system messages from Ask, want them merged into one: %v", systems, msgs)
	}
}

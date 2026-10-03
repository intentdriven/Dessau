package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/stats"
)

// The panel's Load starts a chat model server, so it is refused for a model
// the chat rule says cannot hold a conversation, and nothing is loaded; a chat
// model loads as before. The panel's view carries the same verdict, so the
// card can leave Load out (iss-2610031010371709).
func TestLoadIsRefusedForAModelThatCannotChat(t *testing.T) {
	a, _, mux := newWiredControl(t, config.Default(), "org/ears", "org/talker")
	ears, _ := a.Registry.Get("org/ears")
	ears.ChatTemplate, ears.PipelineTag = false, "automatic-speech-recognition"
	talker, _ := a.Registry.Get("org/talker")
	talker.PipelineTag, talker.Tags = "text-generation", []string{"conversational"}
	for _, m := range []registry.Model{ears, talker} {
		if err := a.Registry.Put(m); err != nil {
			t.Fatal(err)
		}
	}
	srv := serve(t, mux)

	resp, err := srv.Client().Post(srv.URL+"/api/models/load", "application/json", strings.NewReader(`{"model":"org/ears"}`))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict || !strings.Contains(string(b), "not a chat model") {
		t.Errorf("Load of a model that cannot chat: status %d, %s; want 409 saying so", resp.StatusCode, b)
	}
	if got := a.Pool.Resident(); len(got) != 0 {
		t.Errorf("a refused Load put %v in memory", got)
	}

	resp, err = srv.Client().Post(srv.URL+"/api/models/load", "application/json", strings.NewReader(`{"model":"org/talker"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Load of a chat model: status %d, want 202", resp.StatusCode)
	}

	views := (&Control{App: a}).modelViews()
	chat := map[string]bool{}
	for _, v := range views {
		chat[v.RepoID] = v.Chat
	}
	if chat["org/ears"] || !chat["org/talker"] {
		t.Errorf("the panel's chat verdicts = %v, want ears false and talker true", chat)
	}
}

// The bridge holds a conversation, so it refuses a model that cannot chat as
// /v1/chat/completions does, with the generic text travelling.
func TestTheBridgeRefusesAModelThatCannotChat(t *testing.T) {
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m", Reply: "x"})
	t.Cleanup(fake.Close)
	models := &stubModels{models: []registry.Model{{
		RepoID: "org/ears", Path: "/m", State: registry.StateReady, PipelineTag: "automatic-speech-recognition",
	}}}
	g := New(Options{Config: config.Default(), Pool: &stubPool{srv: fake}, Models: models})
	err := g.Ask(context.Background(), AskRequest{
		Model:  "org/ears",
		Body:   []byte(`{"model":"org/ears","messages":[{"role":"user","content":"hi"}]}`),
		Source: stats.SourceBridge,
	})
	var askErr *AskError
	if !errors.As(err, &askErr) || !strings.Contains(askErr.Error(), "not a chat model") {
		t.Fatalf("Ask to a model that cannot chat = %v, want an AskError saying so", err)
	}
	if fake.LastBody() != nil {
		t.Error("the refused conversation reached the model server")
	}
}

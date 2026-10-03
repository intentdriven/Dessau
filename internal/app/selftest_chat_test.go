package app

import (
	"testing"

	"github.com/intentdriven/Dessau/internal/registry"
)

// The self-test sends a chat request, so it never loads a model the chat rule
// says cannot hold a conversation (iss-2610031010371709).
func TestTheSelfTestLeavesOutAModelThatCannotChat(t *testing.T) {
	a := newTestApp(t)
	for _, m := range []registry.Model{
		{RepoID: "org/talker", State: registry.StateReady, PipelineTag: "text-generation", Tags: []string{"conversational"}},
		{RepoID: "org/ears", State: registry.StateReady, PipelineTag: "automatic-speech-recognition"},
	} {
		if err := a.Registry.Put(m); err != nil {
			t.Fatal(err)
		}
	}
	if ready := (selfTestServer{a}).Ready(); len(ready) != 1 || ready[0] != "org/talker" {
		t.Errorf("the self-test's ready list = %v, want only the chat model", ready)
	}
}

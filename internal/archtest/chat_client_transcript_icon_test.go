package archtest_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A model's transcript state is shown as an icon wherever a person picks a
// model (itd-2609091715089488, the 2026-09-20 decision): the chat client's
// picker draws it beside each server model, read from the models-list
// `recording` field, with an accessible label in words rather than a
// symbol's name — so the person knows before choosing rather than after
// asking. A boolean on a models list tells software; this is what tells a
// person. Each promise is about the client's SOURCE, so each is read here,
// in the shape chat_client_load_state_test.go reads the category fields.

// clientRecordingField matches the client decoding the recording fact off a
// models-list entry. Codable keys off the property name, so the property IS
// the wire field, and it is optional because an older server publishes none
// — an absent fact is not a promise, and draws no icon.
var clientRecordingField = regexp.MustCompile(
	`(?m)^\s*let\s+recording\s*:\s*Bool\?`)

// gatewayRecordingField matches the gateway writing the same field onto a
// models-list entry.
var gatewayRecordingField = regexp.MustCompile(
	`(?m)^\s*entry\["recording"\]\s*=`)

// clientTranscriptLabels are the two accessible labels the picker's icon
// carries, one per state, in words. A symbol's name is not a label: a
// screen reader that says "pencil slash" has told the person nothing.
var clientTranscriptLabels = []string{"keeps no transcript", "recorded"}

// TestChatClientReadsTheRecordingFieldTheGatewayPublishes holds the client's
// decode of the transcript state to the field the server writes it under:
// a renamed field decodes as nil, no icon is drawn, and nothing errors.
func TestChatClientReadsTheRecordingFieldTheGatewayPublishes(t *testing.T) {
	root := repoRootDir(t)
	source := clientSharedSource(t, root)
	gatewaySource := readRepoFile(t, root, filepath.Join("internal", "gateway", "gateway.go"))

	if !clientRecordingField.MatchString(source) {
		t.Error("client/DessauChat/ declares no `let recording: Bool?` on its models decoder; " +
			"the picker has no fact to draw the transcript icon from")
	}
	if !gatewayRecordingField.MatchString(gatewaySource) {
		t.Error("internal/gateway/gateway.go no longer writes entry[\"recording\"] onto a models-list entry; " +
			"the chat client decodes the transcript state under that name")
	}
}

// TestChatClientPickerShowsTheTranscriptStateWithWords holds the icon to the
// picker's model rows and its label to words: the state is visible before
// the model is chosen, and audible to a person who cannot see the glyph.
func TestChatClientPickerShowsTheTranscriptStateWithWords(t *testing.T) {
	picker, ok := clientSources(t, repoRootDir(t))["Picker.swift"]
	if !ok {
		t.Fatal("client/DessauChat/Picker.swift is missing; the model rows have no home")
	}
	rows, ok := swiftFunctionBody(picker, "private var modelRows: some View")
	if !ok {
		t.Fatal("client/DessauChat/Picker.swift no longer declares modelRows; the server models are drawn somewhere this test cannot see")
	}
	if !strings.Contains(rows, "accessibilityLabel") {
		t.Error("the picker's model rows carry no accessibilityLabel; the transcript icon is a glyph a screen reader cannot read")
	}
	for _, words := range clientTranscriptLabels {
		if !strings.Contains(picker, `"`+words+`"`) {
			t.Errorf("client/DessauChat/Picker.swift never says %q; the transcript icon's label is not in words", words)
		}
	}
}

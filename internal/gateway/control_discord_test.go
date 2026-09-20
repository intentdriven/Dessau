package gateway

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
)

// The bot token is a secret on every surface the API key is, and these are the
// three obligations itd-2609081259493890 established for the key, one test
// each, beside the key's own.

// One: it is never sent to the panel.
func TestTheDiscordTokenIsRedactedInState(t *testing.T) {
	cfg := config.Default()
	cfg.DiscordBridge = true
	cfg.DiscordToken = "not-a-real-bot-token"
	srv := newTestControl(t, cfg)

	resp, err := srv.Client().Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	raw := string(body[:n])

	if strings.Contains(raw, "not-a-real-bot-token") {
		t.Error("the Discord bot token was sent to the control panel in plaintext")
	}
	if !strings.Contains(raw, redacted) {
		t.Error("the panel was sent no placeholder for the token, so it cannot show that one is set")
	}
}

// Two: the placeholder the panel echoes back does not overwrite the real one.
// Every save posts every field the form owns, so without this the first save
// of any unrelated setting would replace the token with asterisks.
func TestSavingTheRedactedPlaceholderKeepsTheRealDiscordToken(t *testing.T) {
	cfg := config.Default()
	cfg.DiscordBridge = true
	cfg.DiscordToken = "not-a-real-bot-token"
	srv, a := newTestControlApp(t, cfg)

	body := `{"host":"0.0.0.0","port":11535,"api_key":"","discord_bridge":true,` +
		`"discord_token":"` + redacted + `","decode_concurrency":1,"idle_timeout_sec":0}`
	resp, err := srv.Client().Post(srv.URL+"/api/settings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// Read from the running configuration rather than from the panel. The
	// panel is served the placeholder either way, so a snapshot cannot tell a
	// stored token from a stored row of asterisks — which is precisely the
	// wipe being guarded against.
	if got := a.Config().DiscordToken; got != "not-a-real-bot-token" {
		t.Errorf("the stored token is now %q; saving the placeholder overwrote the real one", got)
	}

	// And the panel still reports that a token is set.
	stResp, err := srv.Client().Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer stResp.Body.Close()
	var st State
	if err := json.NewDecoder(stResp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.Config.DiscordToken != redacted {
		t.Errorf("the panel reports the token as %q", st.Config.DiscordToken)
	}
}

// Three: it is never compared against a posted value. The control plane is
// loopback-only and asks nobody for a password, and loopback includes every
// other account on this Mac — so a refusal that named the token among the
// changed settings would be an oracle on a guess.
func TestARefusalDoesNotSayWhetherAGuessedDiscordTokenWasRight(t *testing.T) {
	if !secretSettingKeys["discord_token"] {
		t.Fatal("discord_token is not excluded from the value comparison in changedSettings")
	}
	before := config.Default()
	before.DiscordToken = "the-real-token"
	after := before.Clone()
	after.DiscordToken = "the-real-token"
	after.Port = 12345

	// A caller guessing the token: the guess is wrong, and the answer must
	// not say so.
	wrong := changedSettings(before, after, []byte(`{"discord_token":"a-wrong-guess","port":12345}`))
	// And a caller guessing it right.
	right := changedSettings(before, after, []byte(`{"discord_token":"the-real-token","port":12345}`))
	if !equalStrings(wrong, right) {
		t.Errorf("a wrong guess is answered with %v and a right one with %v — the refusal is an oracle",
			wrong, right)
	}
	// What it does say is what the caller itself sent.
	if !slices.Contains(wrong, "discord_token") || !slices.Contains(wrong, "port") {
		t.Errorf("the refusal named %v, want both fields the caller posted", wrong)
	}
	// A caller that posted the placeholder asked to change nothing.
	quiet := changedSettings(before, after, []byte(`{"discord_token":"`+redacted+`","port":12345}`))
	if slices.Contains(quiet, "discord_token") {
		t.Errorf("the refusal named the token for a caller that posted the placeholder: %v", quiet)
	}
}

// A save is never refused over the bridge. A token that is absent, malformed
// or rejected by Discord is the bridge's state to report, and an operator
// changing an unrelated setting is never turned away over a credential they
// did not touch (itd-2609180959397172).
func TestASaveIsNeverRefusedOverTheDiscordToken(t *testing.T) {
	srv := newTestControl(t, config.Default())
	for _, token := range []string{
		"",
		"obviously-not-a-token",
		strings.Repeat("x", 4096),
		"a token with spaces and é accents",
	} {
		encoded, err := json.Marshal(token)
		if err != nil {
			t.Fatal(err)
		}
		body := `{"host":"0.0.0.0","port":11535,"api_key":"","discord_bridge":true,` +
			`"discord_token":` + string(encoded) + `,"decode_concurrency":1,"idle_timeout_sec":0}`
		resp, err := srv.Client().Post(srv.URL+"/api/settings", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("saving a token of %d characters was refused with %d; a save is never refused "+
				"over the bridge", len(token), resp.StatusCode)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

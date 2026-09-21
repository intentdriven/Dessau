package ui

import (
	"fmt"
	"testing"

	"github.com/intentdriven/Dessau/internal/runtime"
)

// Debug logging for one model (itd-2609062346072707) is armed from the
// model's card and shown while it is armed or on. The card reads the armed
// set from the snapshot and the running fact from the resident entry;
// internal/archtest holds that the two pills are drawn from those two fields.
// What is held here is the join and the posture line's words.

// The armed set is joined on the folded repo id, the way every other join on
// a repo id in this panel is, so a model armed under another spelling than
// the registry's still carries its pill.
func TestTheArmedSetIsMatchedFolded(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"armed", `debugArmedFor({"repo_id":"org/m"}, ["org/m"])`, "true"},
		{"armed under another spelling", `debugArmedFor({"repo_id":"org/Model"}, ["ORG/model"])`, "true"},
		{"another model is armed", `debugArmedFor({"repo_id":"org/m"}, ["org/other"])`, "false"},
		{"nothing is armed", `debugArmedFor({"repo_id":"org/m"}, [])`, "false"},
		{"the snapshot carries no set", `debugArmedFor({"repo_id":"org/m"}, undefined)`, "false"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := evalPanelExpr(t, c.expr, "foldRepoID", "debugArmedFor"); got != c.want {
				t.Errorf("%s = %q, want %q", c.expr, got, c.want)
			}
		})
	}
}

// The posture line is present only while a model is armed or running at
// debug, names those models and keeps armed apart from running, says what the
// level writes and who it does not tell, and names the bound and the kept
// file. The figure is the code's, not the page's.
func TestThePostureNamesTheModelsAtDebug(t *testing.T) {
	if _, ok := posture(t, baseSnapshot)["debug_log"]; ok {
		t.Error("the page has a debug_log line while no model is armed or at debug")
	}

	armed := posture(t, edited(t, `{"debug_armed":["org/armed"]}`))
	wants(t, armed, "debug_log",
		"armed for org/armed", "next start",
		"every request sent to it and every answer it produced",
		"prompts and the completions, whoever sent them",
		"Dessau's own probes",
		"another client's request",
		fmt.Sprintf("%d MB", runtime.DebugLogMaxBytes>>20),
		"previous run's file is kept",
		"clients are not told")
	refuses(t, armed, "debug_log", "on for")

	running := posture(t, edited(t, `{"resident":[{"repo_id":"org/on","debug_log":true},{"repo_id":"org/quiet","debug_log":false}]}`))
	wants(t, running, "debug_log", "on for org/on")
	refuses(t, running, "debug_log", "armed for", "org/quiet")

	both := posture(t, edited(t, `{"debug_armed":["org/armed"],"resident":[{"repo_id":"org/on","debug_log":true}]}`))
	wants(t, both, "debug_log", "armed for org/armed", "on for org/on")
}

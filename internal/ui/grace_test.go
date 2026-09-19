package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/Gropius/internal/config"
)

// The form owns three fields, and a save that left any of them out would
// switch the feature off — the settings endpoint replaces what the form sends.
func TestSettingsFormPostsTheEvictionGraceFields(t *testing.T) {
	src := readPanelSource(t)
	for _, field := range []string{"eviction_grace:", "eviction_grace_sec:", "eviction_max_wait_sec:"} {
		if !strings.Contains(src, field) {
			t.Errorf("the settings form does not post %s", field)
		}
	}
	for _, id := range []string{"setGrace", "setGraceSec", "setGraceWait"} {
		if !strings.Contains(src, "'"+id+"'") {
			t.Errorf("the panel never reads %s, so the field cannot reach a save", id)
		}
	}
}

// The browser refuses a form whose number is outside a field's own min/max
// before the submit listener runs, so a bound here that is tighter than the
// server's would make a figure saved by any other client unsavable from the
// panel — the wedge step="0.1" was on the budget field. Bound the two fields
// to the ceiling the server actually enforces, read from the server's own
// constant so the two cannot drift.
func TestTheEvictionGraceFieldsAreBoundedByTheServersOwnCeiling(t *testing.T) {
	page, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"setGraceSec", "setGraceWait"} {
		field := fieldTag(t, string(page), id)
		if got := attr(field, "min"); got != "0" {
			t.Errorf("%s has min=%q, want 0 — zero is how the panel says \"the default\"", id, got)
		}
		want := strconv.Itoa(config.MaxEvictionWaitSec)
		if got := attr(field, "max"); got != want {
			t.Errorf("%s has max=%q, want the server's ceiling %s", id, got, want)
		}
	}
}

// A queue of requests waiting for room is live state, not a stored setting, so
// it is drawn where the models are and not behind the form's editing guard.
func TestThePanelSaysHowManyRequestsAreWaitingForRoom(t *testing.T) {
	for _, tc := range []struct {
		waiting int
		want    string
	}{
		{0, ""},
		{1, "1 request is waiting for memory to free up."},
		{4, "4 requests are waiting for memory to free up."},
	} {
		t.Run(fmt.Sprintf("%d waiting", tc.waiting), func(t *testing.T) {
			got := evalPanel(t, fmt.Sprintf("waitingLine(%d)", tc.waiting), "waitingLine")
			if got != tc.want {
				t.Errorf("waitingLine(%d) = %q, want %q", tc.waiting, got, tc.want)
			}
		})
	}
}

// fieldTag returns the whole <input> tag carrying the given id.
func fieldTag(t *testing.T, page, id string) string {
	t.Helper()
	re := regexp.MustCompile(`<input id="` + regexp.QuoteMeta(id) + `"[^>]*>`)
	m := re.FindString(page)
	if m == "" {
		t.Fatalf("no input with id %q in the control panel", id)
	}
	return m
}

// attr reads one attribute out of a tag.
func attr(tag, name string) string {
	re := regexp.MustCompile(name + `="([^"]*)"`)
	m := re.FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	return m[1]
}

// The panel offers the two intervals side by side, so it is where a maximum
// wait below the grace is easiest to type. The server refuses that pair, and a
// form that let it be posted would answer a save with an error the operator
// has to decode; say it beside the field instead.
//
// The rule is not restated here. Each case is taken to the server's own
// validator over the same two figures, and the hint has to appear exactly when
// that refuses — so config.validateGrace, through Validate, stays the one home
// of the rule and the panel is held to it rather than to a second copy of it
// in this file. The figures the hint names are the server's too: a blank field
// is the default, and the defaults reach the panel from the snapshot.
func TestThePanelWarnsExactlyWhenTheServerRefusesTheGracePair(t *testing.T) {
	defaults := fmt.Sprintf(`{"eviction_grace_sec":%d,"eviction_max_wait_sec":%d}`,
		config.DefaultEvictionGraceSec, config.DefaultEvictionMaxWaitSec)
	for _, tc := range []struct {
		name           string
		grace, maxWait string
	}{
		{"a maximum wait below the grace", "300", "60"},
		{"a maximum wait equal to the grace", "300", "300"},
		{"a maximum wait above the grace", "60", "300"},
		{"both left to the defaults", "", ""},
		{"a blank maximum wait against a grace above the default", "600", ""},
		{"a blank grace against a maximum wait below the default", "", "60"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := gracePairProbe(t, tc.grace, tc.maxWait)
			refused := cfg.Validate()
			got := evalPanel(t,
				fmt.Sprintf("graceWaitHint(%q, %q, %s)", tc.grace, tc.maxWait, defaults),
				"graceWaitHint")
			if refused != nil && got == "" {
				t.Fatalf("the server refuses grace=%q wait=%q (%v) and the panel says nothing — "+
					"the save is answered with an error the operator has to decode",
					tc.grace, tc.maxWait, refused)
			}
			if refused == nil {
				if got != "" {
					t.Fatalf("graceWaitHint(%q, %q) = %q, and the server accepts that pair — "+
						"the panel is warning about a save that would have worked",
						tc.grace, tc.maxWait, got)
				}
				return
			}
			// And it warns in the figures that would be in force, which for a
			// blank field is the server's default rather than a number the
			// panel chose.
			for _, want := range []string{
				fmt.Sprintf("%d s", cfg.MaxWaitSeconds()),
				fmt.Sprintf("%d s", cfg.GraceSeconds()),
			} {
				if !strings.Contains(got, want) {
					t.Errorf("graceWaitHint(%q, %q) = %q, want it to name %s — the figure the "+
						"server resolves that field to", tc.grace, tc.maxWait, got, want)
				}
			}
		})
	}
}

// gracePairProbe is a configuration valid but for the pair under test: the
// grace is on so the rule applies, it carries an API key so the exposure rule
// is not what refuses, and the idle timeout is off so the other cross-field
// rule on these two figures cannot fire either. What is left to refuse the
// pair is the rule the hint is about.
func gracePairProbe(t *testing.T, grace, maxWait string) config.Config {
	t.Helper()
	c := settingsProbeBase()
	c.EvictionGrace = true
	c.APIKey = "a-key-so-the-exposure-rule-is-not-the-one-refusing"
	c.IdleTimeoutSec = 0
	c.EvictionGraceSec = graceField(t, grace)
	c.EvictionMaxWaitSec = graceField(t, maxWait)
	return c
}

// graceField reads a field the way the server reads the setting behind it: a
// blank field is unset, and unset is the default.
func graceField(t *testing.T, value string) int {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("%q is not a number the field could hold", value)
	}
	return n
}

// The hint is bound to the server's defaults only while the panel hands it
// what the server sent. A call that dropped them would leave a blank field
// resolving to nothing, and the warning would quietly stop appearing; a hint
// that kept its own copy of the two figures is the drift this binding exists
// to end (iss-2609190029273153).
func TestTheGraceHintTakesItsDefaultsFromTheServer(t *testing.T) {
	src := readPanelSource(t)
	if body := extractFunction(t, src, "updateGraceHint"); !strings.Contains(body, "state.defaults") {
		t.Error("updateGraceHint no longer passes the server's defaults to graceWaitHint, " +
			"so a blank field resolves to a figure the panel made up")
	}
	hint := extractFunction(t, src, "graceWaitHint")
	for _, literal := range []string{
		strconv.Itoa(config.DefaultEvictionGraceSec),
		strconv.Itoa(config.DefaultEvictionMaxWaitSec),
	} {
		if strings.Contains(hint, literal) {
			t.Errorf("graceWaitHint carries %s — the server's own default, written out a second "+
				"time in the panel, which is the copy this hint was bound to the snapshot to be rid of",
				literal)
		}
	}
}

// And the same for the two placeholders. A blank field is the default, the
// placeholder is where the form says which figure that is, and the markup used
// to say it in two literals — config.DefaultEvictionGraceSec and
// config.DefaultEvictionMaxWaitSec written out a third time, bound by nothing
// (iss-2609190045306656). They are filled from the snapshot instead, so a Mac
// serving a different pair shows that pair and a panel that has not been told
// the defaults offers no figure at all rather than one it made up.
func TestTheGracePlaceholdersAreFilledFromTheServersDefaults(t *testing.T) {
	page, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"setGraceSec", "setGraceWait"} {
		if got := attr(fieldTag(t, string(page), id), "placeholder"); got != "" {
			t.Errorf("%s carries placeholder=%q in the markup — a figure nothing binds to the "+
				"server, which is the copy this field was bound to the snapshot to be rid of", id, got)
		}
	}

	// The renderer puts the snapshot's figures on the fields themselves.
	doc := evalPanelDOM(t, fmt.Sprintf("renderDefaults(%s);",
		fmt.Sprintf(`{"eviction_grace_sec":%d,"eviction_max_wait_sec":%d}`,
			config.DefaultEvictionGraceSec, config.DefaultEvictionMaxWaitSec)),
		"renderDefaults", "blankIsSentence", "intervalWords")
	for _, tc := range []struct{ id, want string }{
		{"setGraceSec", strconv.Itoa(config.DefaultEvictionGraceSec)},
		{"setGraceWait", strconv.Itoa(config.DefaultEvictionMaxWaitSec)},
	} {
		if got, _ := doc[tc.id]["placeholder"].(string); got != tc.want {
			t.Errorf("%s's placeholder is %q, want the server's default %s", tc.id, got, tc.want)
		}
	}

	// Told nothing, it offers nothing: an empty placeholder is a field with no
	// claim about what blank means, which beats a figure the panel chose.
	blank := evalPanelDOM(t, "renderDefaults(undefined);", "renderDefaults", "blankIsSentence", "intervalWords")
	for _, id := range []string{"setGraceSec", "setGraceWait"} {
		if got, _ := blank[id]["placeholder"].(string); got != "" {
			t.Errorf("%s's placeholder is %q for a snapshot carrying no defaults, want it blank", id, got)
		}
	}

	// And the settings renderer is what hands it the snapshot; a renderer
	// nothing calls would leave the fields blank on a live panel.
	if body := extractFunction(t, readPanelSource(t), "renderSettings"); !strings.Contains(body, "renderDefaults(state.defaults)") {
		t.Error("renderSettings no longer calls renderDefaults(state.defaults), " +
			"so the placeholders are never filled on a live panel")
	}
}

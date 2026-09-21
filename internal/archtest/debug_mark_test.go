package archtest_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/runtime"
)

// debugMarkReaders are the files allowed to name the per-model debug mark,
// each with the reason it is allowed to.
//
// The mark is the mirror of the statistics switch (statistics_switch_test.go):
// where that switch must never reach the runtime, this mark must never be
// derived from that switch, or from log_level, and must never reach anything
// but the launcher's level argument. adr-2609201008477513 narrows
// adr-2609061503319212 to exactly this — one model, one process lifetime,
// operator-invoked, never on by default, never reachable from the statistics
// switch or log_level — and this list is the instrument: a reader is a file
// that has to be added here, in a diff someone reviews, with the reason.
var debugMarkReaders = map[string]string{
	"internal/runtime/pool.go":      "holds the mark and consumes it at launch",
	"internal/runtime/launcher.go":  "the one level argument, and the bound on what an armed run writes",
	"internal/gateway/control.go":   "the route that arms and disarms, and the snapshot the panel polls",
	"internal/ui/static/app.js":     "the card's action and its pills, and the posture line",
	"internal/ui/static/index.html": "the plain-words paragraph and the markup",
}

// debugMarkNames are the ways the mark gets named: the Spec field and the
// resident entry's, the pool's map, the arming call, and the keys it is
// written down as on the snapshot.
var debugMarkNames = []string{"DebugLog", "debugArmed", "ArmDebugLog", `"debug_log"`, `"debug_armed"`}

// The mark reaches its readers and nothing else.
func TestTheDebugMarkIsNamedOnlyByItsReaders(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	subject := walkOptions{AlsoSkip: []string{"client", "build", "docs", "site-src"}}
	walkRepoFiles(t, repoRoot, subject, func(path string, d fs.DirEntry) error {
		name := d.Name()
		if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".js") && !strings.HasSuffix(name, ".html") {
			return nil
		}
		if strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(b)
		for _, field := range debugMarkNames {
			if !strings.Contains(src, field) {
				continue
			}
			if _, allowed := debugMarkReaders[filepath.ToSlash(rel)]; allowed {
				return nil
			}
			t.Errorf("%s names %s, so the per-model debug mark reaches it. The mark is read by the "+
				"launcher's level argument and by nothing else (adr-2609201008477513). If this is "+
				"genuinely a new reader, it needs a decision record and an entry in debugMarkReaders "+
				"saying why", rel, field)
			return nil
		}
		return nil
	})
}

// The mark is derived from nothing: the two runtime files that hold and spend
// it name neither the statistics switch nor log_level, and neither does the
// body of the handler that arms it.
func TestTheDebugMarkIsDerivedFromNothingElse(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// ".LogLevel" rather than the bare word: config's constants and the field
	// are reached that way, and the launcher's own debugLogLevel contains the
	// bare word.
	forbidden := []string{".Statistics", "Statistics bool", ".LogLevel", `"log_level"`}
	for _, rel := range []string{"internal/runtime/pool.go", "internal/runtime/launcher.go"} {
		src := readRepoFile(t, repoRoot, rel)
		for _, name := range forbidden {
			if strings.Contains(src, name) {
				t.Errorf("%s names %s, from which the debug mark must never be derived", rel, name)
			}
		}
	}

	// The arming handler's own body. It is found by name, so a handler under
	// another name is a failure here rather than an exemption.
	control := readRepoFile(t, repoRoot, "internal/gateway/control.go")
	const handler = "func (c *Control) handleDebugLog("
	i := strings.Index(control, handler)
	if i < 0 {
		t.Fatalf("internal/gateway/control.go has no %s: the route that arms debug logging is served by a handler of that name", strings.TrimSuffix(handler, "("))
	}
	body := control[i+len(handler):]
	if j := strings.Index(body, "\nfunc "); j >= 0 {
		body = body[:j]
	}
	for _, name := range forbidden {
		if strings.Contains(body, name) {
			t.Errorf("handleDebugLog names %s: arming the mark must read nothing from the statistics switch or log_level, and write nothing to them", name)
		}
	}
}

// The list above is only a boundary while every entry on it is a file that
// exists: a stale entry exempts nothing, and a renamed reader would leave its
// exemption behind for the next file to inherit.
func TestDebugMarkReadersAllExist(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for rel, why := range debugMarkReaders {
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel))); err != nil {
			t.Errorf("debugMarkReaders lists %s (%s), which is not in the tree", rel, why)
		}
	}
}

// The panel says in plain words what arming debug logging writes down, in
// one place: a paragraph in the Models pane, which the card's button takes its
// title from at render time so the two cannot drift. The phrases are the
// promise's load-bearing parts — what is written, whoever sent it, that it
// begins at the next start and changes nothing now, the bound, the kept file,
// that clients are not told, and that a no-transcript model refuses.
func TestThePanelSaysWhatDebugLoggingWrites(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	markup := readRepoFile(t, repoRoot, "internal/ui/static/index.html")
	const blurb = `id="debugLogBlurb"`
	i := strings.Index(markup, blurb)
	if i < 0 {
		t.Fatal("internal/ui/static/index.html has no element with id debugLogBlurb, so the panel has no plain-words paragraph for debug logging")
	}
	para := markup[i:]
	if j := strings.Index(para, "</p>"); j >= 0 {
		para = para[:j]
	}
	for _, want := range []string{
		"off unless you arm it",
		"every request sent to it and every answer it produced",
		"prompts and the completions, whoever sent them",
		"Dessau's own probes",
		"next start",
		"changes nothing about the run going on now",
		"unload the model",
		"back to the ordinary level",
		fmt.Sprintf("stops at %d MB", runtime.DebugLogMaxBytes>>20),
		"previous run's file is kept",
		"nothing is sent anywhere",
		"clients are not told",
		"keeps no transcript refuses this",
	} {
		if !containsAll(para, want) {
			t.Errorf("the debug-logging paragraph does not say %q", want)
		}
	}

	panel := readRepoFile(t, repoRoot, "internal/ui/static/app.js")
	if !strings.Contains(panel, "debugLogBlurb") {
		t.Error("app.js does not read the debugLogBlurb paragraph, so the card's button cannot take its title from it")
	}
	for _, label := range []string{"'Debug logging'", "'Stop debug logging'"} {
		if !strings.Contains(panel, label) {
			t.Errorf("app.js has no card action labelled %s", label)
		}
	}
	if !strings.Contains(panel, "/api/models/debug-log") {
		t.Error("app.js does not post to /api/models/debug-log")
	}
}

// Armed and running are different runs and are drawn differently: a pill for
// a model with a mark waiting, another for a model whose running process was
// launched with it, and nothing on a model that is neither. The armed fact is
// read from the snapshot's debug_armed and the running fact from the resident
// entry's debug_log, so the pills move from one to the other without a
// reload. The posture line names the models while any is at either.
func TestTheCardDrawsTheDebugPills(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	panel := readRepoFile(t, repoRoot, "internal/ui/static/app.js")
	for _, pill := range []struct{ text, readsFrom string }{
		{">debug armed<", "debug_armed"},
		{">logging at debug<", "debug_log"},
	} {
		i := strings.Index(panel, pill.text)
		if i < 0 {
			t.Errorf("app.js draws no pill reading %s", strings.Trim(pill.text, "<>"))
			continue
		}
		// Conditional on the state field: the pill is drawn in the statement
		// that reads the field, so a model without it draws nothing. The
		// window is the pill's own line and the two before it.
		lines := strings.Split(panel[:i], "\n")
		from := len(lines) - 3
		if from < 0 {
			from = 0
		}
		window := strings.Join(lines[from:], "\n") + panel[i:i+len(pill.text)]
		if !strings.Contains(window, pill.readsFrom) {
			t.Errorf("the %s pill is not drawn from %s, so a model without it might carry the pill", strings.Trim(pill.text, "<>"), pill.readsFrom)
		}
	}
	if !strings.Contains(panel, "id: 'debug_log'") {
		t.Error("postureLines has no debug_log line, so a model at debug is not named on the posture page")
	}
}

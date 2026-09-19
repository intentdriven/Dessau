package archtest_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Every rule the OPINIONS domain injects ends with a pointer of the form
// "see .abcd/development/principles/<name>.md", and the directory carries one
// file per pointer. That is a cross-surface promise between two things nothing
// else connects: the rule text lives in the abcd binary, the prose lives in
// this repository. The pointers dangled once already
// (iss-2609190029331004) -- the directory did not exist at all -- and an
// upstream rule added, renamed or reworded would dangle them again with no
// signal. iss-2609190042447070 asks for the promise to be armed the way every
// other cross-surface promise here is: a test, not a habit.
//
// WHAT IS CHECKED, against `abcd rules OPINIONS --json` (the machine surface of
// the same render the hook injects):
//
//   - every pointer names a file that exists under
//     .abcd/development/principles/;
//   - every such file quotes its whole rule sentence verbatim, in the
//     blockquote under its "## The rule" heading -- so a reworded rule upstream
//     fails here rather than leaving a file that misquotes what it explains;
//   - README.md in that directory links every file beside it, and every file
//     it links exists.
//
// WHAT IS NOT, stated so a green run is not over-read. Nothing here judges the
// prose of a principle file beyond its quoted rule: the "Why" and "How to apply
// here" sections are a human's work and can age without this failing. A rule
// that carries no pointer at all (the domain's closing line names the directory
// rather than a file) is not required to have one.

const principlesDir = ".abcd/development/principles"

// principlePointer matches the pointer every pointing rule ends with. The
// trailing "." is the sentence's, not the path's.
var principlePointer = regexp.MustCompile(`see \.abcd/development/principles/([a-z0-9-]+\.md)`)

// ruleBlockquote matches the blockquote under a principle file's "## The rule"
// heading: the run of "> " lines that follows it.
var ruleBlockquote = regexp.MustCompile(`(?m)^## The rule\n\n((?:>.*\n)+)`)

// opinionsRules returns the rule sentences of the OPINIONS domain as the abcd
// binary renders them. It SKIPS LOUDLY when abcd is not on PATH: a skip here
// means the pointers were not checked at all, and a stage that no-ops says so
// -- which is itself one of the rules this test holds.
func opinionsRules(t *testing.T) []string {
	t.Helper()
	bin, err := exec.LookPath("abcd")
	if err != nil {
		t.Skip("SKIPPED, NOT PASSED: abcd is not on PATH, so the OPINIONS rule " +
			"pointers were NOT checked against " + principlesDir + " -- this check did not run")
	}
	out, err := exec.Command(bin, "rules", "OPINIONS", "--json").Output()
	if err != nil {
		t.Fatalf("`abcd rules OPINIONS --json` failed: %v", err)
	}
	var domain struct {
		Name  string   `json:"name"`
		Rules []string `json:"rules"`
	}
	if err := json.Unmarshal(out, &domain); err != nil {
		t.Fatalf("`abcd rules OPINIONS --json` is not the expected object: %v", err)
	}
	if domain.Name != "OPINIONS" {
		t.Fatalf("asked abcd for OPINIONS and got %q", domain.Name)
	}
	if len(domain.Rules) == 0 {
		t.Fatal("`abcd rules OPINIONS --json` carries no rules; the pointers cannot be checked")
	}
	return domain.Rules
}

// TestEveryOpinionsPointerResolvesToAPrinciplesFile walks the pointers and
// holds each one to a file that exists and quotes it.
func TestEveryOpinionsPointerResolvesToAPrinciplesFile(t *testing.T) {
	root := repoRootDir(t)
	rules := opinionsRules(t)

	pointed := 0
	for _, rule := range rules {
		m := principlePointer.FindStringSubmatch(rule)
		if m == nil {
			continue
		}
		pointed++
		name := m[1]
		rel := principlesDir + "/" + name
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("the OPINIONS rule %q points at %s, which cannot be read: %v",
				firstWords(rule), rel, err)
			continue
		}
		quote := ruleBlockquote.FindStringSubmatch(string(raw))
		if quote == nil {
			t.Errorf("%s has no blockquote under a `## The rule` heading; "+
				"it must quote the rule it explains verbatim", rel)
			continue
		}
		var quoted []string
		for _, line := range strings.Split(quote[1], "\n") {
			quoted = append(quoted, strings.TrimPrefix(strings.TrimPrefix(line, ">"), " "))
		}
		// collapse (honest_marking_test.go) folds every run of whitespace to
		// one space, so a sentence wrapped across blockquote lines compares
		// equal to the single-line rule it quotes.
		if got, want := collapse(strings.Join(quoted, " ")), collapse(rule); got != want {
			t.Errorf("%s does not quote its rule verbatim.\n  quoted: %s\n  rule:   %s",
				rel, got, want)
		}
	}
	if pointed == 0 {
		t.Errorf("no OPINIONS rule carries a `see %s/<name>.md` pointer; "+
			"either the rules changed shape or this test stopped reading them", principlesDir)
	}
}

// TestThePrinciplesReadmeIndexesEveryFile holds the directory's own index:
// a file beside README.md that README.md does not link is unreachable, and a
// link README.md carries to a file that is gone is the dangling pointer this
// whole test exists about, one level down.
func TestThePrinciplesReadmeIndexesEveryFile(t *testing.T) {
	root := repoRootDir(t)
	dir := filepath.Join(root, filepath.FromSlash(principlesDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("%s cannot be read: %v", principlesDir, err)
	}
	readme := readRepoFile(t, root, principlesDir+"/README.md")

	var files []string
	for _, e := range entries {
		if e.IsDir() || e.Name() == "README.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		files = append(files, e.Name())
	}
	if len(files) == 0 {
		t.Fatalf("%s holds no principle file beside its README", principlesDir)
	}
	sort.Strings(files)
	for _, name := range files {
		if !strings.Contains(readme, "("+name+")") {
			t.Errorf("%s/README.md does not index %s; every file in the directory is listed there",
				principlesDir, name)
		}
	}

	linked := regexp.MustCompile(`\]\(([a-z0-9-]+\.md)\)`)
	for _, m := range linked.FindAllStringSubmatch(readme, -1) {
		if _, err := os.Stat(filepath.Join(dir, m[1])); err != nil {
			t.Errorf("%s/README.md links %s, which is not there: %v", principlesDir, m[1], err)
		}
	}
}

// firstWords is the head of a rule sentence, for an error message that names
// the rule without reprinting a paragraph of it.
func firstWords(rule string) string {
	if i := strings.Index(rule, ":"); i > 0 {
		return rule[:i]
	}
	if len(rule) > 60 {
		return rule[:60] + "..."
	}
	return rule
}

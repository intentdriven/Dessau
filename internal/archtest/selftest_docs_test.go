package archtest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/selftest"
)

// The self-test's reference page is the promise the results file is held to:
// every field a run or a test carries is named there, so a figure nobody can
// look up is a test failure rather than a surprise.
func TestTheSelfTestReferenceNamesEveryField(t *testing.T) {
	page := readDoc(t, "self-test-reference.md")
	for _, field := range append(selftest.RunFields(), selftest.TestFields()...) {
		if !strings.Contains(page, "`"+field+"`") {
			t.Errorf("docs/self-test-reference.md does not name the field `%s`", field)
		}
	}
	for _, name := range []string{selftest.TestPP512, selftest.TestTG128, selftest.TestTG128Parallel + "N"} {
		if !strings.Contains(page, "`"+name+"`") {
			t.Errorf("docs/self-test-reference.md does not name the test `%s`", name)
		}
	}
	for _, outcome := range []string{selftest.OutcomeOK, selftest.OutcomeYielded, selftest.OutcomeStopped, selftest.OutcomeFailed} {
		if !strings.Contains(page, "`"+outcome+"`") {
			t.Errorf("docs/self-test-reference.md does not name the outcome `%s`", outcome)
		}
	}
	for _, s := range []string{selftest.FileName, fmt.Sprintf("%d MiB", selftest.DefaultMaxBytes>>20)} {
		if !strings.Contains(page, s) {
			t.Errorf("docs/self-test-reference.md does not say %q", s)
		}
	}
}

// The pages state what the code never writes, and are linked from where a
// reader starts.
func TestTheSelfTestPagesAreLinkedAndSayWhatIsNeverWritten(t *testing.T) {
	howTo := readDoc(t, "self-test.md")
	for _, want := range []string{"never records", "prompt", "answer", "self-test-reference.md"} {
		if !strings.Contains(howTo, want) {
			t.Errorf("docs/self-test.md does not say %q", want)
		}
	}
	readme, err := os.ReadFile(filepath.Join(repoRootDir(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "docs/self-test.md") {
		t.Error("README.md does not link docs/self-test.md")
	}
}

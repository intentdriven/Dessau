package archtest_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// CI runs on a push to main, on every pull request and in the merge queue,
// and not on a push to any other branch: a branch with a pull request is
// already tested by the pull_request run, so a push run beside it doubled the
// load on the macOS runners and starved the merge queue
// (iss-2609200823571893). A branch with no pull request is tested when one is
// opened.
func TestCIRunsOnPushesToMainOnly(t *testing.T) {
	ci := withoutComments(readRepoFile(t, repoRootDir(t), filepath.Join(".github", "workflows", "ci.yml")))
	on := ci[strings.Index(ci, "\non:"):]
	if end := strings.Index(on, "\nconcurrency:"); end > 0 {
		on = on[:end]
	}
	push := on[strings.Index(on, "push:"):]
	if end := strings.Index(push, "pull_request:"); end > 0 {
		push = push[:end]
	}
	var branches []string
	for _, line := range strings.Split(push, "\n") {
		if item, ok := strings.CutPrefix(strings.TrimSpace(line), "- "); ok {
			branches = append(branches, strings.Trim(item, `'"`))
		}
	}
	if len(branches) != 1 || branches[0] != "main" {
		t.Errorf("ci.yml runs on pushes to %v, want main alone; every other branch is tested through its pull request", branches)
	}
	for _, trigger := range []string{"pull_request:", "merge_group:"} {
		if !strings.Contains(on, trigger) {
			t.Errorf("ci.yml no longer runs on %s", strings.TrimSuffix(trigger, ":"))
		}
	}
}

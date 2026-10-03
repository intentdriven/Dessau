package archtest_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// The server code builds and its tests run on Linux, so that an implementation
// session in a Linux container can check its own work. Dessau still only ever
// runs on a Mac, and nothing on a Mac notices a Darwin-only symbol creeping
// back into a file every platform compiles: ci.yml's linux job does, and this
// holds that job in place — on a Linux runner, vetting and testing the whole
// module rather than a hand-picked part of it.
func TestCIVetsAndTestsTheModuleOnLinux(t *testing.T) {
	root := repoRootDir(t)
	if label := workflowRunners(t, root)["ci.yml:linux"]; !strings.HasPrefix(label, "ubuntu-") {
		t.Fatalf("ci.yml's linux job runs on %q, want an ubuntu runner: it is the only thing that compiles the module for Linux", label)
	}
	job := withoutComments(workflowJob(t, readRepoFile(t, root, filepath.Join(".github", "workflows", "ci.yml")), "linux"))
	for _, step := range []string{"go vet ./...", "go vet -tags prod ./...", "go test ./..."} {
		if !strings.Contains(job, "run: "+step+"\n") {
			t.Errorf("ci.yml's linux job does not run `%s`", step)
		}
	}
}

package runtime

import (
	"strings"
	"testing"
)

// A model server is given no TCP address to listen on (iss-2610030846581757).
//
// The pinned server binds whatever --host and --port it is handed, and
// nothing on a loopback port can tell one account's process from another's:
// any account on the Mac could reach it there and have it load a directory of
// its choosing. The launch carries neither flag, and the interpreter runs
// isolated (-I), so no module from the working directory, the model directory
// or a PYTHONPATH can stand in for the launcher Dessau hands it.
func TestTheModelServerIsLaunchedWithNoTCPAddress(t *testing.T) {
	argv := launchArgs(Spec{RepoID: "org/a", ModelPath: "/models/org/a", DecodeConcurrency: 4})
	for _, a := range argv {
		for _, flag := range []string{"--host", "--port"} {
			if a == flag || strings.HasPrefix(a, flag+"=") {
				t.Errorf("the model server is launched with %s: %v", a, argv)
			}
		}
	}
	if len(argv) == 0 || argv[0] != "-I" {
		t.Errorf("the interpreter is not started isolated (-I first): %v", argv)
	}
}

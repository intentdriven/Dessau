package archtest_test

import (
	"strings"
	"testing"
)

// The unloading page names both routes that unload a model, the switch that
// turns the program's one off, and what an unload is recorded as — the
// reference a script writer reads before choosing one (itd-2610031024247803;
// iss-2610031018046897).
func TestTheUnloadPageNamesBothRoutes(t *testing.T) {
	page := readDoc(t, "unload-reference.md")
	for _, want := range []string{
		"POST /v1/dessau/unload", "POST /api/models/unload", "POST /api/models/load",
		"api_unload_off", "`released`", "`unloaded`",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("docs/unload-reference.md does not name %s", want)
		}
	}
}

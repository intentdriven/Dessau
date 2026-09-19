package ui

import (
	"strings"
	"testing"
)

// The picker searches one organisation and looks a typed repository id up
// exactly. Neither is discoverable from a search box that says nothing, and a
// person who cannot see the second route reads a model the search does not
// surface as a model the server cannot have. Both are stated where the text is
// typed.
func TestSearchBoxStatesHowToReachAnotherAccount(t *testing.T) {
	page, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	hint := string(page)
	for _, want := range []string{
		"author:",                    // the prefix that searches another account
		"repository id",              // the typed-in-full route
		"<code>mlx-community</code>", // the organisation the plain search asks
	} {
		if !strings.Contains(hint, want) {
			t.Errorf("the Find Models tab does not mention %q; the picker's two routes are stated where the person types", want)
		}
	}
}

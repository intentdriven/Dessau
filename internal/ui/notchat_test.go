package ui

import (
	"regexp"
	"testing"
)

// The card offers Load only for a model the server says can chat, and Unload
// for any loaded model (iss-2610031010371709).
func TestTheCardOffersLoadOnlyForAChatModel(t *testing.T) {
	src := readPanelSource(t)
	if !regexp.MustCompile(`\} else if \(m\.chat\) \{\s*actions\.append\(btn\('Load'`).MatchString(src) {
		t.Error("the card's Load is not guarded by the model's chat verdict")
	}
}

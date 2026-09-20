package archtest_test

import (
	"regexp"
	"strings"
	"testing"
)

// The file that owns the chat verdict. ChatRule.Matches is the operator's
// rule over the Hub's words, and it is half of the answer: a model the Hub
// has said nothing about — one adopted from the shared cache by an account
// that never downloaded it — is judged by its chat template instead. Every
// surface that says whether a model can chat must read the whole answer from
// registry.Model.CanChat, so that the two halves cannot come apart the way
// they did when the models list applied the rule itself and every adopted
// model read chat: false (iss-2609202237468921).
const chatVerdictHome = "internal/registry/registry.go"

// The file that defines the rule; its own body is not a caller.
const chatRuleFile = "internal/config/chatrule.go"

// No shipping Go file outside the one home applies the chat rule to a model.
//
// The check is name-shaped: the rule's method is Matches, and nothing else in
// this tree is called Matches, so any `.Matches(` outside the home is a
// second reading of the rule — a surface that would judge an adopted model by
// words it does not carry.
func TestTheChatVerdictHasOneHome(t *testing.T) {
	matches := regexp.MustCompile(`\.Matches\(`)
	root := repoRootDir(t)
	seenHome := false
	forEachShippingGoFile(t, root, func(rel string, body string) {
		if rel == chatRuleFile {
			return
		}
		if rel == chatVerdictHome {
			if matches.MatchString(body) {
				seenHome = true
			}
			return
		}
		for i, line := range strings.Split(body, "\n") {
			if matches.MatchString(line) {
				t.Errorf("%s:%d applies the chat rule itself; read registry.Model.CanChat, the one home of the verdict: %s",
					rel, i+1, strings.TrimSpace(line))
			}
		}
	})
	if !seenHome {
		t.Errorf("%s no longer calls ChatRule.Matches, so this test is guarding the wrong file", chatVerdictHome)
	}
}

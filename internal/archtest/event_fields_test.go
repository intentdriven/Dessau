package archtest_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/stats"
)

// Every field a load, removal or footprint event carries is named on the
// reference page, as every request field already is.
func TestTheStatisticsPageNamesEveryEventField(t *testing.T) {
	page := readDoc(t, "statistics-store-reference.md")
	for _, field := range stats.EventFields() {
		if !strings.Contains(page, "`"+field+"`") {
			t.Errorf("docs/statistics-store-reference.md does not name the event field `%s`", field)
		}
	}
}

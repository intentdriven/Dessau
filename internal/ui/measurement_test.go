package ui

import (
	"reflect"
	"regexp"
	"testing"
)

// The three measurement views are drawn from the history the endpoint
// serves, and each row is a pure function a test can hold to its figures.
func TestTheMeasurementViewsReachThePanel(t *testing.T) {
	src := readPanelSource(t)
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`\(h\.prompt_sizes \|\| \[\]\)\.map\(sizesRowHtml\)`),
		regexp.MustCompile(`\(h\.overrides \|\| \[\]\)\.map\(overridesRowHtml\)`),
		regexp.MustCompile(`\(h\.footprints \|\| \[\]\)\.map\(footprintRowHtml\)`),
	} {
		if !want.MatchString(src) {
			t.Errorf("the control panel no longer matches %s — that view is then drawn by nothing", want)
		}
	}
	sizes := evalPanelValue(t, `sizesRow({model:"org/m", declared_context:131072, served_context:65536, requests:10, buckets:[4,3,2,1,0], refused_for_size:0, largest_estimate:60000})`, "tokensLabel", "sizesRow")
	if sizes["declared"] != "128K" || sizes["served"] != "64K" || sizes["largest"] != "58K" || sizes["refused"] != float64(0) {
		t.Errorf("sizesRow = %v", sizes)
	}
	if bands, _ := sizes["bands"].([]any); !reflect.DeepEqual(bands, []any{float64(4), float64(3), float64(2), float64(1)}) {
		t.Errorf("bands = %v", sizes["bands"])
	}
	over := evalPanelValue(t, `overridesRow({model:"org/m", requests:4, by_parameter:{temperature:2, top_p:0, top_k:0, min_p:0, max_tokens:1}})`, "overridesRow")
	if over["temperature"] != "50%" || over["max_tokens"] != "25%" || over["top_p"] != "0%" {
		t.Errorf("overridesRow = %v", over)
	}
	none := evalPanelValue(t, `overridesRow({model:"org/m", requests:0, by_parameter:{}})`, "overridesRow")
	if none["temperature"] != "—" {
		t.Errorf("no requests gave %v", none["temperature"])
	}
	fp := evalPanelValue(t, `footprintRow({model:"org/m", points:[{at:1,bytes:1073741824},{at:2,bytes:3221225472},{at:3,bytes:2147483648}]})`, "bytes", "sparkline", "footprintRow")
	if fp["samples"] != float64(3) || fp["lowest"] != "1.0 GB" || fp["highest"] != "3.0 GB" || fp["latest"] != "2.0 GB" {
		t.Errorf("footprintRow = %v", fp)
	}
	if fp["line"] != "▁█▅" {
		t.Errorf("sparkline = %q, want the three points at their heights", fp["line"])
	}
}

package analytics

import "testing"

func TestEventNameRE(t *testing.T) {
	if !EventNameRE.MatchString(`make.sure-all_these.Are-Allowed12`) {
		t.Fatal("Event name regular expression fails to match allowed characters")
	}
}

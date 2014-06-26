package lvm

import (
	"testing"
)

func TestParseVGDisplay(t *testing.T) {
	out := []byte(`  mysql-data-vg:r/w:772:-1:0:1:1:-1:0:4:4:41926656:4096:10236:10236:0:vIadoo-aom9-jid7-3AgF-LYVL-xkcq-l8PXi1
`)
	vgs, err := parseVGDisplay(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(vgs) != 1 {
		t.Fatalf("Expected 1 vgs info not %d", len(vgs))
	}
	vg := vgs["mysql-data-vg"]
	if vg == nil {
		t.Fatalf("Volume group %s not found", "mysql-data-vg")
	} else if vg.Name != "mysql-data-vg" {
		t.Fatalf("Expected 'mysql-data-vg' for Name not '%s'", vg.Name)
	}
}

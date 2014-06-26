package dmsetup

import (
	"testing"
)

func TestParseDMInfo(t *testing.T) {
	out := []byte(`Name|Maj|Min|Stat|Open|Targ|Event|UUID
mysql--data--vg-mysql--data--lv|252|0|L--w|1|1|0|LVM-vIadooaom9jid73AgFLYVLxkcql8PXi1ci3y4Wq5PoFqwmgykNZmK7aCEKVBhqn0
mysql-data-encrypted|252|1|L--w|1|2|3|CRYPT-LUKS1-ca323f61f2cd4554b0ae8c344f526281-mysql-data-encrypted
docker-202:1-282520-base|252|3|L--w|0|1|0|
docker-202:1-282520-pool|252|2|L--w|15|1|0|
docker-202:1-282520-28179a4deabb37187d7c4561a9bb4c88bdb3745b84523353b23755aa34f0b4f9|252|12|L--w|1|1|0|
`)
	devices, err := parseDMInfo(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 5 {
		t.Fatalf("Expected 5 dm devices not %d", len(devices))
	}
	if d := devices["mysql-data-encrypted"]; d == nil {
		t.Fatalf("Device mysql-data-encrypted not found")
	} else if d.Major != 252 {
		t.Fatalf("Expected major 252 instead of %d", d.Major)
	} else if d.Minor != 1 {
		t.Fatalf("Expected minor 1 instead of %d", d.Minor)
	} else if d.UUID != "CRYPT-LUKS1-ca323f61f2cd4554b0ae8c344f526281-mysql-data-encrypted" {
		t.Fatalf("Wrong UUID")
	} else if d.Stat != "L--w" {
		t.Fatalf("Wrong Stat")
	} else if d.Open != 1 {
		t.Fatalf("Wrong Open")
	} else if d.Targets != 2 {
		t.Fatalf("Wrong Targets")
	} else if d.Event != 3 {
		t.Fatalf("Wrong Events")
	}
}

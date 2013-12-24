package lvm

import (
	"testing"
)

func TestParseLVS(t *testing.T) {
	out := []byte(`LV|VG|Attr|#Str|Type|SSize|Devices
mysql-data-lv|mysql-data-vg|-wi-ao---|4|striped|39.98g|/dev/xvdh(0),/dev/xvdi(0),/dev/xvdj(0),/dev/xvdk(0)
`)
	infos, err := parseLVS(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("Expected 1 lvs info not %d", len(infos))
	}
	info := infos[0]
	if info.Name != "mysql-data-lv" {
		t.Fatalf("Expected 'mysql-data-lv' for Name not '%s'", info.Name)
	} else if info.VGName != "mysql-data-vg" {
		t.Fatalf("Expected 'mysql-data-vg' for VGName not '%s'", info.VGName)
	} else if info.Type != "striped" {
		t.Fatalf("Expected 'striped' for Type not '%s'", info.Type)
	} else if len(info.Devices) != 4 {
		t.Fatalf("Expected 4 devices, not %d", len(info.Devices))
	} else if info.Devices[0] != "/dev/xvdh" {
		t.Fatalf("Expected '/dev/xvdh' for first device, not '%s'", info.Devices[0])
	}
}

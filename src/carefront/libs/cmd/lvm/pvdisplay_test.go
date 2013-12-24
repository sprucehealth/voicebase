package lvm

import (
	"testing"
)

func TestParsePVDisplay(t *testing.T) {
	out := []byte(`  /dev/xvdh:mysql-data-vg:20971520:-1:8:8:-1:4096:2559:0:2559:myo3nn-uceC-fxtm-mupt-zuQ6-pQaj-0gCsT4
  /dev/xvdi:mysql-data-vg:20971520:-1:8:8:-1:4096:2559:0:2559:mI5gsq-CyEF-kFOB-pw3m-QXef-w28s-on9gAI
  /dev/xvdj:mysql-data-vg:20971520:-1:8:8:-1:4096:2559:0:2559:1NsHeA-gVP0-Bn75-CYu2-Nfsf-E43g-xEccJr
  /dev/xvdk:mysql-data-vg:20971520:-1:8:8:-1:4096:2559:0:2559:j3HLMd-72g4-vtdN-V4CX-FrbR-QEOx-yqfQgc
`)
	pvs, err := parsePVDisplay(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(pvs) != 4 {
		t.Fatalf("Expected 4 pv info not %d", len(pvs))
	}
	pv := pvs["/dev/xvdh"]
	if pv == nil {
		t.Fatalf("Physical volume %s not found", "/dev/xvdh")
	} else if pv.Name != "/dev/xvdh" {
		t.Fatalf("Expected '/dev/xvdh' for Name not '%s'", pv.Name)
	} else if pv.VGName != "mysql-data-vg" {
		t.Fatalf("Expected 'mysql-data-vg' for VGName not '%s'", pv.VGName)
	}
}

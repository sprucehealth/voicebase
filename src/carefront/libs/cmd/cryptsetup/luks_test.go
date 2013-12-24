package cryptsetup

import (
	"testing"
)

func TestParseStatus(t *testing.T) {
	out := []byte(`/dev/mapper/mysql-data-encrypted is active and is in use.
  type:    LUKS1
  cipher:  aes-cbc-essiv:sha256
  keysize: 256 bits
  device:  /dev/mapper/mysql--data--vg-mysql--data--lv
  offset:  4096 sectors
  size:    83849216 sectors
  mode:    read/write
`)
	status, err := parseStatus(out)
	if err != nil {
		t.Fatal(err)
	}
	if status.Type != "LUKS1" {
		t.Fatalf("Expected type LUKS1 not %s", status.Type)
	}
	if status.Device != "/dev/mapper/mysql--data--vg-mysql--data--lv" {
		t.Fatalf("Wrong device")
	}
	if status.Offset != 4096 {
		t.Fatalf("Expected offset 4096 not %d", status.Offset)
	}
	if status.Size != 83849216 {
		t.Fatalf("Expected size 83849216 not %d", status.Size)
	}
	if status.Cipher != "aes-cbc-essiv:sha256" {
		t.Fatalf("Expected cipher aes-cbc-essiv:sha256 not %s", status.Cipher)
	}
	if status.Keysize != 256 {
		t.Fatalf("Expected keysize 256 not %d", status.Keysize)
	}
	if status.Mode != "read/write" {
		t.Fatalf("Expected mode read/write not %s", status.Mode)
	}
}

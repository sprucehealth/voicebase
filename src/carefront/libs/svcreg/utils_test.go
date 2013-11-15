package svcreg

import "testing"

func TestAddr(t *testing.T) {
	addr, err := Addr()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Address: %+v", addr)
}

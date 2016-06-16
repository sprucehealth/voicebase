package worker

import (
	"github.com/sprucehealth/backend/libs/test"

	"testing"
)

func TestParseAddress(t *testing.T) {

	addr, err := parseAddress("Joe Schmoe (Joe) <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("Joe Schmoe <Joe> <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("Joe <3 Schmoe <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("Joe Schmoe <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("\"Joe Schmoe\" <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("<joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("joe@schmoe.com")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("I<joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress(" 		<joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)
}

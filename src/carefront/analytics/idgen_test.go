package analytics

import "testing"

func TestIDGen(t *testing.T) {
	id, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ID: %d\n", id)
}

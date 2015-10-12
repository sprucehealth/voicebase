package conc

import "testing"

func TestMapAccess(t *testing.T) {
	c := NewMap()
	noVal := c.Get("TESTING")
	if noVal != nil {
		t.Fatal("Expected no value to exist but something did")
	}
	c.Set("TESTING", "1")
	val := c.Get("TESTING")
	if val.(string) != "1" {
		t.Fatalf("Expected value 1 to exist but didnt")
	}

	p := NewParallel()

	p.Go(func() error {
		c.Set("TESTING2", "2")
		return nil
	})

	p.Go(func() error {
		c.Set("TESTING3", "3")
		return nil
	})

	if err := p.Wait(); err != nil {
		t.Fatal(err.Error())
	}

	if c.Get("TESTING2").(string) != "2" {
		t.Fatalf("Expected value 2 to exist but it didnt")
	}

	// now lets get a snapshot to ensure all three values are present
	snapshot := c.Snapshot()
	if snapshot["TESTING"] == nil {
		t.Fatalf("Expected value for TESTING to be present")
	} else if snapshot["TESTING2"] == nil {
		t.Fatalf("Expected value for TESTING2 to be present")
	} else if snapshot["TESTING3"] == nil {
		t.Fatalf("Expected value for TESTING3 to be present")
	}

}

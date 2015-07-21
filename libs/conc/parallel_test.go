package conc

import (
	"errors"
	"testing"
)

func TestError(t *testing.T) {
	var err error = Errors([]error{errors.New("foo")})
	if s := err.Error(); s != "[foo]" {
		t.Fatalf("Expected '[foo]' got '%s'", s)
	}
}

func TestParallel(t *testing.T) {
	t.Parallel()

	p := NewParallel()
	if err := p.Wait(); err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	x := 0
	p = NewParallel()
	p.Go(func() error { x++; return nil })
	if err := p.Wait(); err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	if x != 1 {
		t.Fatalf("Expected 1, got %d", x)
	}

	x = 0
	y := 0
	p = NewParallel()
	p.Go(func() error { x++; return nil })
	p.Go(func() error { y += 2; return nil })
	if err := p.Wait(); err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	if x != 1 {
		t.Fatalf("Expected 1, got %d", x)
	}
	if y != 2 {
		t.Fatalf("Expected 2, got %d", y)
	}

	p = NewParallel()
	p.Go(func() error { return errors.New("foo") })
	if err := p.Wait(); err == nil {
		t.Fatal("Expected an error")
	} else if e, ok := err.(Errors); !ok {
		t.Fatalf("Expected a Errors got %T", err)
	} else if len(e) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(e))
	} else if e[0].Error() != "foo" {
		t.Fatalf("Expected 'foo', got '%s'", e[0])
	}

	p = NewParallel()
	p.Go(func() error { panic("BOOM") })
	p.Go(func() error { return errors.New("POW") })
	p.Go(func() error { panic(errors.New("SURPRISE")) })
	if err := p.Wait(); err == nil {
		t.Fatal("Expected an error")
	} else if e, ok := err.(Errors); !ok {
		t.Fatalf("Expected a Errors got %T", err)
	} else if len(e) != 3 {
		t.Fatalf("Expected 3 errors, got %d", len(e))
	} else {
		for _, e := range e {
			if e.Error() != "runtime error: BOOM" && e.Error() != "POW" && e.Error() != "SURPRISE" {
				t.Fatalf("Unexpected error %s", e)
			}
		}
	}
}

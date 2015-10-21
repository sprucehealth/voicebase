package response

import "testing"

func TestRounding(t *testing.T) {
	expected := 1.0
	res := roundToClosestHalve(1.2)
	if res != expected {
		t.Fatalf("Expected %f got %f", expected, res)
	}

	expected = 1.5
	res = roundToClosestHalve(1.3)
	if res != expected {
		t.Fatalf("Expected %f got %f", expected, res)
	}

	expected = 1.5
	res = roundToClosestHalve(1.6)
	if res != expected {
		t.Fatalf("Expected %f got %f", expected, res)
	}

	expected = 2.0
	res = roundToClosestHalve(1.8)
	if res != expected {
		t.Fatalf("Expected %f got %f", expected, res)
	}

	expected = 2.0
	res = roundToClosestHalve(2.0)
	if res != expected {
		t.Fatalf("Expected %f got %f", expected, res)
	}

	expected = 2.0
	res = roundToClosestHalve(2.23)
	if res != expected {
		t.Fatalf("Expected %f got %f", expected, res)
	}
}

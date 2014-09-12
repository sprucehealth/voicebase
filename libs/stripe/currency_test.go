package stripe

import (
	"encoding/json"
	"testing"
)

func TestCurrencyUnmarshalling(t *testing.T) {

	testString := `{ "currency" : "usd" }`

	type response struct {
		C Currency `json:"currency"`
	}
	var r response

	if err := json.Unmarshal([]byte(testString), &r); err != nil {
		t.Fatal(err)
	} else if r.C != USD {
		t.Fatalf("Expected %+v but got %+v", USD, r.C)
	}

	testString = `{ "currency" : "USD" }`
	r = response{}
	if err := json.Unmarshal([]byte(testString), &r); err != nil {
		t.Fatal(err)
	} else if r.C != USD {
		t.Fatalf("Expected %+v but got %+v", USD, r.C)
	}
}

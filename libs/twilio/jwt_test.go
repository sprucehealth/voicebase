package twilio

import (
	"reflect"
	"testing"
)

var testKey = []byte{3, 35, 53, 75, 43, 15, 165, 188, 131, 126, 6, 101, 119, 123, 166, 143, 90, 179, 40, 230, 240, 84, 201, 40, 169, 15, 132, 178, 210, 80, 46, 191, 211, 251, 90, 146, 210, 6, 71, 239, 150, 138, 180, 195, 119, 98, 61, 34, 61, 46, 33, 114, 5, 46, 79, 8, 192, 205, 154, 245, 103, 208, 128, 163}

func TestJWT(t *testing.T) {
	payload := map[string]interface{}{"str": "string", "int": 1234, "bool": true}
	token, err := jwtEncode(payload, testKey, hs256, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	if err := jwtDecode(token, testKey, &out); err != nil {
		t.Fatal(err)
	}
	exp := map[string]interface{}{"str": "string", "int": float64(1234), "bool": true}
	if !reflect.DeepEqual(out, exp) {
		t.Fatalf("Expected %v got %v", exp, out)
	}
}

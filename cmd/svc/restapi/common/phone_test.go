package common

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestPhoneNumber_MarshalUnmarshalJson(t *testing.T) {

	// marshal
	expectedResult := `{"phone":"7348465522"}`
	jsonData, err := json.Marshal(map[string]interface{}{
		"phone": Phone("7348465522"),
	})
	if err != nil {
		t.Fatal(err)
	} else if string(jsonData) != expectedResult {
		t.Fatalf("Expected %s but got %s", expectedResult, string(jsonData))
	}

	// unmarshal
	enteredPhone := "2068773590"
	expectedPhone := "206-877-3590"
	dataToUnmarshal := []byte(fmt.Sprintf(`{"phone" : %s}`, enteredPhone))
	var p struct {
		P Phone `json:"phone"`
	}
	if err := json.Unmarshal(dataToUnmarshal, &p); err != nil {
		t.Fatal(err)
	} else if p.P.String() != expectedPhone {
		t.Fatalf("Expected %s but got %s", expectedPhone, p.P.String())
	}

	enteredPhone = "206 877 3590"
	expectedPhone = "206-877-3590"
	dataToUnmarshal = []byte(fmt.Sprintf(`{"phone" : "%s"}`, enteredPhone))
	if err := json.Unmarshal(dataToUnmarshal, &p); err != nil {
		t.Fatal(err)
	} else if p.P.String() != expectedPhone {
		t.Fatalf("Expected %s but got %s", expectedPhone, p.P.String())
	}

	enteredPhone = "1206 877 3590"
	dataToUnmarshal = []byte(fmt.Sprintf(`{"phone" : "%s"}`, enteredPhone))
	if err := json.Unmarshal(dataToUnmarshal, &p); err != nil {
		t.Fatal(err)
	} else if p.P.String() != expectedPhone {
		t.Fatalf("Expected %s but got %s", expectedPhone, p.P.String())
	}

	enteredPhone = "1206.877.3590"
	dataToUnmarshal = []byte(fmt.Sprintf(`{"phone" : "%s"}`, enteredPhone))
	if err := json.Unmarshal(dataToUnmarshal, &p); err != nil {
		t.Fatal(err)
	} else if p.P.String() != expectedPhone {
		t.Fatalf("Expected %s but got %s", expectedPhone, p.P.String())
	}

	// test invalid unmarshalling
	enteredPhone = "1231231234"
	dataToUnmarshal = []byte(fmt.Sprintf(`{"phone" : %s}`, enteredPhone))
	var a struct {
		P Phone `json:"phone"`
	}
	if err := json.Unmarshal(dataToUnmarshal, &a); err == nil {
		t.Fatal("Expected number to be invalid but it wasn't")
	}
}

func testValidPhone(number string, t *testing.T) {
	if _, err := ParsePhone(number); err != nil {
		t.Fatalf("Expected %s phone number to be valid: %s", number, err)
	}
}

func TestValidPhoneNumber(t *testing.T) {
	testValidPhone("2068773590", t)
	testValidPhone("206-877-3590", t)
	testValidPhone("1206-877-3590", t)
	testValidPhone("206 877 3590", t)
	testValidPhone("1 206 877 3590", t)
	testValidPhone("206-877-3590", t)
	testValidPhone("206.877.3590", t)
	testValidPhone("12068773590", t)
	testValidPhone("1 206-877-3590", t)
	testValidPhone("1 206.877.3590", t)
	testValidPhone("1 206-877 3590", t)
	testValidPhone("1 206 877-3590", t)
	testValidPhone("1 206877 3590", t)
	testValidPhone("1 206877 3590", t)
	testValidPhone("(206)8773590", t)
	testValidPhone("(206) 877 3590", t)
	testValidPhone("(206) 877-3590", t)
}

func testInvalidPhone(number string, t *testing.T) {
	if _, err := ParsePhone(number); err == nil {
		t.Fatalf("Expected phone number %s to be invalid", number)
	}
}

func TestInvalidPhoneNumber(t *testing.T) {
	testInvalidPhone("-12068773590", t)
	testInvalidPhone("206x877x3590", t)
	testInvalidPhone("1206x877x3590", t)
	testInvalidPhone("((206) 877 3590)", t)
	testInvalidPhone("(206877 3590", t)
}

func TestValidPhoneNumberWithExtension(t *testing.T) {
	if _, err := ParsePhone("2068773590x123"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}

	if _, err := ParsePhone("206-877-3590x12345135351"); err != nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("2068773590X123"); err != nil {
		t.Fatalf("Expected phone number to be invalid: %+v", err)
	}
	if _, err := ParsePhone("206-877-3590X1243"); err != nil {
		t.Fatal("Expected phone number to be valid")
	}
}

func TestInvalidPhoneNumberShort(t *testing.T) {
	if _, err := ParsePhone("206877359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberAlpha(t *testing.T) {
	if _, err := ParsePhone("a206877359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("206877359a"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("2068a77359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("206-a12-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("206-123-3590xa24"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("a06-123-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberLength(t *testing.T) {

	if _, err := ParsePhone("206-1243-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("2064-123-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("206-123-35904"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberEmpty(t *testing.T) {
	if _, err := ParsePhone(""); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberExtensionInvalid(t *testing.T) {
	if _, err := ParsePhone("2068773590x"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberInvalidAreaCode(t *testing.T) {
	if _, err := ParsePhone("0008773590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumber_RepeatingDigits(t *testing.T) {
	if _, err := ParsePhone("1111111111"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("000-000-0000"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if _, err := ParsePhone("888-888-8888"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func BenchmarkSimplePhoneNumber(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParsePhone("2068773590"); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
}

func BenchmarkPhoneNumberWithExtension(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParsePhone("206-877-3590x123456"); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
}

func BenchmarkPhoneNumberInNormalizedFormat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParsePhone("206-877-3590"); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
}

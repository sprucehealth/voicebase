package encoding

import (
	"encoding/json"
	"testing"
)

type DOBExampleObject struct {
	DOB DOB `json:"dob"`
}

const (
	testDOBString = `{
		"dob" : "1987-11-08"
				}`

	testDOBStringWithEmptyValue = `{
		"dob" : ""
				}`

	testDOBStringWithNullValue = `{
		"dob" : null
		}`
)

func TestDOBMarshal(t *testing.T) {
	dobTest := DOB{Day: 11, Month: 12, Year: 2014}

	e1 := &DOBExampleObject{
		DOB: dobTest,
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal dob as expected: %+v", err)
	}

	expectedResult := `{"dob":"2014-12-11"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("DOB did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}
}

func TestDOBMarshalSingleMonthDay(t *testing.T) {
	dobTest := DOB{Day: 5, Month: 12, Year: 2014}

	e1 := &DOBExampleObject{
		DOB: dobTest,
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal dob as expected: %+v", err)
	}

	expectedResult := `{"dob":"2014-12-05"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("DOB did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}

	dobTest = DOB{Day: 5, Month: 5, Year: 2014}

	e1 = &DOBExampleObject{
		DOB: dobTest,
	}

	jsonData, err = json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal dob as expected: %+v", err)
	}

	expectedResult = `{"dob":"2014-05-05"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("DOB did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}
}

func TestDOBUnMarshal(t *testing.T) {
	testObject := &DOBExampleObject{}
	if err := json.Unmarshal([]byte(testDOBString), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.DOB.Month != 11 || testObject.DOB.Year != 1987 || testObject.DOB.Day != 8 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.DOB)
	}
}

func TestDOBUnMarshallNullValue(t *testing.T) {
	testObject := &DOBExampleObject{}
	if err := json.Unmarshal([]byte(testDOBStringWithNullValue), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.DOB.Month != 0 || testObject.DOB.Year != 0 || testObject.DOB.Day != 0 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.DOB)
	}
}

func TestDOBUnMarshallEmptyValue(t *testing.T) {
	testObject := &DOBExampleObject{}
	if err := json.Unmarshal([]byte(testDOBStringWithEmptyValue), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.DOB.Month != 0 || testObject.DOB.Year != 0 || testObject.DOB.Day != 0 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.DOB)
	}
}

func TestDOBFromString(t *testing.T) {
	dobString := "1987-11-08"
	dob, err := NewDOBFromString(dobString)
	if err != nil {
		t.Fatalf("unexpected error from dob parsing: %s", err)
	}
	if dob.Year != 1987 || dob.Month != 11 || dob.Day != 8 {
		t.Fatalf("Expected dob to be 1987-11-08 instead got %d-%02d-%02d", dob.Year, dob.Month, dob.Day)
	}
}

func TestDOBFromString_Error(t *testing.T) {
	dobString := "1987-aa-08"
	_, err := NewDOBFromString(dobString)
	if err == nil {
		t.Fatalf("expected error but got none")
	}
}

func TestParseDOB(t *testing.T) {
	testCases := []struct {
		Str        string
		Order      string
		Separators []rune
		DOB        DOB
	}{
		{"1921-03-09", "YMD", []rune{'-'}, DOB{Year: 1921, Month: 3, Day: 9}},
		{"01/08/1973", "MDY", []rune{'-', '/'}, DOB{Year: 1973, Month: 1, Day: 8}},
		{"05/12/87", "DMY", []rune{'/'}, DOB{Year: 1987, Month: 12, Day: 5}},
		{"6/1/03", "DMY", []rune{'-', '/'}, DOB{Year: 2003, Month: 1, Day: 6}},
	}
	for _, tc := range testCases {
		dob, err := ParseDOB(tc.Str, tc.Order, tc.Separators)
		if err != nil {
			t.Errorf("Failed to parse %s: %s", tc.Str, err.Error())
		} else if dob != tc.DOB {
			t.Errorf("Parsing %s resulted in %#v which does not match %#v", tc.Str, dob, tc.DOB)
		}
	}
}

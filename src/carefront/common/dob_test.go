package common

import (
	"encoding/json"
	"testing"
)

type DobExampleObject struct {
	Dob Dob `json:"dob"`
}

const (
	testDobString = `{
		"dob" : "1987/11/08"
				}`

	testDobStringWithEmptyValue = `{
		"dob" : ""
				}`

	testDobStringWithNullValue = `{
		"dob" : null
		}`
)

func TestDobMarshal(t *testing.T) {
	dobTest := Dob{Day: 11, Month: 12, Year: 2014}

	e1 := &DobExampleObject{
		Dob: dobTest,
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal dob as expected: %+v", err)
	}

	expectedResult := `{"dob":"2014/12/11"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("Dob did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}
}

func TestDobMarshalSingleMonthDay(t *testing.T) {
	dobTest := Dob{Day: 5, Month: 12, Year: 2014}

	e1 := &DobExampleObject{
		Dob: dobTest,
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal dob as expected: %+v", err)
	}

	expectedResult := `{"dob":"2014/12/05"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("Dob did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}

	dobTest = Dob{Day: 5, Month: 5, Year: 2014}

	e1 = &DobExampleObject{
		Dob: dobTest,
	}

	jsonData, err = json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal dob as expected: %+v", err)
	}

	expectedResult = `{"dob":"2014/05/05"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("Dob did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}
}

func TestDobUnMarshal(t *testing.T) {
	testObject := &DobExampleObject{}
	if err := json.Unmarshal([]byte(testDobString), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.Dob.Month != 11 || testObject.Dob.Year != 1987 || testObject.Dob.Day != 8 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.Dob)
	}
}

func TestDobUnMarshallNullValue(t *testing.T) {
	testObject := &DobExampleObject{}
	if err := json.Unmarshal([]byte(testDobStringWithNullValue), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.Dob.Month != 0 || testObject.Dob.Year != 0 || testObject.Dob.Day != 0 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.Dob)
	}
}

func TestDobUnMarshallEmptyValue(t *testing.T) {
	testObject := &DobExampleObject{}
	if err := json.Unmarshal([]byte(testDobStringWithEmptyValue), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.Dob.Month != 0 || testObject.Dob.Year != 0 || testObject.Dob.Day != 0 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.Dob)
	}
}

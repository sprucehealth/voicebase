package encoding

import (
	"encoding/json"
	"testing"
)

type DateExampleObject struct {
	Date Date `json:"date"`
}

const (
	testDateString = `{
		"date" : "1987-11-08"
				}`

	testDateStringWithEmptyValue = `{
		"date" : ""
				}`

	testDateStringWithNullValue = `{
		"date" : null
		}`
)

func TestDateMarshal(t *testing.T) {
	dateTest := Date{Day: 11, Month: 12, Year: 2014}

	e1 := &DateExampleObject{
		Date: dateTest,
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal date as expected: %+v", err)
	}

	expectedResult := `{"date":"2014-12-11"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("Date did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}
}

func TestDateMarshalSingleMonthDay(t *testing.T) {
	dateTest := Date{Day: 5, Month: 12, Year: 2014}

	e1 := &DateExampleObject{
		Date: dateTest,
	}

	jsonData, err := json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal date as expected: %+v", err)
	}

	expectedResult := `{"date":"2014-12-05"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("Date did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}

	dateTest = Date{Day: 5, Month: 5, Year: 2014}

	e1 = &DateExampleObject{
		Date: dateTest,
	}

	jsonData, err = json.Marshal(e1)
	if err != nil {
		t.Fatalf("Unable to marshal date as expected: %+v", err)
	}

	expectedResult = `{"date":"2014-05-05"}`
	if string(jsonData) != expectedResult {
		t.Fatalf("Date did not get marshalled as expected. Got %s when expected %s", string(jsonData), expectedResult)
	}
}

func TestDateUnMarshal(t *testing.T) {
	testObject := &DateExampleObject{}
	if err := json.Unmarshal([]byte(testDateString), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.Date.Month != 11 || testObject.Date.Year != 1987 || testObject.Date.Day != 8 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.Date)
	}
}

func TestDateUnMarshallNullValue(t *testing.T) {
	testObject := &DateExampleObject{}
	if err := json.Unmarshal([]byte(testDateStringWithNullValue), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.Date.Month != 0 || testObject.Date.Year != 0 || testObject.Date.Day != 0 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.Date)
	}
}

func TestDateUnMarshallEmptyValue(t *testing.T) {
	testObject := &DateExampleObject{}
	if err := json.Unmarshal([]byte(testDateStringWithEmptyValue), testObject); err != nil {
		t.Fatalf("Unable to unmarshal object as expected: %+v", err)
	}

	if testObject.Date.Month != 0 || testObject.Date.Year != 0 || testObject.Date.Day != 0 {
		t.Fatalf("testObject not unmarshalled into values as expected: %+v", testObject.Date)
	}
}

func TestParseDate(t *testing.T) {
	testCases := []struct {
		Str        string
		Order      string
		Separators []rune
		Date       Date
	}{
		{"1921-03-09", "YMD", []rune{'-'}, Date{Year: 1921, Month: 3, Day: 9}},
		{"01/08/1973", "MDY", []rune{'-', '/'}, Date{Year: 1973, Month: 1, Day: 8}},
		{"05/12/87", "DMY", []rune{'/'}, Date{Year: 1987, Month: 12, Day: 5}},
		{"6/1/03", "DMY", []rune{'-', '/'}, Date{Year: 2003, Month: 1, Day: 6}},
	}
	for _, tc := range testCases {
		date, err := ParseDate(tc.Str, tc.Order, tc.Separators, 0)
		if err != nil {
			t.Errorf("Failed to parse %s: %s", tc.Str, err.Error())
		} else if date != tc.Date {
			t.Errorf("Parsing %s resulted in %#v which does not match %#v", tc.Str, date, tc.Date)
		}
	}
}

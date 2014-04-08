package encoding

import (
	"encoding/json"
	"testing"
)

type exampleHighPrecisionFloat64Container struct {
	HPFloatValue HighPrecisionFloat64
}

func TestJSONMarshalHighPrecisionFloat64(t *testing.T) {
	marshalHighPrecisionFloat64AndCheckResult(12354.1567889, "{\"HPFloatValue\":\"12354.1567889\"}", t)
	marshalHighPrecisionFloat64AndCheckResult(12354.00000, "{\"HPFloatValue\":\"12354\"}", t)
	marshalHighPrecisionFloat64AndCheckResult(12354.0, "{\"HPFloatValue\":\"12354\"}", t)
	marshalHighPrecisionFloat64AndCheckResult(123456, "{\"HPFloatValue\":\"123456\"}", t)
	marshalHighPrecisionFloat64AndCheckResult(12354.00001, "{\"HPFloatValue\":\"12354.00001\"}", t)
	marshalHighPrecisionFloat64AndCheckResult(0.123456789, "{\"HPFloatValue\":\"0.123456789\"}", t)
	marshalHighPrecisionFloat64AndCheckResult(123456.12312115, "{\"HPFloatValue\":\"123456.12312115\"}", t)
}

func TestJSONUnmarshalHighPrecisionFloat64(t *testing.T) {

	unmarshalHighPrecisionFloat64AndCheckResult("{\"HPFloatValue\":\"12354.1567889\"}", 12354.1567889, t)
	unmarshalHighPrecisionFloat64AndCheckResult("{\"HPFloatValue\":\"12354\"}", 12354, t)
	unmarshalHighPrecisionFloat64AndCheckResult("{\"HPFloatValue\":\"12354.0\"}", 12354, t)
	unmarshalHighPrecisionFloat64AndCheckResult("{\"HPFloatValue\":\"0.12354\"}", 0.12354, t)
	unmarshalHighPrecisionFloat64AndCheckResult("{\"HPFloatValue\":\"012354\"}", 12354, t)
	unmarshalHighPrecisionFloat64AndCheckResult("{\"HPFloatValue\":\"11251515161.9\"}", 11251515161.9, t)
}

func unmarshalHighPrecisionFloat64AndCheckResult(jsonData string, floatValue float64, t *testing.T) {
	e1 := exampleHighPrecisionFloat64Container{}
	if err := json.Unmarshal([]byte(jsonData), &e1); err != nil {
		t.Fatalf("Unable to unmarshal json into its object: %+v", err)
	}

	if e1.HPFloatValue != HighPrecisionFloat64(floatValue) {
		t.Fatalf("Expected %g instead got %g", floatValue, e1.HPFloatValue)
	}
}

func marshalHighPrecisionFloat64AndCheckResult(floatValue float64, expectedResult string, t *testing.T) {
	e1 := exampleHighPrecisionFloat64Container{
		HPFloatValue: HighPrecisionFloat64(floatValue),
	}

	jsonData, err := json.Marshal(&e1)
	if err != nil {
		t.Fatalf("Unable to marshal high precision float64: %+v", err)
	}

	output := string(jsonData)
	if output != expectedResult {
		t.Fatalf("Expected marshalling of object to result in %s but instead resulted in %s", expectedResult, output)
	}
}

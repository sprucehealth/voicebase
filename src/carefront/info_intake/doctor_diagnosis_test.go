package info_intake

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestDiagnosisParsing(t *testing.T) {

	diagnosis := &DiagnosisIntake{}
	data, err := ioutil.ReadFile("../api-response-examples/v1/doctor/diagnosis.json")
	if err != nil {
		t.Fatal("unable to parse file " + err.Error())
	}

	err = json.Unmarshal(data, diagnosis)
	if err != nil {
		t.Fatal("Unable to parse diagnosis object " + err.Error())
	}
}

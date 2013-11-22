package main

import (
	"carefront/info_intake"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../info_intake/patient_visit_layout.json")
	patientVisitOverview := &info_intake.PatientVisitOverview{}
	err := json.Unmarshal(fileContents, &patientVisitOverview)

	if err != nil {
		panic(err)
	}

	marshalledBytes, err := json.MarshalIndent(patientVisitOverview, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(marshalledBytes))
}

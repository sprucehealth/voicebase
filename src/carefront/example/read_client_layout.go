package main

import (
	"carefront/info_intake"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../info_intake/condition_intake.json")
	treatmentRes := &info_intake.HealthCondition{}
	err := json.Unmarshal(fileContents, &treatmentRes)
	if err != nil {
		panic(err)
	}

	marshalledBytes, err := json.MarshalIndent(treatmentRes, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(marshalledBytes))
}

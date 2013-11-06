package main

import (
	"carefront/layout_transformer"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func main() {
	fileContents, _ := ioutil.ReadFile("../layout_transformer/client_intake.json")
	treatmentRes := &layout_transformer.Treatment{}
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

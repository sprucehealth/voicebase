package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type PhotoContainer struct {
	Title  string   `json:"titles"`
	Types  []string `json:"types"`
	Photos []photo  `json:"photos"`
}

type photo struct {
	Types          []string `json:"types"`
	Title          string   `json:"title"`
	PhotoUrl       string   `json:"photo_url"`
	PlaceholderUrl string   `json:"placeholder_url"`
}

type AlertLabelList struct {
	Types []string
}

func main() {
	file, err := os.Open("../carefront/api-response-examples/v1/doctor/visit/review_v2.json")
	if err != nil {
		panic(err)
	}

	var jsonContents map[string]interface{}
	if err := json.NewDecoder(file).Decode(&jsonContents); err != nil {
		panic(err)
	}

	if jsonContents["views"] != nil {
		// we know it contains and array of things
		var things []interface{}
		things = jsonContents["views"].([]interface{})

		// lets inspect the first item
		firstItem := things[0].(map[string]interface{})
		appropriateInstance := getAppropriateInstance(firstItem["types"].([]interface{}))
		iface := reflect.New(appropriateInstance).Interface()
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: "json",
			Result:  iface,
		})
		if err != nil {
			panic(err)
		}

		if err := decoder.Decode(firstItem); err != nil {
			panic(err)
		}
		fmt.Printf("%#v", iface)
	}

	// fmt.Printf("%#v", jsonContents)
	// var doctorLayout map[string]interface{}
	// for key, data := range jsonContents {
	// 	if key == "types" {

	// 	}
	// }

}

func getAppropriateInstance(types []interface{}) reflect.Type {
	for _, typeDef := range types {
		switch typeDef.(string) {
		case "photo_section":
			return reflect.TypeOf(PhotoContainer{})
		}
	}
	return nil
}

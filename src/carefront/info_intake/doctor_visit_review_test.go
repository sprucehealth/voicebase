package info_intake

import (
	"carefront/common"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/SpruceHealth/mapstructure"
)

func TestParsingTemplateForDoctorVisitReview(t *testing.T) {

	fileContents, err := ioutil.ReadFile("../carefront/api-response-examples/v1/doctor/visit/review_v2_template.json")
	if err != nil {
		t.Fatalf("error parsing file: %s", err)
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(fileContents, &jsonData)
	if err != nil {
		t.Fatalf("error unmarshalling file contents into json: %s", err)
	}

	sectionList := &common.SectionListView{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:  sectionList,
		TagName: "json",
	}
	if err := decoderConfig.SetRegistry(dVisitReviewViewTypeRegistry.Map()); err != nil {
		t.Fatalf("Error setting registry for decoder config: %s", err)
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		t.Fatalf("error creating new decoder: %s", err)
	}

	err = d.Decode(jsonData)
	if err != nil {
		t.Fatalf("error decoding template into native go structures: %s", err)
	}
}

func TestParsingLayoutForDoctorVisitReview(t *testing.T) {

	fileContents, err := ioutil.ReadFile("../carefront/api-response-examples/v1/doctor/visit/review_v2.json")
	if err != nil {
		t.Fatalf("error parsing file: %s", err)
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(fileContents, &jsonData)
	if err != nil {
		t.Fatalf("error unmarshalling file contents into json: %s", err)
	}

	sectionList := &common.SectionListView{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:  sectionList,
		TagName: "json",
	}
	if err := decoderConfig.SetRegistry(dVisitReviewViewTypeRegistry.Map()); err != nil {
		t.Fatalf("Error setting registry for decoder config: %s", err)
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		t.Fatalf("error creating new decoder: %s", err)
	}

	err = d.Decode(jsonData)
	if err != nil {
		t.Fatalf("error decoding template into native go structures: %s", err)
	}
}

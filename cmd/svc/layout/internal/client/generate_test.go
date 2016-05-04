package client

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/mapstructure"
)

func TestTransform(t *testing.T) {
	testTransform(t, "testdata/athletes_foot.saml")
}

func testTransform(t *testing.T, fileName string) {

	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf(err.Error())
	}

	intake, err := saml.Parse(file)
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = GenerateIntakeLayout(intake)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// indentedJSON, err := json.MarshalIndent(intakeLayout, "", " ")
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }

	// fmt.Println(string(indentedJSON))
}

func TestReviewGeneration(t *testing.T) {
	file, err := os.Open("testdata/athletes_foot.saml")
	if err != nil {
		t.Fatalf(err.Error())
	}

	intake, err := saml.Parse(file)
	if err != nil {
		t.Fatal(err.Error())
	}

	visitReviewLayout, err := GenerateReviewLayout(intake)
	if err != nil {
		t.Fatalf(err.Error())
	}

	jsonData, err := json.Marshal(visitReviewLayout)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// ensure that we can generate the structure again from the marshalled form
	var dermReviewLayout visitreview.SectionListView

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:   &dermReviewLayout,
		TagName:  "json",
		Registry: *visitreview.TypeRegistry,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	var unmarshalledJSONData map[string]interface{}
	if err := json.Unmarshal(jsonData, &unmarshalledJSONData); err != nil {
		t.Fatalf(err.Error())
	}

	if err := d.Decode(unmarshalledJSONData); err != nil {
		t.Fatalf(err.Error())
	}

	_, err = json.Marshal(dermReviewLayout)
	if err != nil {
		t.Fatal(err.Error())
	}

	// fmt.Println(string(jsonData))
	// fmt.Println()

	// formatDermReview(t)
}

// func formatDermReview(t *testing.T) {

// 	data, err := ioutil.ReadFile("derm.json")
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	var dermReviewLayout visitreview.SectionListView

// 	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
// 		Result:   &dermReviewLayout,
// 		TagName:  "json",
// 		Registry: *visitreview.TypeRegistry,
// 	})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	var unmarshalledJSONData map[string]interface{}
// 	if err := json.Unmarshal(data, &unmarshalledJSONData); err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	if err := d.Decode(unmarshalledJSONData); err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	// remarshal and print
// 	jsonData, err := json.Marshal(dermReviewLayout)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}

// 	fmt.Println(string(jsonData))
// }

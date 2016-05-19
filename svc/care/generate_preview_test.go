package care

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/mapstructure"
)

func TestBuildPreviewContext(t *testing.T) {

	intakeData, err := ioutil.ReadFile("testdata/intake.json")
	if err != nil {
		t.Fatal(err)
	}

	var intake layout.Intake
	if err := json.Unmarshal(intakeData, &intake); err != nil {
		t.Fatal(err)
	}

	reviewData, err := ioutil.ReadFile("testdata/review.json")
	if err != nil {
		t.Fatal(err)
	}

	var reviewLayout visitreview.SectionListView

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:   &reviewLayout,
		TagName:  "json",
		Registry: *visitreview.TypeRegistry,
	})
	if err != nil {
		t.Fatal(err)
	}

	var unmarshalledJSONData map[string]interface{}
	if err := json.Unmarshal(reviewData, &unmarshalledJSONData); err != nil {
		t.Fatal(err)
	}

	if err := d.Decode(unmarshalledJSONData); err != nil {
		t.Fatal(err)
	}

	_, err = GenerateVisitLayoutPreview(&intake, &reviewLayout)
	if err != nil {
		t.Fatal(err)
	}
}

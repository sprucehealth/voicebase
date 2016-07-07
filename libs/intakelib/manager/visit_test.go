package manager

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestVisitParse(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/eczema.json")
	if err != nil {
		t.Fatal(err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatal(err)
	}

	v := &visit{}
	if err := v.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	if len(v.Sections) != 3 {
		t.Fatal("Expected 3 sections to be fully populated")
	}

	test.Equals(t, 4, len(v.Transitions))
	test.Equals(t, true, v.OverviewHeader != nil)
}

func TestVisit_transitionsParse(t *testing.T) {
	var transitionJSON = `
	{
	  "message": "We'll start by asking you questions about your symptoms.",
	  "buttons": [
	    {
	      "button_text": "Begin",
	      "tap_url": "spruce:///action/view_next_visit_section",
	      "style": "filled"
	    }
	  ]
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(transitionJSON), &data); err != nil {
		t.Fatal(err)
	}

	var tItem transitionItem
	if err := tItem.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "We'll start by asking you questions about your symptoms.", tItem.Message)
	test.Equals(t, "Begin", tItem.Buttons[0].Text)
	test.Equals(t, "filled", tItem.Buttons[0].Style)
	test.Equals(t, "spruce:///action/view_next_visit_section", tItem.Buttons[0].TapLink)
}

func TestVisit_overviewParse(t *testing.T) {
	var overviewJSON = `
	{
    	"title": "Eczema Visit",
    	"subtitle": "With First Available Doctor",
    	"icon_url": "spruce:///icon"
  	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(overviewJSON), &data); err != nil {
		t.Fatal(err)
	}

	var overview visitOverviewHeader
	if err := overview.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "Eczema Visit", overview.Title)
	test.Equals(t, "With First Available Doctor", overview.Subtitle)
	test.Equals(t, "spruce:///icon", overview.IconURL)
}

func TestVisit_Questions(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/eczema.json")
	if err != nil {
		t.Fatal(err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatal(err)
	}

	v := &visit{}
	if err := v.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	questions := v.questions()
	if len(questions) == 0 {
		t.Fatalf("Expected questions to exist in the visit but none were returned")
	}
}

func TestVisit_visibility(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/eczema.json")
	if err != nil {
		t.Fatal(err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatal(err)
	}

	v := &visit{}
	if err := v.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// by default all screens, sections and questions shoudl be visible
	for _, section := range v.Sections {
		test.Equals(t, visible, section.visibility())
		for _, screen := range section.Screens {
			test.Equals(t, visible, screen.visibility())
			switch s := screen.(type) {
			case *questionScreen:
				for _, qItem := range s.Questions {
					test.Equals(t, visible, qItem.visibility())
				}
			case *mediaScreen:
				for _, qItem := range s.MediaQuestions {
					test.Equals(t, visible, qItem.visibility())
				}
			}
		}
	}
}

//TestVisit_screenTitle ensures that the screen title is set for
// photo, pharmacy and question screens and matches that of its section
func TestVisit_screenTitle(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/eczema.json")
	if err != nil {
		t.Fatal(err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatal(err)
	}

	v := &visit{}
	if err := v.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// iterate through the sections and its screens
	for _, section := range v.Sections {
		for _, sc := range section.Screens {
			test.Equals(t, true, section.Title != "")
			switch s := sc.(type) {
			case *mediaScreen:
				test.Equals(t, section.Title, s.Title)
			case *pharmacyScreen:
				test.Equals(t, section.Title, s.Title)
			case *questionScreen:
				test.Equals(t, section.Title, s.Title)
			}
		}
	}
}

func BenchmarkJSONParsing(b *testing.B) {
	jsonData, err := ioutil.ReadFile("testdata/eczema.json")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var data map[string]interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			b.Fatal(err)
		}

	}
}

func BenchmarkVisitParsing(b *testing.B) {
	jsonData, err := ioutil.ReadFile("testdata/eczema.json")
	if err != nil {
		b.Fatal(err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := &visit{}
		if err := v.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
			b.Fatal(err)
		}
	}
}

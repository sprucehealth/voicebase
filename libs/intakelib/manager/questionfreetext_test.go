package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const freeTextJSON = `
{
              "question": "q_derm_eczema_locations_that_make_rash_worse",
              "id": "40477",
              "question_title": "What locations make it worse?",
              "question_title_has_tokens": false,
              "question_type": "q_type_free_text",
              "type": "q_type_free_text",
              "question_summary": "Locations that make rash worse",
              "additional_fields": {
                "placeholder_text": "Describe what locations make your rash worse…"
              },
              "to_prefill": false,
              "prefilled_with_previous_answers": false,
              "required": true,
              "to_alert": false,
              "alert_text": ""
}`

func TestFreeText_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(freeTextJSON), &data); err != nil {
		t.Fatal(err)
	}

	ftq := &freeTextQuestion{}
	if err := ftq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "What locations make it worse?", ftq.questionInfo.Title)
	test.Equals(t, "40477", ftq.questionInfo.ID)
	test.Equals(t, "Describe what locations make your rash worse…", ftq.PlaceholderText)
}

func TestFreeText_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(freeTextJSON), &data); err != nil {
		t.Fatal(err)
	}

	ftq := &freeTextQuestion{}
	if err := ftq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// nullify answers given its not static info
	ftq.answer = nil

	ftq1 := ftq.staticInfoCopy(nil).(*freeTextQuestion)
	test.Equals(t, ftq, ftq1)
}

func TestFreeText_Answer(t *testing.T) {

	freeTextWithoutAdditionalFieldsJSON := `
	{
	"question": "q_derm_eczema_locations_that_make_rash_worse",
	"id": "40477",
	"question_title": "What locations make it worse?",
	"question_title_has_tokens": false,
	"question_type": "q_type_free_text",
	"type": "q_type_free_text",
	"question_summary": "Locations that make rash worse",
	"additional_fields": {
	"placeholder_text": "Describe what locations make your rash worse…"
	},
	"to_prefill": false,
	"prefilled_with_previous_answers": false,
	"required": true,
	"to_alert": false,
	"alert_text": "",
	"answers": [{
		"answer_id": "64406",
		"type" : "q_type_free_text",
		"question_id": "40477",
		"answer_text": "Testing free text."
	}]
}`
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(freeTextWithoutAdditionalFieldsJSON), &data); err != nil {
		t.Fatal(err)
	}

	ftq := &freeTextQuestion{}
	if err := ftq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	if err := ftq.setPatientAnswer(&freeTextAnswer{
		Text: "Hello",
	}); err != nil {
		t.Fatal(err)
	}

	// set empty answer
	if err := ftq.setPatientAnswer(&freeTextAnswer{}); err != nil {
		t.Fatal(err)
	}

	// set invalid answer
	if err := ftq.setPatientAnswer(&autocompleteAnswer{}); err == nil {
		t.Fatal("Expected invalid answer but got valid answer")
	}

}

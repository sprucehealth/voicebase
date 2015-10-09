package feedback

import (
	"encoding/json"
	"testing"
)

func TestFreeTextResponseValidation(t *testing.T) {
	f := &FreeTextTemplate{
		Title:           "hello",
		PlaceholderText: "placeholder_text",
		ButtonTitle:     "bt",
	}

	r := &FreeTextResponse{
		Response: "response",
	}

	jsonData, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	r1, err := f.ParseAndValidateResponse(10, jsonData)
	if err != nil {
		t.Fatal(err)
	}

	if r1.TemplateID() != 10 {
		t.Fatal("ID didn't match as expected.")
	}
}

func TestMultipleChoiceResponseValidation(t *testing.T) {
	f := &MultipleChoiceTemplate{
		Title:       "hello",
		Subtitle:    "subtitle",
		ButtonTitle: "button_title",
		PotentialAnswers: []PotentialAnswer{
			{
				ID:   "1",
				Text: "Yes",
			},
			{
				ID:   "2",
				Text: "No",
			},
		},
	}

	r := &MultipleChoiceResponse{
		AnswerSelections: []MultipleChoiceSelection{
			{
				PotentialAnswerID: "1",
			},
			{
				PotentialAnswerID: "2",
			},
		},
	}

	jsonData, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	r1, err := f.ParseAndValidateResponse(10, jsonData)
	if err != nil {
		t.Fatal(err)
	}

	if r1.TemplateID() != 10 {
		t.Fatal("ID didn't match as expected")
	}
}

func TestMultipleChoiceResponseValidation_Invalid(t *testing.T) {
	f := &MultipleChoiceTemplate{
		Title:       "hello",
		Subtitle:    "subtitle",
		ButtonTitle: "button_title",
		PotentialAnswers: []PotentialAnswer{
			{
				ID:   "1",
				Text: "Yes",
			},
			{
				ID:   "2",
				Text: "No",
			},
		},
	}

	r := &MultipleChoiceResponse{
		AnswerSelections: []MultipleChoiceSelection{
			{
				PotentialAnswerID: "1",
			},
			{
				PotentialAnswerID: "23",
			},
		},
	}

	jsonData, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.ParseAndValidateResponse(10, jsonData)
	// should fail because one of the answers does not exist
	if err == nil {
		t.Fatal("Expected validation error but got none")
	}
}

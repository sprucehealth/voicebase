package feedback

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/common"
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
		PotentialAnswers: []*PotentialAnswer{
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
		PotentialAnswers: []*PotentialAnswer{
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

func TestOpenURLTemplate_ClientView(t *testing.T) {
	out := &OpenURLTemplate{
		Title:       "test",
		ButtonTitle: "button",
		AndroidConfig: OpenURLTemplatePlatformConfig{
			IconURL:  "android_icon",
			OpenURL:  "android_open",
			BodyText: "android_body",
		},
		IOSConfig: OpenURLTemplatePlatformConfig{
			IconURL:  "ios_icon",
			OpenURL:  "ios_open",
			BodyText: "ios_body",
		},
	}

	if err := out.Validate(); err != nil {
		t.Fatal(err)
	}

	cv := out.ClientView(5, common.Android)
	if iconURL := cv.(*openURLClientView).Body.IconURL; iconURL != "android_icon" {
		t.Fatalf("Expected urls for android but got %s instead", iconURL)
	}
	if openURL := cv.(*openURLClientView).URL; openURL != "android_open" {
		t.Fatalf("Expected urls for android but got %s instead", openURL)
	}
	if bodyText := cv.(*openURLClientView).Body.Text; bodyText != "android_body" {
		t.Fatalf("Expected urls for android but got %s instead", bodyText)
	}

	cv = out.ClientView(5, common.IOS)

	if iconURL := cv.(*openURLClientView).Body.IconURL; iconURL != "ios_icon" {
		t.Fatalf("Expected urls for android but got %s instead", iconURL)
	}
	if openURL := cv.(*openURLClientView).URL; openURL != "ios_open" {
		t.Fatalf("Expected urls for android but got %s instead", openURL)
	}
	if bodyText := cv.(*openURLClientView).Body.Text; bodyText != "ios_body" {
		t.Fatalf("Expected urls for android but got %s instead", bodyText)
	}
}

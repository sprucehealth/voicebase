package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const photoQuestionJSON = `{
              "question": "q_derm_eczema_fingernails",
              "id": "40499",
              "question_title": "Fingernails",
              "question_title_has_tokens": true,
              "question_type": "q_type_photo_section",
              "type": "q_type_photo_section",
              "question_summary": "Fingernails",
              "condition": {
                "op": "answer_contains_any",
                "type": "answer_contains_any",
                "question": "q_derm_eczema_number_of_affected_fingernails",
                "question_id": "40461",
                "potential_answers_id": [
                  "126034",
                  "126035",
                  "126036"
                ],
                "potential_answers": [
                  "a_derm_eczema_number_of_affected_fingernails_1",
                  "a_derm_eczema_number_of_affected_fingernails_2_to_9",
                  "a_derm_eczema_number_of_affected_fingernails_all_of_them"
                ]
              },
              "to_prefill": false,
              "prefilled_with_previous_answers": false,
              "required": true,
              "to_alert": false,
              "alert_text": "",
              "media_slots": [
                {
                  "id": "8705",
                  "name": "Fingernails",
                  "type": "",
                  "required": true,
                  "client_data": {
                    "flash": "auto",
                    "initial_camera_direction": "back",
                    "media_missing_error_message": "Add a photo of your fingernails to continue.",
                    "tip": "Make sure your doctor can clearly see your natural nails.",
                    "tips" : {
                    	"inline" : {
                    		"tip" : "inline tip",
                    		"tip_subtext" : "inline tip subtext"
                    	}
                    }
                  }
                }
              ]
            }`

func TestPhotoQuestion_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(photoQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	pq := &mediaQuestion{}
	if err := pq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	if pq.questionInfo.Title != "Fingernails" {
		t.Fatal("photo question title not parsed correctly")
	} else if pq.questionInfo.ID != "40499" {
		t.Fatal("photo question id not parsed correctly")
	} else if !pq.questionInfo.TitleHasTokens {
		t.Fatal("title_has_tokens false")
	} else if pq.questionInfo.Cond == nil {
		t.Fatal("condition not parsed correctly")
	} else if len(pq.Slots) != 1 {
		t.Fatal("number of slots not parsed correctly")
	} else if pq.Slots[0].ID != "8705" {
		t.Fatal("photo slot id not parsed correctly")
	} else if pq.Slots[0].Name != "Fingernails" {
		t.Fatal("photo slot name not parsed correctly")
	} else if !pq.Slots[0].Required {
		t.Fatal("photo slot required flag not parsed correctly")
	} else if pq.Slots[0].FlashState != "auto" {
		t.Fatal("photo slot flash state not parsed correctly")
	} else if pq.Slots[0].InitialCameraDirection != "back" {
		t.Fatal("photo slot camera direction not parsed")
	} else if pq.Slots[0].MediaMissingErrorMessage != "Add a photo of your fingernails to continue." {
		t.Fatal("photo missing error message not parsed correctly")
	} else if pq.Slots[0].Tip != "Make sure your doctor can clearly see your natural nails." {
		t.Fatal("photo slot tip not parsed correctly")
	} else if len(pq.Slots[0].Tips) != 1 {
		t.Fatalf("Expected inline tips but got none")
	} else if pq.Slots[0].Tips["inline"].Tip != "inline tip" {
		t.Fatalf("Expected inline tip but got none")
	} else if pq.Slots[0].Tips["inline"].TipSubtext != "inline tip subtext" {
		t.Fatalf("Expected inline tip but got none")
	}
}

func TestPhotoQuestion_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(photoQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	pq := &mediaQuestion{}
	if err := pq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// nullify answers since that is not part of map
	pq.answer = nil

	pq2 := pq.staticInfoCopy(nil).(*mediaQuestion)

	// compare photo slots
	// test.Equals(t, pq.Slots[0].Tips, pq2.Slots[0].Tips)
	test.Equals(t, pq, pq2)
}

func TestPhotoQuestion_Answer(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(photoQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	pq := &mediaQuestion{}
	if err := pq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// set single complete photo section
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// set multiple photo sections
	pq.AllowsMultipleSections = true
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// ensure can set 0 photo sections
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{},
	}); err != nil {
		t.Fatal(err)
	}

	// ensure cannot set multiple sections for photo question if not allowed
	pq.AllowsMultipleSections = false
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// ensure cannot set photo section without name
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// ensure cannot set photo section with one of the slots not having a name
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Media: []*mediaSlotAnswerItem{
					{
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// ensure cannot set a photo section with no photos
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

}

func TestPhotoQuestion_canPersistAnswer(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(photoQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	pq := &mediaQuestion{}
	if err := pq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// set multiple photo sections
	pq.AllowsMultipleSections = true
	if err := pq.setPatientAnswer(&mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
			{
				Name: "sectionName",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "slot",
						ServerMediaID: "213a",
						SlotID:        "10",
					},
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// answer should be considered complete
	test.Equals(t, true, pq.canPersistAnswer())

	// lets change one of the photos to be in the process of being uploaded
	pq.answer.Sections[0].Media[0].ServerMediaID = ""
	pq.answer.Sections[1].Media[0].LocalMediaID = "adgkhag"

	// answer should no longer be considered complete
	test.Equals(t, false, pq.canPersistAnswer())
}

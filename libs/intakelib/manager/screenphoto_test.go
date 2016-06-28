package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const screenPhotoJSON = `
{
				"header_title": "Take photos of the areas where you are currently experiencing a rash.",
				"header_title_has_tokens": false,
				"header_subtitle": "Photo quality matters. Your doctor will use these photos to make a diagnosis.",
				"header_summary": "Areas Affected by Rash",
				"condition": {
						"op": "answer_contains_any",
						"type": "answer_contains_any",
						"question": "q_derm_rash_affected_areas",
						"question_id": "40551",
						"potential_answers_id": ["126307"],
						"potential_answers": ["a_derm_rash_affected_areas_face"]
					},
				"questions": [{
					"question": "q_derm_rash_face",
					"question_id": "40585",
					"question_title": "Face",
					"question_title_has_tokens": false,
					"question_type": "q_type_photo_section",
					"type": "q_type_photo_section",
					"question_summary": "Face",
					"to_prefill": false,
					"prefilled_with_previous_answers": false,
					"required": true,
					"to_alert": false,
					"alert_text": "",
					"photo_slots": [{
						"id": "8707",
						"name": "Face Front",
						"type": "",
						"required": true,
						"client_data": {
							"initial_camera_direction": "front",
							"overlay_image_url": "spruce:///image/photo_face_outline",
							"photo_missing_error_message": "A photo of the front of your face is required to continue.",
							"tip": "Center your face in the dotted lines."
						}
					}, {
						"id": "8708",
						"name": "Side",
						"type": "",
						"required": true,
						"client_data": {
							"initial_camera_direction": "front",
							"overlay_image_url": "spruce:///image/photo_face_outline",
							"photo_missing_error_message": "A photo of the side of your face is required to continue.",
							"tip": "Turn your face to the side.",
							"tip_style": "point_left",
							"tip_subtext": "Just move your face, not your phone."
						}
					}, {
						"id": "8709",
						"name": "Other Side",
						"type": "",
						"required": true,
						"client_data": {
							"initial_camera_direction": "front",
							"overlay_image_url": "spruce:///image/photo_face_outline",
							"photo_missing_error_message": "A second photo of the side of your face is required to continue.",
							"tip": "Now turn to the other side.",
							"tip_style": "point_right",
							"tip_subtext": "Just move your face, not your phone."
						}
					}, {
						"id": "8710",
						"name": "Face",
						"type": "",
						"required": false,
						"client_data": {}
					}]
				}],
				"screen_type": "screen_type_photo",
				"type": "screen_type_photo"
}`

func TestScreenPhoto_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenPhotoJSON), &data); err != nil {
		t.Fatal(err)
	}

	ps := &photoScreen{}
	if err := ps.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "Take photos of the areas where you are currently experiencing a rash.", ps.ContentHeaderTitle)
	test.Equals(t, 1, len(ps.PhotoQuestions))
}

func TestScreenPhoto_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenPhotoJSON), &data); err != nil {
		t.Fatal(err)
	}

	ps := &photoScreen{}
	if err := ps.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	ps2 := ps.staticInfoCopy(nil).(*photoScreen)
	test.Equals(t, len(ps.PhotoQuestions), len(ps2.PhotoQuestions))
	for i, pqItem := range ps.PhotoQuestions {
		test.Equals(t, true, pqItem != ps2.PhotoQuestions[i])
		test.Equals(t, pqItem, ps2.PhotoQuestions[i])
	}

	// lets also make sure that title parsing works
	ps.ContentHeaderTitle = "Hi <parent_answer_text>"
	ps.ContentHeaderTitleHasTokens = true

	ps3 := ps.staticInfoCopy(map[string]string{"answer": "spruce"}).(*photoScreen)
	test.Equals(t, "Hi spruce", ps3.ContentHeaderTitle)
}

type mockDataSource_screenphoto struct {
	q question
	questionAnswerDataSource
}

func (m *mockDataSource_screenphoto) question(questionID string) question {
	return m.q
}

func TestScreenPhoto_requirementsMet(t *testing.T) {
	s := &photoScreen{
		screenInfo: &screenInfo{},
		PhotoQuestions: []question{
			&photoQuestion{
				questionInfo: &questionInfo{},
			},
			&photoQuestion{
				questionInfo: &questionInfo{},
			},
		},
		RequiresAtleastOneQuestionAnswered: true,
	}

	// when both questions are optional, requirements of screen should not be met
	// because at least one of the questions is required to be answered
	if res, err := s.requirementsMet(&mockDataSource_screenphoto{}); err == nil || res {
		t.Fatal("Requirements for screen should not be met if the not a single question on the screen is answered.")
	}

	// lets answer one question and ensure requirements are met
	s.PhotoQuestions[0].(*photoQuestion).answer = &photoSectionAnswer{
		Sections: []*photoSectionAnswerItem{&photoSectionAnswerItem{}},
	}
	if res, err := s.requirementsMet(&mockDataSource_screenphoto{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected photo screen to have its requirements met")

	}

	// when screen is hidden even if questions are required, requirements should be met
	s.setVisibility(hidden)
	s.PhotoQuestions[0].(*photoQuestion).Required = true
	s.PhotoQuestions[1].(*photoQuestion).Required = true
	if res, err := s.requirementsMet(&mockDataSource_screenphoto{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected photo screen to have its requirements met")

	}

	// if the requirements for the questions are not met, the requirements for the screen should not be met
	s.setVisibility(visible)
	s.PhotoQuestions[0].setVisibility(visible)
	s.PhotoQuestions[1].setVisibility(visible)
	if res, err := s.requirementsMet(&mockDataSource_screenphoto{}); err == nil || res {
		t.Fatal("Requirements for screen should not be met if the requirements for its questions are not met")
	}
}

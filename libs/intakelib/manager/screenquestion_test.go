package manager

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

var screenQuestionJSON = `
{
	"questions": [{
		"question": "q_derm_rash_hand_locations",
		"id": "40552",
		"question_title": "Where is the rash located on your hands?",
		"question_title_has_tokens": false,
		"question_type": "q_type_single_select",
		"type": "q_type_single_select",
		"question_summary": "Hand locations",
		"potential_answers": [{
			"id": "126321",
			"potential_answer": "Top",
			"potential_answer_summary": "Top",
			"answer_type": "a_type_multiple_choice",
			"type": "a_type_multiple_choice",
			"ordering": "0",
			"to_alert": false,
			"answer_tag": "a_derm_rash_hand_locations_top"
		}, {
			"id": "126322",
			"potential_answer": "Palms",
			"potential_answer_summary": "Palms",
			"answer_type": "a_type_multiple_choice",
			"type": "a_type_multiple_choice",
			"ordering": "1",
			"to_alert": false,
			"answer_tag": "a_derm_rash_hand_locations_palms"
		}, {
			"id": "126323",
			"potential_answer": "Both top and palms",
			"potential_answer_summary": "Both top and palms",
			"answer_type": "a_type_multiple_choice",
			"type": "a_type_multiple_choice",
			"ordering": "2",
			"to_alert": false,
			"answer_tag": "a_derm_rash_hand_locations_both_top_and_palms"
		}],
		"additional_fields": {
                "answer_groups": [
                  {
                    "count": 4,
                    "title": "Head & Neck"
                  },
                  {
                    "count": 5,
                    "title": "Torso"
                  }
                ]
              },
		"condition": {
			"op": "answer_contains_any",
			"type": "answer_contains_any",
			"question": "q_derm_rash_affected_areas",
			"question_id": "40551",
			"potential_answers_id": ["126317"],
			"potential_answers": ["a_derm_rash_affected_areas_hands"]
		},
		"to_prefill": false,
		"prefilled_with_previous_answers": false,
		"required": true,
		"to_alert": false,
		"alert_text": ""
	},
	{
		"question": "q_derm_rash_locations_that_make_rash_worse",
		"id": "40577",
		"question_title": "What locations make it worse?",
		"question_title_has_tokens": false,
		"question_type": "q_type_free_text",
		"type": "q_type_free_text",
		"question_summary": "Locations that make rash worse",
		"additional_fields": {
			"placeholder_text": "Describe what locations make your rash worse…"
		},
		"condition": {
			"op": "answer_contains_any",
			"type": "answer_contains_any",
			"question": "q_derm_rash_anything_that_causes_a_rash_or_makes_it_worse",
			"question_id": "40576",
			"potential_answers_id": ["126429"],
			"potential_answers": ["a_derm_rash_anything_that_causes_a_rash_or_makes_it_worse_certain_locations"]
		},
		"to_prefill": false,
		"prefilled_with_previous_answers": false,
		"required": true,
		"to_alert": false,
		"alert_text": ""
	}],
	"type": "screen_type_questions"
}
`

func TestScreenQuestion_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	qs := &questionScreen{}
	if err := qs.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "", qs.Title)
	test.Equals(t, 2, len(qs.Questions))
	_, ok := qs.Questions[0].(*multipleChoiceQuestion)
	test.Equals(t, true, ok)
	_, ok = qs.Questions[1].(*freeTextQuestion)
	test.Equals(t, true, ok)
}

func TestScreenQuestion_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	qs := &questionScreen{}
	if err := qs.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	qs2 := qs.staticInfoCopy(nil).(*questionScreen)

	test.Equals(t, len(qs.Questions), len(qs2.Questions))
	test.Equals(t, qs2, qs2.Questions[0].layoutParent())
	test.Equals(t, qs2, qs2.Questions[1].layoutParent())

	test.Equals(t, true, qs.Questions[0] != qs2.Questions[0])

	test.Equals(t, true, qs2.Questions[0] != nil)
	test.Equals(t, true, qs2.Questions[1] != nil)

	test.Equals(t, true, qs.Questions[0] != qs2.Questions[0])
	test.Equals(t, true, qs.Questions[1] != qs2.Questions[1])

	// lets also make sure that title parsing works
	qs.ContentHeaderTitle = "Hi <parent_answer_text>"
	qs.ContentHeaderTitleHasTokens = true

	qs3 := qs.staticInfoCopy(map[string]string{"answer": "spruce"}).(*questionScreen)
	test.Equals(t, "Hi spruce", qs3.ContentHeaderTitle)

}

func TestScreenQuestion_Transforming(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenQuestionJSON), &data); err != nil {
		t.Fatal(err)
	}

	qs := &questionScreen{}
	if err := qs.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	transformedQ, err := qs.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	marshalledQ, err := proto.Marshal(transformedQ.(proto.Message))
	if err != nil {
		t.Fatal(err)
	}

	var unmarshalMapledQ intake.QuestionScreen
	if err := proto.Unmarshal(marshalledQ, &unmarshalMapledQ); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "", qs.Title)
	test.Equals(t, 2, len(unmarshalMapledQ.Questions))
	test.Equals(t, intake.QuestionData_MULTIPLE_CHOICE, *unmarshalMapledQ.Questions[0].Type)

	var mcq intake.MultipleChoiceQuestion
	if err := proto.Unmarshal(unmarshalMapledQ.Questions[0].Data, &mcq); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, intake.QuestionData_FREE_TEXT, *unmarshalMapledQ.Questions[1].Type)

	var ftq intake.FreeTextQuestion
	if err := proto.Unmarshal(unmarshalMapledQ.Questions[1].Data, &ftq); err != nil {
		t.Fatal(err)
	}
}

type mockDataSource_screenQuestion struct {
	q question
	questionAnswerDataSource
}

func (m *mockDataSource_screenQuestion) question(questionID string) question {
	return m.q
}

func TestScreenQuestion_requirementsMet(t *testing.T) {
	s := &questionScreen{
		screenInfo: &screenInfo{},
		Questions: []question{
			&multipleChoiceQuestion{
				questionInfo: &questionInfo{},
			},
			&freeTextQuestion{
				questionInfo: &questionInfo{},
			},
		},
	}

	// when both questions are optional, requirements of screen should be met
	if res, err := s.requirementsMet(&mockDataSource_screenQuestion{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected questions screen to have its requirements met")
	}

	// when screen is hidden even if questions are required, requirements should be met
	s.setVisibility(hidden)
	s.Questions[0].(*multipleChoiceQuestion).Required = true
	s.Questions[1].(*freeTextQuestion).Required = true
	if res, err := s.requirementsMet(&mockDataSource_screenQuestion{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected questions screen to have its requirements met")
	}

	// if the requirements for the questions are not met, the requirements for the screen should not be met
	s.setVisibility(visible)
	s.Questions[0].setVisibility(visible)
	s.Questions[1].setVisibility(visible)
	if res, err := s.requirementsMet(&mockDataSource_screenQuestion{}); err == nil || res {
		t.Fatal("Requirements for screen should not be met if the requirements for its questions are not met")
	}

	// even if both questions are optional, if the screen is configured to require an answer for alteast
	// one question, then requirements should fail if no answers are present for all questions
	s.RequiresAtleastOneQuestionAnswered = true
	s.Questions[0].(*multipleChoiceQuestion).Required = false
	s.Questions[1].(*freeTextQuestion).Required = false
	if res, err := s.requirementsMet(&mockDataSource_screenQuestion{}); err == nil || res {
		t.Fatal(`Requirements for screen should not be met if no question is answered
			when the screen is configured to require atleast one question to be answered.`)
	}

	// an empty answer should still cause requirements to fail
	if res, err := s.requirementsMet(&mockDataSource_screenQuestion{}); err == nil || res {
		t.Fatal("Requirements for screen should not be met if the requirements for its questions if all answers to questions are empty")
	}

	// now that at least one question has been answered, requirements should be met.
	s.Questions[0].(*multipleChoiceQuestion).answer = &multipleChoiceAnswer{Answers: []topLevelAnswerItem{&multipleChoiceAnswerSelection{}, &multipleChoiceAnswerSelection{}}}
	if res, err := s.requirementsMet(&mockDataSource_screenQuestion{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected questions screen to have its requirements met")
	}
}

func TestScreenQuestion_registerSubscreens(t *testing.T) {

	m1 := &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			LayoutUnitID: "m1",
		},
	}
	m2 := &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			LayoutUnitID: "m2",
		},
	}

	s := &questionScreen{
		subscreensMap: map[string][]screen{},
		screenInfo:    &screenInfo{},
		Questions: []question{
			m1,
			m2,
		},
	}

	s.registerSubscreensForQuestion(m1, []screen{nil, nil})
	s.registerSubscreensForQuestion(m1, []screen{nil, nil})

	// ensure that subscreens for the given question are registered just once
	test.Equals(t, 2, len(s.subscreensMap["m1"]))

	s.registerSubscreensForQuestion(m2, []screen{nil, nil})

	// ensure 2 subscreens for question
	test.Equals(t, 2, len(s.subscreensMap["m2"]))

	// ensure there are a total of 6 subscreens on the screen
	test.Equals(t, 4, len(s.subscreens()))

	// deregister screens for m1
	s.deregisterSubscreensForQuestion(m1)

	// ensure there are only 2 remaining screens
	test.Equals(t, 2, len(s.subscreens()))

	// deregister screens for m2
	s.deregisterSubscreensForQuestion(m2)

	// ensure no screens remain
	test.Equals(t, 0, len(s.subscreens()))

}

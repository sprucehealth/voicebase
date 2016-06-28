package manager

import (
	"container/list"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const multipleChoiceJSON = `
			{
              "question": "q_derm_eczema_type_of_healthcare_provider_previously_seen",
              "question_id": "40501",
              "question_title": "What type of healthcare provider have you previously seen for eczema?",
              "question_title_has_tokens": false,
              "question_type": "q_type_multiple_choice",
              "type": "q_type_multiple_choice",
              "question_summary": "Type of healthcare provider previously seen",
              "potential_answers": [
                {
                  "potential_answer_id": "126135",
                  "potential_answer": "Internal medicine, family practice, or general practitioner",
                  "potential_answer_summary": "Internal medicine, family practice, or general practitioner",
                  "answer_type": "a_type_multiple_choice",
                  "type": "a_type_multiple_choice",
                  "ordering": "0",
                  "to_alert": false,
                  "answer_tag": "a_derm_eczema_type_of_healthcare_provider_previously_seen_internal_medicine_family_practice_or_general_practitioner"
                },
                {
                  "potential_answer_id": "126139",
                  "potential_answer": "Other",
                  "potential_answer_summary": "Other",
                  "answer_type": "a_type_multiple_choice_other_free_text",
                  "type": "a_type_multiple_choice_other_free_text",
                  "ordering": "4",
                  "to_alert": false,
                  "answer_tag": "a_derm_eczema_type_of_healthcare_provider_previously_seen_other",
                  "client_data" : {
                  	"placeholder_text": "Type another"
                  }
                },
                {
                  "potential_answer_id": "126140",
                  "potential_answer": "None of the above",
                  "potential_answer_summary": "None of the above",
                  "answer_type": "a_type_multiple_choice_none",
                  "type": "a_type_multiple_choice_none",
                  "ordering": "4",
                  "to_alert": false,
                  "answer_tag": "a_derm_eczema_type_of_healthcare_provider_previously_seen_other"
                }
              ],
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
              "to_prefill": false,
              "prefilled_with_previous_answers": true,
              "required": true,
              "to_alert": false,
              "alert_text": "",
              "answers": [{
					"answer_id": "64319",
					"question_id": "40501",
					"potential_answer_id": "126135",
					"type": "q_type_multiple_choice"
				}, {
					"answer_id": "64320",
					"question_id": "40501",
					"potential_answer_id": "126139",
					"answer_text" : "geoahg",
					"type": "q_type_multiple_choice"
				}]
            }`

const multipleChoiceWithoutClientDataJSON = `
{
  "question": "q_derm_eczema_type_of_healthcare_provider_previously_seen",
  "question_id": "40501",
  "question_title": "What type of healthcare provider have you previously seen for eczema?",
  "question_title_has_tokens": false,
  "question_type": "q_type_multiple_choice",
  "type": "q_type_multiple_choice",
  "question_summary": "Type of healthcare provider previously seen",
  "potential_answers": [
    {
      "potential_answer_id": "126135",
      "potential_answer": "Internal medicine, family practice, or general practitioner",
      "potential_answer_summary": "Internal medicine, family practice, or general practitioner",
      "answer_type": "a_type_multiple_choice",
      "type": "a_type_multiple_choice",
      "ordering": "0",
      "to_alert": false,
      "answer_tag": "a_derm_eczema_type_of_healthcare_provider_previously_seen_internal_medicine_family_practice_or_general_practitioner"
    },
    {
      "potential_answer_id": "126139",
      "potential_answer": "Other",
      "potential_answer_summary": "Other",
      "answer_type": "a_type_multiple_choice_other_free_text",
      "type": "a_type_multiple_choice_other_free_text",
      "ordering": "4",
      "to_alert": false,
      "answer_tag": "a_derm_eczema_type_of_healthcare_provider_previously_seen_other",
      "client_data" : {
      	"placeholder_text": "Type another"
      }
    },
    {
      "potential_answer_id": "126140",
      "potential_answer": "None of the above",
      "potential_answer_summary": "None of the above",
      "answer_type": "a_type_multiple_choice_none",
      "type": "a_type_multiple_choice_none",
      "ordering": "4",
      "to_alert": false,
      "answer_tag": "a_derm_eczema_type_of_healthcare_provider_previously_seen_other"
    }
  ],
  "to_prefill": false,
  "prefilled_with_previous_answers": false,
  "required": true,
  "to_alert": false,
  "alert_text": "",
  "answers": [{
		"answer_id": "64319",
		"question_id": "40501",
		"potential_answer_id": "126135",
		"type": "q_type_multiple_choice"
	}, {
		"answer_id": "64320",
		"question_id": "40501",
		"potential_answer_id": "126139",
		"answer_text" : "geoahg",
		"type": "q_type_multiple_choice"
	}]
}`

func TestMultipleChoiceQuestion_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(multipleChoiceJSON), &data); err != nil {
		t.Fatal(err)
	}

	mcq := &multipleChoiceQuestion{}
	if err := mcq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "What type of healthcare provider have you previously seen for eczema?", mcq.questionInfo.Title)
	test.Equals(t, "40501", mcq.questionInfo.ID)
	test.Equals(t, true, mcq.prefilled())
	test.Equals(t, 3, len(mcq.PotentialAnswers))
	test.Equals(t, "a_type_multiple_choice", mcq.PotentialAnswers[0].Type)
	test.Equals(t, "Internal medicine, family practice, or general practitioner", mcq.PotentialAnswers[0].Text)
	test.Equals(t, "Internal medicine, family practice, or general practitioner", mcq.PotentialAnswers[0].Summary)
	test.Equals(t, "a_type_multiple_choice_other_free_text", mcq.PotentialAnswers[1].Type)
	test.Equals(t, "Other", mcq.PotentialAnswers[1].Text)
	test.Equals(t, "Type another", mcq.PotentialAnswers[1].PlaceholderText)
	test.Equals(t, "Other", mcq.PotentialAnswers[1].Summary)
	test.Equals(t, 2, len(mcq.AnswerGroups))
	test.Equals(t, "Head & Neck", mcq.AnswerGroups[0].Title)
	test.Equals(t, 4, mcq.AnswerGroups[0].Count)
	test.Equals(t, "Torso", mcq.AnswerGroups[1].Title)
	test.Equals(t, 5, mcq.AnswerGroups[1].Count)
	test.Equals(t, true, mcq.answer != nil)
	test.Equals(t, 2, len(mcq.answer.Answers))
}

func TestMultipleChoice_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(multipleChoiceJSON), &data); err != nil {
		t.Fatal(err)
	}

	mcq := &multipleChoiceQuestion{}
	if err := mcq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// nullify answers given its not static info
	mcq.answer = nil
	mcq2 := mcq.staticInfoCopy(nil).(*multipleChoiceQuestion)
	test.Equals(t, mcq, mcq2)
}

func TestMultipleChoiceQuestion_ParsingNoClientData(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(multipleChoiceWithoutClientDataJSON), &data); err != nil {
		t.Fatal(err)
	}

	mcq := &multipleChoiceQuestion{}
	if err := mcq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "What type of healthcare provider have you previously seen for eczema?", mcq.questionInfo.Title)
	test.Equals(t, "40501", mcq.questionInfo.ID)
	test.Equals(t, 3, len(mcq.PotentialAnswers))
	test.Equals(t, "a_type_multiple_choice", mcq.PotentialAnswers[0].Type)
	test.Equals(t, "Internal medicine, family practice, or general practitioner", mcq.PotentialAnswers[0].Text)
	test.Equals(t, "Internal medicine, family practice, or general practitioner", mcq.PotentialAnswers[0].Summary)
	test.Equals(t, "a_type_multiple_choice_other_free_text", mcq.PotentialAnswers[1].Type)
	test.Equals(t, "Other", mcq.PotentialAnswers[1].Text)
	test.Equals(t, "Type another", mcq.PotentialAnswers[1].PlaceholderText)
	test.Equals(t, "Other", mcq.PotentialAnswers[1].Summary)
	test.Equals(t, true, mcq.answer != nil)
	test.Equals(t, 2, len(mcq.answer.Answers))
}

func TestMultipleChoiceQuestion_Answer(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(multipleChoiceJSON), &data); err != nil {
		t.Fatal(err)
	}

	mcq := &multipleChoiceQuestion{}
	if err := mcq.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	// set answer with none of the above selected only
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126140",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// set answer with multiple options (one of which is other)
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126135",
			},
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126139",
				Text:              "Hello",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// set answer with option and none of the above (should fail)
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126135",
			},
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126140",
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// set answer with other and none of the above (should fail)
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126139",
				Text:              "Hello",
			},
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126140",
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// set answer with option and text set (should fail)
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126135",
				Text:              "Hello",
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// set answer with no options (should succeed)
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{},
	}); err != nil {
		t.Fatal(err)
	}

	// set answer with non-existent option (should fail)
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "124031285",
				Text:              "Hello",
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// change question type to single select and then try setting multiple answers (should fail)
	mcq.Type = questionTypeSingleSelect.String()
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126135",
			},
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126139",
				Text:              "Hello",
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}

	// do the same for segmented control
	mcq.Type = questionTypeSegmentedControl.String()
	if err := mcq.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126135",
			},
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: "126139",
				Text:              "Hello",
			},
		},
	}); err == nil {
		t.Fatal("Expected invalid answer")
	}
}

func TestMultipleChoiceQuestion_requirementsMet_subscreens(t *testing.T) {

	subQ := &freeTextQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
	}

	// required question, with answers having subscreens, should have its requirements met only if
	// all the screens requirements are met.
	s := &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
		answer: &multipleChoiceAnswer{
			Answers: []topLevelAnswerItem{
				&multipleChoiceAnswerSelection{
					Text: "agkahg",
					subScreens: []screen{
						&questionScreen{
							screenInfo: &screenInfo{},
							Questions: []question{
								subQ,
								&freeTextQuestion{
									questionInfo: &questionInfo{
										Required: true,
									},
									answer: &freeTextAnswer{
										Text: "Agag",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := s.requirementsMet(&mockDataSource_question{}); err == nil {
		t.Fatal("Expected requirements to not be met for question")
	} else if err != errSubQuestionRequirements {
		t.Fatalf("Expected a subquesitons requirements not met error but got %T", err)
	}

	// now lets make all required subquestions have an answer
	// and all requirements should be met.
	subQ.answer = &freeTextAnswer{Text: "adgkb"}
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}

	// lets have a question with subscreens that are all just optional
	// and screens that are not question screens and ensure
	// that the requirements should be met
	s = &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
		answer: &multipleChoiceAnswer{
			Answers: []topLevelAnswerItem{
				&multipleChoiceAnswerSelection{
					Text: "agkahg",
					subScreens: []screen{
						&questionScreen{
							screenInfo: &screenInfo{},
							Questions: []question{
								&freeTextQuestion{
									questionInfo: &questionInfo{},
								},
							},
						},
						&warningPopupScreen{
							screenInfo: &screenInfo{
								screenClientData: &screenClientData{},
							},
						},
					},
				},
			},
		},
	}
	if res, err := s.requirementsMet(&mockDataSource_question{}); err != nil {
		t.Fatal(err)
	} else if !res {
		t.Fatal("Expected requirements to be met for question")
	}
}

func TestMultipleChoiceQuestion_answerMerge(t *testing.T) {

	subscreens1 := []screen{

		&questionScreen{
			screenInfo: &screenInfo{
				LayoutUnitID: "1",
			},
			Questions: []question{
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
				},
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
					answer: &freeTextAnswer{
						Text: "Agag",
					},
				},
			},
		},

		&questionScreen{
			screenInfo: &screenInfo{
				LayoutUnitID: "2",
			},
			Questions: []question{
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
				},
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
					answer: &freeTextAnswer{
						Text: "Agag",
					},
				},
			},
		},
	}

	subscreens2 := []screen{
		&questionScreen{
			screenInfo: &screenInfo{
				LayoutUnitID: "3",
			},
			Questions: []question{
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
				},
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
					answer: &freeTextAnswer{
						Text: "dgahaharha",
					},
				},
			},
		},
		&questionScreen{
			screenInfo: &screenInfo{
				LayoutUnitID: "4",
			},
			Questions: []question{
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
				},
				&freeTextQuestion{
					questionInfo: &questionInfo{
						Required: true,
					},
					answer: &freeTextAnswer{
						Text: "dgahaharha",
					},
				},
			},
		},
	}

	// prepare a multiple choice question with top level answers
	// for which subscreens exist
	s := &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			Required: true,
		},
		PotentialAnswers: []*potentialAnswer{
			{
				ID:   "10",
				Type: answerTypeOption,
			},
			{
				ID:   "11",
				Type: answerTypeOther,
			},
		},
		potentialAnswerMap: map[string]*potentialAnswer{
			"10": &potentialAnswer{},
			"11": &potentialAnswer{},
		},
		answer: &multipleChoiceAnswer{
			Answers: []topLevelAnswerItem{
				&multipleChoiceAnswerSelection{
					Text:              "answer1",
					PotentialAnswerID: "10",
					subScreens:        subscreens1,
				},
				&multipleChoiceAnswerSelection{
					Text:              "answer2",
					PotentialAnswerID: "11",
					subScreens:        subscreens2,
				},
			},
		},
	}

	qScreen := &questionScreen{
		screenInfo: &screenInfo{},
		Questions: []question{
			s,
		},
		subscreensMap: map[string][]screen{},
		allSubscreens: []screen{},
	}

	// setup the parent of the question
	s.setLayoutParent(qScreen)

	se := &section{
		LayoutUnitID: "se1",
	}
	qScreen.setLayoutParent(se)

	// ensure that the screen is already available in the section-level list in the visit manager
	l := list.New()
	l.PushBack(qScreen)

	s.subquestionsManager = newSubquestionsManagerForQuestion(s, &visitManager{
		questionMap: map[string]*questionData{},
		sectionScreensMap: map[string]*list.List{
			"se1": l,
		},
	})
	s.subquestionsManager.inflateSubscreensForPatientAnswer()

	// now if I attempt to re-answer the question with yet another top level
	// selection added, the subscreens of the other two answers, if they remain selected
	// should also be maintained
	if err := s.setPatientAnswer(&multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				Text:              "answer1",
				PotentialAnswerID: "10",
			},
			&multipleChoiceAnswerSelection{
				Text:              "answer2",
				PotentialAnswerID: "11",
			},
			&multipleChoiceAnswerSelection{
				Text:              "answer3",
				PotentialAnswerID: "11",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// there should now be 3 top level answers set
	test.Equals(t, 3, len(s.answer.Answers))

	// subscreens should match for answer 1 (before and after being set)
	test.Equals(t, subscreens1, s.answer.Answers[0].subscreens())

	// same for answer2
	test.Equals(t, subscreens2, s.answer.Answers[1].subscreens())

	// answer 3 should not have any subscreens
	test.Equals(t, 0, len(s.answer.Answers[2].subscreens()))

	// top level screen should contain 4 subscreens in order
	subscreens := s.layoutParent().(*questionScreen).subscreens()
	test.Equals(t, 4, len(subscreens))

	// given that the subscreens were just merged, there should be no alteration
	// to their layoutUnitIDs as assigned before the answer being set.
	for i, ss := range subscreens {
		test.Equals(t, strconv.Itoa(i+1), ss.layoutUnitID())
	}

}

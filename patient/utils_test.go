package patient

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
)

type mockDataAPI_prefillQuestions struct {
	api.DataAPI
	answers map[string][]common.Answer
}

func (m *mockDataAPI_prefillQuestions) PreviousPatientAnswersForQuestions(questionTags []string, id int64, before time.Time) (map[string][]common.Answer, error) {
	return m.answers, nil
}
func (m *mockDataAPI_prefillQuestions) PatientPhotoSectionsForQuestionIDs(questionIDs []int64, patientID, visitID int64) (map[int64][]common.Answer, error) {
	return nil, nil
}
func (m *mockDataAPI_prefillQuestions) AnswersForQuestions(questionIDs []int64, i api.IntakeInfo) (map[int64][]common.Answer, error) {
	return make(map[int64][]common.Answer), nil
}

func TestPrefillQuestions(t *testing.T) {

	visitLayout := &info_intake.InfoIntakeLayout{
		Sections: []*info_intake.Section{
			{
				Screens: []*info_intake.Screen{
					{
						Questions: []*info_intake.Question{
							{
								QuestionID:   100,
								QuestionTag:  "q_free_text_tag",
								QuestionType: info_intake.QuestionTypeFreeText,
								ToPrefill:    true,
							},
							{
								QuestionID:   101,
								QuestionTag:  "q_multiple_choice_tag",
								QuestionType: info_intake.QuestionTypeMultipleChoice,
								ToPrefill:    true,
								PotentialAnswers: []*info_intake.PotentialAnswer{
									{
										AnswerID:  1,
										Answer:    "Option 1",
										AnswerTag: "q_multiple_choice_tag_option_1",
									},
									{
										AnswerID:  2,
										Answer:    "Option 2",
										AnswerTag: "q_multiple_choice_tag_option_2",
									},
									{
										AnswerID:  3,
										Answer:    "Option 3",
										AnswerTag: "q_multiple_choice_tag_option_3",
									},
								},
							},
							{
								QuestionID:   102,
								QuestionTag:  "q_single_select_tag",
								QuestionType: info_intake.QuestionTypeMultipleChoice,
								ToPrefill:    true,
								PotentialAnswers: []*info_intake.PotentialAnswer{
									{
										AnswerID:  4,
										Answer:    "Option 1",
										AnswerTag: "q_single_select_tag_option_1",
									},
									{
										AnswerID:  5,
										Answer:    "Option 2",
										AnswerTag: "q_single_select_tag_option_2",
									},
								},
							},
							{
								QuestionID:   103,
								QuestionTag:  "q_multiple_choice_tag_unmatched_answer",
								QuestionType: info_intake.QuestionTypeMultipleChoice,
								ToPrefill:    true,
								PotentialAnswers: []*info_intake.PotentialAnswer{
									{
										AnswerID:  1,
										Answer:    "Option 1",
										AnswerTag: "q_multiple_choice_tag_unmatched_answer_option_1",
									},
									{
										AnswerID:  2,
										Answer:    "Option 2",
										AnswerTag: "q_multiple_choice_tag_unmatched_answer_option_2",
									},
									{
										AnswerID:  3,
										Answer:    "Option 3",
										AnswerTag: "q_multiple_choice_tag_unmatched_answer_option_3",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	freeTextResponse := "free text response"

	m := &mockDataAPI_prefillQuestions{
		answers: map[string][]common.Answer{
			"q_free_text_tag": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID: encoding.NewObjectID(10),
					QuestionID:     encoding.NewObjectID(90),
					AnswerText:     freeTextResponse,
				},
			},
			"q_multiple_choice_tag": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.NewObjectID(11),
					QuestionID:        encoding.NewObjectID(91),
					PotentialAnswer:   "Option 1",
					PotentialAnswerID: encoding.NewObjectID(6),
				},
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.NewObjectID(12),
					QuestionID:        encoding.NewObjectID(91),
					PotentialAnswer:   "Option 2",
					PotentialAnswerID: encoding.NewObjectID(7),
				},
				&common.AnswerIntake{
					AnswerIntakeID: encoding.NewObjectID(12),
					QuestionID:     encoding.NewObjectID(91),
					AnswerText:     freeTextResponse,
				},
			},
			"q_single_select_tag": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.NewObjectID(12),
					QuestionID:        encoding.NewObjectID(92),
					PotentialAnswer:   "Option 1",
					PotentialAnswerID: encoding.NewObjectID(8),
				},
			},
			"q_multiple_choice_tag_unmatched_answer": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.NewObjectID(13),
					QuestionID:        encoding.NewObjectID(103),
					PotentialAnswer:   "Option 1",
					PotentialAnswerID: encoding.NewObjectID(9),
				},
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.NewObjectID(14),
					QuestionID:        encoding.NewObjectID(103),
					PotentialAnswer:   "Option 2",
					PotentialAnswerID: encoding.NewObjectID(10),
				},
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.NewObjectID(14),
					QuestionID:        encoding.NewObjectID(103),
					PotentialAnswer:   "Option 4",
					PotentialAnswerID: encoding.NewObjectID(11),
				},
			},
		},
	}

	if err := populateLayoutWithAnswers(visitLayout, m, nil, time.Duration(0), &common.PatientVisit{
		Status: common.PVStatusOpen,
	}); err != nil {
		t.Fatalf(err.Error())
	}

	questions := visitLayout.Sections[0].Screens[0].Questions

	if len(questions[0].Answers) != 1 {
		t.Fatalf("Expected prefilled answer for free text but got none")
	} else if !questions[0].PrefilledWithPreviousAnswers {
		t.Fatalf("Expected question to be marked as being prefilled with previous answers but wasn't")
	} else if ai := questions[0].Answers[0].(*common.AnswerIntake); ai.AnswerText != freeTextResponse {
		t.Fatalf("Expected answer for free text response to be %s but was %s", freeTextResponse, ai.AnswerText)
	}

	if len(questions[1].Answers) != 3 {
		t.Fatalf("Expected prefilled answer for multiple choice but got %d", len(questions[1].Answers))
	} else if !questions[1].PrefilledWithPreviousAnswers {
		t.Fatalf("Expected question to be marked as being prefilled with previous answers but wasn't")
	} else if ai := questions[1].Answers[0].(*common.AnswerIntake); ai.PotentialAnswer != "Option 1" {
		t.Fatalf("Expected option 1 to be selected but it wasnt")
	} else if ai.PotentialAnswerID.Int64() != questions[1].PotentialAnswers[0].AnswerID {
		t.Fatalf("Expected the potential answer id of the patient answer prefilled to match the potential answer id of the question but it didnt")
	} else if ai := questions[1].Answers[1].(*common.AnswerIntake); ai.PotentialAnswer != "Option 2" {
		t.Fatalf("Expected option 2 to be selected but it wasnt")
	} else if ai.PotentialAnswerID.Int64() != questions[1].PotentialAnswers[1].AnswerID {
		t.Fatalf("Expected the potential answer id of the patient answer prefilled to match the potential answer id of the question but it didnt")
	} else if ai := questions[1].Answers[2].(*common.AnswerIntake); ai.AnswerText != freeTextResponse {
		t.Fatalf("Expected free text response to be present but it wasn't")
	}

	if len(questions[2].Answers) != 1 {
		t.Fatalf("Expected prefilled answer for single select but got %d", len(questions[2].Answers))
	} else if !questions[2].PrefilledWithPreviousAnswers {
		t.Fatalf("Expected question to be marked as being prefilled with previous answers but wasn't")
	} else if ai := questions[2].Answers[0].(*common.AnswerIntake); ai.PotentialAnswer != "Option 1" {
		t.Fatalf("Expected option 1 to be selected but it wasnt")
	}

	// even though there were some matches question 4 should not have any matches as one of them
	if len(questions[3].Answers) != 0 {
		t.Fatalf("Expected no answers to be populated for a question where the patient picked an answer that did not match any of the answers in the current set")
	} else if questions[3].PrefilledWithPreviousAnswers {
		t.Fatal("Didn't exist the question to indicate that it was prefilled with answers")
	}
}

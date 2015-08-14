package patient

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_prefillQuestions struct {
	api.DataAPI
	answers map[string][]common.Answer
}

func (m *mockDataAPI_prefillQuestions) PreviousPatientAnswersForQuestions(questionTags []string, patientID common.PatientID, before time.Time) (map[string][]common.Answer, error) {
	return m.answers, nil
}
func (m *mockDataAPI_prefillQuestions) PatientPhotoSectionsForQuestionIDs(questionIDs []int64, patientID common.PatientID, visitID int64) (map[int64][]common.Answer, error) {
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
					AnswerIntakeID: encoding.DeprecatedNewObjectID(10),
					QuestionID:     encoding.DeprecatedNewObjectID(90),
					AnswerText:     freeTextResponse,
				},
			},
			"q_multiple_choice_tag": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.DeprecatedNewObjectID(11),
					QuestionID:        encoding.DeprecatedNewObjectID(91),
					PotentialAnswer:   "Option 1",
					PotentialAnswerID: encoding.DeprecatedNewObjectID(6),
				},
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.DeprecatedNewObjectID(12),
					QuestionID:        encoding.DeprecatedNewObjectID(91),
					PotentialAnswer:   "Option 2",
					PotentialAnswerID: encoding.DeprecatedNewObjectID(7),
				},
				&common.AnswerIntake{
					AnswerIntakeID: encoding.DeprecatedNewObjectID(12),
					QuestionID:     encoding.DeprecatedNewObjectID(91),
					AnswerText:     freeTextResponse,
				},
			},
			"q_single_select_tag": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.DeprecatedNewObjectID(12),
					QuestionID:        encoding.DeprecatedNewObjectID(92),
					PotentialAnswer:   "Option 1",
					PotentialAnswerID: encoding.DeprecatedNewObjectID(8),
				},
			},
			"q_multiple_choice_tag_unmatched_answer": []common.Answer{
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.DeprecatedNewObjectID(13),
					QuestionID:        encoding.DeprecatedNewObjectID(103),
					PotentialAnswer:   "Option 1",
					PotentialAnswerID: encoding.DeprecatedNewObjectID(9),
				},
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.DeprecatedNewObjectID(14),
					QuestionID:        encoding.DeprecatedNewObjectID(103),
					PotentialAnswer:   "Option 2",
					PotentialAnswerID: encoding.DeprecatedNewObjectID(10),
				},
				&common.AnswerIntake{
					AnswerIntakeID:    encoding.DeprecatedNewObjectID(14),
					QuestionID:        encoding.DeprecatedNewObjectID(103),
					PotentialAnswer:   "Option 4",
					PotentialAnswerID: encoding.DeprecatedNewObjectID(11),
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

type mockDataAPIPathwayForPatient struct {
	api.DataAPI
	pathways map[string]*common.Pathway // pathway tag -> pathway
}

func (d *mockDataAPIPathwayForPatient) PathwayForTag(tag string, opts api.PathwayOption) (*common.Pathway, error) {
	if p := d.pathways[tag]; p != nil {
		return p, nil
	}
	return nil, api.ErrNotFound("pathway")
}

func TestPathwayForPatient(t *testing.T) {
	dataAPI := &mockDataAPIPathwayForPatient{
		pathways: map[string]*common.Pathway{
			"acne": {
				ID:  1,
				Tag: "acne",
				Details: &common.PathwayDetails{
					AgeRestrictions: []*common.PathwayAgeRestriction{
						{
							MaxAgeOfRange: ptr.Int(12),
							VisitAllowed:  false,
							Alert: &common.PathwayAlert{
								Message: "Sorry!",
							},
						},
						{
							MaxAgeOfRange:       ptr.Int(17),
							VisitAllowed:        true,
							AlternatePathwayTag: "teen_acne",
						},
						{
							MaxAgeOfRange: ptr.Int(70),
							VisitAllowed:  true,
						},
						{
							MaxAgeOfRange: nil,
							VisitAllowed:  false,
							Alert: &common.PathwayAlert{
								Message: "Not Sorry!",
							},
						},
					},
				},
			},
			"teen_acne": {
				ID:  2,
				Tag: "teen_acne",
			},
			"other": {
				ID:      3,
				Tag:     "other",
				Details: &common.PathwayDetails{},
			},
		},
	}

	// Adolescent
	_, err := pathwayForPatient(dataAPI, "acne", &common.Patient{
		DOB: encoding.Date{Year: time.Now().Year() - 11, Month: 1, Day: 1},
	})
	test.Assert(t, err != nil, "Should not allow adolescents for acne pathway")
	test.Equals(t, "Sorry!", err.(*apiservice.SpruceError).UserError)

	// Teen
	pathway, err := pathwayForPatient(dataAPI, "acne", &common.Patient{
		DOB: encoding.Date{Year: time.Now().Year() - 17, Month: 1, Day: 1},
	})
	test.OK(t, err)
	test.Equals(t, "teen_acne", pathway.Tag)

	// Adult
	pathway, err = pathwayForPatient(dataAPI, "acne", &common.Patient{
		DOB: encoding.Date{Year: time.Now().Year() - 18, Month: 1, Day: 1},
	})
	test.OK(t, err)
	test.Equals(t, "acne", pathway.Tag)

	// Senior
	_, err = pathwayForPatient(dataAPI, "acne", &common.Patient{
		DOB: encoding.Date{Year: time.Now().Year() - 75, Month: 1, Day: 1},
	})
	test.Assert(t, err != nil, "Should not allow sensiors for acne pathway")
	test.Equals(t, "Not Sorry!", err.(*apiservice.SpruceError).UserError)

	// When no explicit restrictions should not allow anyone younger than 18

	// < 18
	_, err = pathwayForPatient(dataAPI, "other", &common.Patient{
		DOB: encoding.Date{Year: time.Now().Year() - 15, Month: 1, Day: 1},
	})
	test.Assert(t, err != nil, "Should not allow < 18 for pathway without explicit restrictions")
	test.Equals(t, "Sorry, we do not support the chosen condition for people under 18.", err.(*apiservice.SpruceError).UserError)

	// >= 18
	pathway, err = pathwayForPatient(dataAPI, "other", &common.Patient{
		DOB: encoding.Date{Year: time.Now().Year() - 18, Month: 1, Day: 1},
	})
	test.OK(t, err)
	test.Equals(t, "other", pathway.Tag)
}

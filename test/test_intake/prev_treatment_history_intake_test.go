package test_intake

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test/test_integration"
	"testing"
)

const (
	prevAcnePrescriptionsSelectTag      = "q_acne_prev_prescriptions_select"
	usingPrevAcnePrescriptionTag        = "q_using_prev_acne_prescription"
	howEffectiveAcnePrescriptionTag     = "q_how_effective_prev_acne_prescription"
	usingMoreThreeMonthsPrescriptionTag = "q_use_more_three_months_prev_acne_prescription"
	irritateSkinAcnePrescriptionTag     = "q_use_more_three_months_prev_acne_prescription"
	anythingElseAcnePrescriptionTag     = "q_anything_else_prev_acne_prescription"
	benzaclinTag                        = "a_benzaclin"
	benzoylPeroxideTag                  = "a_benzoyl_peroxide"
	epiduoTag                           = "a_epiduo"
	otherPrevPrescriptionTag            = "a_other_prev_acne_prescription"
	usingPrevPrescriptionYes            = "a_using_prev_prescription_yes"
	somewhatEffectivePrescription       = "a_how_effective_prev_acne_prescription_somewhat"
	lessThanThreeMonthsPrescription     = "a_use_more_three_months_prev_acne_prescription_no"
	irritateYesPrescription             = "a_irritate_skin_prev_acne_prescription_yes"
)

func TestPrevPrescriptions(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientId, testData, t)

	// lets get the question and answer information
	questionInfos, err := testData.DataApi.GetQuestionInfoForTags([]string{prevAcnePrescriptionsSelectTag, usingPrevAcnePrescriptionTag, howEffectiveAcnePrescriptionTag, irritateSkinAcnePrescriptionTag, anythingElseAcnePrescriptionTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err)
	}

	qMapping := make(map[string]int64)
	for _, questionInfo := range questionInfos {
		qMapping[questionInfo.QuestionTag] = questionInfo.QuestionId
	}

	answerInfos, err := testData.DataApi.GetAnswerInfoForTags([]string{benzaclinTag, benzoylPeroxideTag, epiduoTag, otherPrevPrescriptionTag, usingPrevPrescriptionYes, somewhatEffectivePrescription, lessThanThreeMonthsPrescription, irritateYesPrescription}, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err)
	}

	aMapping := make(map[string]int64)
	for _, answerInfo := range answerInfos {
		aMapping[answerInfo.AnswerTag] = answerInfo.AnswerId
	}

	// lets get answer setup: answer one of the selected answers,
	// and for the rest of them select other and specify the answer text
	requestData := &apiservice.AnswerIntakeRequestBody{
		PatientVisitId: patientVisitResponse.PatientVisitId,
		Questions: []*apiservice.AnswerToQuestionItem{
			&apiservice.AnswerToQuestionItem{
				QuestionId: qMapping[prevAcnePrescriptionsSelectTag],
				AnswerIntakes: []*apiservice.AnswerItem{
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[benzaclinTag],
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[usingPrevAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[usingPrevPrescriptionYes],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[irritateSkinAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[irritateYesPrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[usingMoreThreeMonthsPrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[lessThanThreeMonthsPrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[howEffectiveAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[somewhatEffectivePrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[anythingElseAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Testing this answer",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[otherPrevPrescriptionTag],
						AnswerText:        "Duac",
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[usingPrevAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[usingPrevPrescriptionYes],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[irritateSkinAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[irritateYesPrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[usingMoreThreeMonthsPrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[lessThanThreeMonthsPrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[howEffectiveAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[somewhatEffectivePrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[anythingElseAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Testing this answer",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[otherPrevPrescriptionTag],
						AnswerText:        "Duac2",
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[usingPrevAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[usingPrevPrescriptionYes],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[irritateSkinAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[irritateYesPrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[usingMoreThreeMonthsPrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[lessThanThreeMonthsPrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[howEffectiveAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[somewhatEffectivePrescription],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[anythingElseAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Testing this answer",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitAnswersIntakeForPatient(patientId, patient.AccountId.Int64(), requestData, testData, t)

	// now attempt to get the patient visit response to ensure that the answers for the top leve question exists
	// in the structure expected
	patientVisitResponse = test_integration.GetPatientVisitForPatient(patientId, testData, t)
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionTag == prevAcnePrescriptionsSelectTag {

					// there should be 3 top level answers
					if len(question.Answers) != 3 {
						t.Fatalf("Expected 3 answers but got %d", len(question.Answers))
					}

					// there should be a potential answer id for each of the 3 top level answers
					for _, answer := range test_integration.GetAnswerIntakesFromAnswers(question.Answers, t) {
						if answer.PotentialAnswerId.Int64() == 0 {
							t.Fatalf("Expected the potential answer id to be set but it wasnt")
						}
					}

					// for each of the three there should be 5 subanswers
					for _, pAnswer := range question.Answers {
						pa := pAnswer.(*common.AnswerIntake)
						if len(pa.SubAnswers) != 5 {
							t.Fatalf("Expected 5 sub answers instead got %d", len(pa.SubAnswers))
						}
					}

					return
				}
			}
		}
	}

	t.Fatalf("Unable to find question %s in patient visit response", prevAcnePrescriptionsSelectTag)
}

const (
	acnePrevOTCSelectTag          = "q_acne_prev_otc_select"
	acneOTCProductTriedTag        = "q_acne_otc_product_tried"
	acneUsingPrevOTCTag           = "q_using_prev_acne_otc"
	acneUsingPrevOTCYesTag        = "a_using_prev_otc_yes"
	acneHowEffectivePrevOTCTag    = "q_how_effective_prev_acne_otc"
	acneHowEffectiveVeryOTCTag    = "a_how_effective_prev_acne_otc_very_effective"
	acneIrritateSkinPrevOTCTag    = "q_irritate_skin_prev_acne_otc"
	acneIrritateSkinYesPrevOTCTag = "a_irritate_skin_prev_acne_otc_yes"
	acneAnythingElsePrevOTCTag    = "q_anything_else_prev_acne_otc"
	acneFreeTag                   = "a_acne_free"
	cetaphilOTCTag                = "a_cetaphil"
	otherPrevOTCTag               = "a_other_prev_acne_otc"
)

func TestPrevAcne(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientId, testData, t)

	questionInfos, err := testData.DataApi.GetQuestionInfoForTags([]string{acnePrevOTCSelectTag, acneOTCProductTriedTag, acneUsingPrevOTCTag, acneHowEffectivePrevOTCTag, acneIrritateSkinPrevOTCTag, acneAnythingElsePrevOTCTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err)
	}

	qMapping := make(map[string]int64)
	for _, questionInfo := range questionInfos {
		qMapping[questionInfo.QuestionTag] = questionInfo.QuestionId
	}

	answerInfos, err := testData.DataApi.GetAnswerInfoForTags([]string{acneFreeTag, cetaphilOTCTag, otherPrevOTCTag, acneUsingPrevOTCYesTag, acneIrritateSkinYesPrevOTCTag, acneHowEffectiveVeryOTCTag}, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err)
	}

	aMapping := make(map[string]int64)
	for _, answerInfo := range answerInfos {
		aMapping[answerInfo.AnswerTag] = answerInfo.AnswerId
	}

	// lets get answer setup: answer one of the selected answers,
	// and for the rest of them select other and specify the answer text
	requestData := &apiservice.AnswerIntakeRequestBody{
		PatientVisitId: patientVisitResponse.PatientVisitId,
		Questions: []*apiservice.AnswerToQuestionItem{
			&apiservice.AnswerToQuestionItem{
				QuestionId: qMapping[acnePrevOTCSelectTag],
				AnswerIntakes: []*apiservice.AnswerItem{
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[acneFreeTag],
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Clean and clear",
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[cetaphilOTCTag],
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Clean lgkna",
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[otherPrevOTCTag],
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Clean and clear",
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerId: aMapping[otherPrevOTCTag],
						SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Clean and clear",
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerId: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							&apiservice.SubQuestionAnswerIntake{
								QuestionId: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitAnswersIntakeForPatient(patientId, patient.AccountId.Int64(), requestData, testData, t)

	// now attempt to get the patient visit response to ensure that the answers for the top leve question exists
	// in the structure expected
	patientVisitResponse = test_integration.GetPatientVisitForPatient(patientId, testData, t)
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionTag == acnePrevOTCSelectTag {

					if len(question.Answers) != 4 {
						t.Fatalf("Expected 4 answers but got %d", len(question.Answers))
					}

					// there should be a potential answer id for each of the top level answers
					for _, answer := range test_integration.GetAnswerIntakesFromAnswers(question.Answers, t) {
						if answer.PotentialAnswerId.Int64() == 0 {
							t.Fatalf("Expected the potential answer id to be set but it wasnt")
						}
					}

					// for each of the three there should be 5 subanswers
					for _, pAnswer := range question.Answers {
						pa := pAnswer.(*common.AnswerIntake)
						if len(pa.SubAnswers) != 5 {
							t.Fatalf("Expected 5 sub answers instead got %d", len(pa.SubAnswers))
						}
					}

					return
				}
			}
		}
	}
}

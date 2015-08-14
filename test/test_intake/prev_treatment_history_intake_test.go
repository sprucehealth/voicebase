package test_intake

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
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
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.ID
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// lets get the question and answer information
	questionInfos, err := testData.DataAPI.GetQuestionInfoForTags([]string{prevAcnePrescriptionsSelectTag, usingPrevAcnePrescriptionTag, howEffectiveAcnePrescriptionTag, irritateSkinAcnePrescriptionTag, anythingElseAcnePrescriptionTag}, api.LanguageIDEnglish)
	test.OK(t, err)

	qMapping := make(map[string]int64)
	for _, questionInfo := range questionInfos {
		qMapping[questionInfo.QuestionTag] = questionInfo.QuestionID
	}

	answerInfos, err := testData.DataAPI.GetAnswerInfoForTags([]string{benzaclinTag, benzoylPeroxideTag, epiduoTag, otherPrevPrescriptionTag, usingPrevPrescriptionYes, somewhatEffectivePrescription, lessThanThreeMonthsPrescription, irritateYesPrescription}, api.LanguageIDEnglish)
	test.OK(t, err)

	aMapping := make(map[string]int64)
	for _, answerInfo := range answerInfos {
		aMapping[answerInfo.AnswerTag] = answerInfo.AnswerID
	}

	// lets get answer setup: answer one of the selected answers,
	// and for the rest of them select other and specify the answer text
	requestData := &apiservice.IntakeData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		Questions: []*apiservice.QuestionAnswerItem{
			{
				QuestionID: qMapping[prevAcnePrescriptionsSelectTag],
				AnswerIntakes: []*apiservice.AnswerItem{
					{
						PotentialAnswerID: aMapping[benzaclinTag],
						SubQuestions: []*apiservice.QuestionAnswerItem{
							{
								QuestionID: qMapping[usingPrevAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[usingPrevPrescriptionYes],
									},
								},
							},
							{
								QuestionID: qMapping[irritateSkinAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[irritateYesPrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[usingMoreThreeMonthsPrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[lessThanThreeMonthsPrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[howEffectiveAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[somewhatEffectivePrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[anythingElseAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Testing this answer",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerID: aMapping[otherPrevPrescriptionTag],
						AnswerText:        "Duac",
						SubQuestions: []*apiservice.QuestionAnswerItem{
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[usingPrevAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[usingPrevPrescriptionYes],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[irritateSkinAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[irritateYesPrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[usingMoreThreeMonthsPrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[lessThanThreeMonthsPrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[howEffectiveAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[somewhatEffectivePrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[anythingElseAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Testing this answer",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerID: aMapping[otherPrevPrescriptionTag],
						AnswerText:        "Duac2",
						SubQuestions: []*apiservice.QuestionAnswerItem{
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[usingPrevAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[usingPrevPrescriptionYes],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[irritateSkinAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[irritateYesPrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[usingMoreThreeMonthsPrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[lessThanThreeMonthsPrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[howEffectiveAcnePrescriptionTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[somewhatEffectivePrescription],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[anythingElseAcnePrescriptionTag],
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

	test_integration.SubmitAnswersIntakeForPatient(patientID, patient.AccountID.Int64(), requestData, testData, t)

	// now attempt to get the patient visit response to ensure that the answers for the top leve question exists
	// in the structure expected
	patientVisitResponse = test_integration.GetPatientVisitForPatient(patientID, testData, t)
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
						if answer.PotentialAnswerID.Int64() == 0 {
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
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.ID
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	questionInfos, err := testData.DataAPI.GetQuestionInfoForTags([]string{acnePrevOTCSelectTag, acneOTCProductTriedTag, acneUsingPrevOTCTag, acneHowEffectivePrevOTCTag, acneIrritateSkinPrevOTCTag, acneAnythingElsePrevOTCTag}, api.LanguageIDEnglish)
	test.OK(t, err)

	qMapping := make(map[string]int64)
	for _, questionInfo := range questionInfos {
		qMapping[questionInfo.QuestionTag] = questionInfo.QuestionID
	}

	answerInfos, err := testData.DataAPI.GetAnswerInfoForTags([]string{acneFreeTag, cetaphilOTCTag, otherPrevOTCTag, acneUsingPrevOTCYesTag, acneIrritateSkinYesPrevOTCTag, acneHowEffectiveVeryOTCTag}, api.LanguageIDEnglish)
	test.OK(t, err)

	aMapping := make(map[string]int64)
	for _, answerInfo := range answerInfos {
		aMapping[answerInfo.AnswerTag] = answerInfo.AnswerID
	}

	// lets get answer setup: answer one of the selected answers,
	// and for the rest of them select other and specify the answer text
	requestData := &apiservice.IntakeData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		Questions: []*apiservice.QuestionAnswerItem{
			&apiservice.QuestionAnswerItem{
				QuestionID: qMapping[acnePrevOTCSelectTag],
				AnswerIntakes: []*apiservice.AnswerItem{
					&apiservice.AnswerItem{
						PotentialAnswerID: aMapping[acneFreeTag],
						SubQuestions: []*apiservice.QuestionAnswerItem{
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Clean and clear",
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerID: aMapping[cetaphilOTCTag],
						SubQuestions: []*apiservice.QuestionAnswerItem{
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Clean lgkna",
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										PotentialAnswerID: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							&apiservice.QuestionAnswerItem{
								QuestionID: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									&apiservice.AnswerItem{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerID: aMapping[otherPrevOTCTag],
						SubQuestions: []*apiservice.QuestionAnswerItem{
							{
								QuestionID: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										AnswerText: "Clean and clear",
									},
								},
							},
							{
								QuestionID: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							{
								QuestionID: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							{
								QuestionID: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							{
								QuestionID: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										AnswerText: "Nope nothing else",
									},
								},
							},
						},
					},
					&apiservice.AnswerItem{
						PotentialAnswerID: aMapping[otherPrevOTCTag],
						SubQuestions: []*apiservice.QuestionAnswerItem{
							{
								QuestionID: qMapping[acneOTCProductTriedTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										AnswerText: "Clean and clear",
									},
								},
							},
							{
								QuestionID: qMapping[acneUsingPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[acneUsingPrevOTCYesTag],
									},
								},
							},
							{
								QuestionID: qMapping[acneHowEffectivePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[acneHowEffectiveVeryOTCTag],
									},
								},
							},
							{
								QuestionID: qMapping[acneIrritateSkinPrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
										PotentialAnswerID: aMapping[acneIrritateSkinYesPrevOTCTag],
									},
								},
							},
							{
								QuestionID: qMapping[acneAnythingElsePrevOTCTag],
								AnswerIntakes: []*apiservice.AnswerItem{
									{
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

	test_integration.SubmitAnswersIntakeForPatient(patientID, patient.AccountID.Int64(), requestData, testData, t)

	// now attempt to get the patient visit response to ensure that the answers for the top leve question exists
	// in the structure expected
	patientVisitResponse = test_integration.GetPatientVisitForPatient(patientID, testData, t)
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionTag == acnePrevOTCSelectTag {

					if len(question.Answers) != 4 {
						t.Fatalf("Expected 4 answers but got %d", len(question.Answers))
					}

					// there should be a potential answer id for each of the top level answers
					for _, answer := range test_integration.GetAnswerIntakesFromAnswers(question.Answers, t) {
						if answer.PotentialAnswerID.Int64() == 0 {
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

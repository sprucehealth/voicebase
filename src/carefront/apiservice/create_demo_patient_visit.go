package apiservice

import (
	"bytes"
	"carefront/api"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type CreateDemoPatientVisitHandler struct {
	Environment string
	DataApi     api.DataAPI
}

type questionTag int

const (
	qAcneOnset questionTag = iota
	qAcneWorse
	qAcneChangesWorse
	qAcneSymptoms
	qAcneWorsePeriod
	qSkinDescription
	qAcnePrevTreatmentTypes
	qAcnePrevTreatmentList
	qUsingTreatment
	qEffectiveTreatment
	qTreatmentIrritateSkin
	qLengthTreatment
	qAnythingElseAcne
	qAcneLocation
	qPregnancyPlanning
	qCurrentMedications
	qCurrentMedicationsEntry
	qLengthCurrentMedication
	qAllergicMedications
	qPrevSkinConditionDiagnosis
	qListPrevSkinConditionDiagnosis
	qOtherConditionsAcne
)

var (
	questionTags = map[string]questionTag{
		"q_onset_acne":                         qAcneOnset,
		"q_acne_worse":                         qAcneWorse,
		"q_changes_acne_worse":                 qAcneChangesWorse,
		"q_acne_symptoms":                      qAcneSymptoms,
		"q_acne_worse_period":                  qAcneWorsePeriod,
		"q_skin_description":                   qSkinDescription,
		"q_acne_prev_treatment_types":          qAcnePrevTreatmentTypes,
		"q_acne_prev_treatment_list":           qAcnePrevTreatmentList,
		"q_using_treatment":                    qUsingTreatment,
		"q_effective_treatment":                qEffectiveTreatment,
		"q_treatment_irritate_skin":            qTreatmentIrritateSkin,
		"q_length_treatment":                   qLengthTreatment,
		"q_anything_else_acne":                 qAnythingElseAcne,
		"q_acne_location":                      qAcneLocation,
		"q_pregnancy_planning":                 qPregnancyPlanning,
		"q_current_medications":                qCurrentMedications,
		"q_current_medications_entry":          qCurrentMedicationsEntry,
		"q_length_current_medication":          qLengthCurrentMedication,
		"q_allergic_medications":               qAllergicMedications,
		"q_prev_skin_condition_diagnosis":      qPrevSkinConditionDiagnosis,
		"q_list_prev_skin_condition_diagnosis": qListPrevSkinConditionDiagnosis,
		"q_other_conditions_acne":              qOtherConditionsAcne,
	}
)

type potentialAnswerTag int

const (
	aSixToTwelveMonths potentialAnswerTag = iota
	aAcneWorseYes
	aDiscoloration
	aScarring
	aPainfulToTouch
	aCysts
	aAcneWorsePeriodNo
	aSkinDescriptionOily
	aPrevTreatmentsTypeOTC
	aUsingTreatmentYes
	aSomewhatEffectiveTreatment
	aIrritateSkinYes
	aLengthTreatmentLessThanMonth
	aAcneLocationChest
	aAcneLocationNeck
	aAcneLocationFace
	aCurrentlyPregnant
	aCurrentMedicationsYes
	aTwoToFiveMonthsLength
	aAllergicMedicationsNo
	aPrevSkinConditionDiagnosisYes
	aListPrevSkinConditionDiagnosisAcne
	aListPrevSkinConditionDiagnosisPsoriasis
	aNoneOfTheAboveOtherConditions
)

var (
	answerTags = map[string]potentialAnswerTag{
		"a_six_twelve_months_ago":                     aSixToTwelveMonths,
		"a_yes_acne_worse":                            aAcneWorseYes,
		"a_discoloration":                             aDiscoloration,
		"a_scarring":                                  aScarring,
		"a_painful_touch":                             aPainfulToTouch,
		"a_cysts":                                     aCysts,
		"a_acne_worse_no":                             aAcneWorsePeriodNo,
		"a_oil_skin":                                  aSkinDescriptionOily,
		"a_otc_prev_treatment_type":                   aPrevTreatmentsTypeOTC,
		"a_using_treatment_yes":                       aUsingTreatmentYes,
		"a_effective_treatment_somewhat":              aSomewhatEffectiveTreatment,
		"a_irritate_skin_yes":                         aIrritateSkinYes,
		"a_length_treatment_less_one":                 aLengthTreatmentLessThanMonth,
		"a_chest_acne_location":                       aAcneLocationChest,
		"a_neck_acne_location":                        aAcneLocationNeck,
		"a_face_acne_location":                        aAcneLocationFace,
		"a_pregnant":                                  aCurrentlyPregnant,
		"a_current_medications_yes":                   aCurrentMedicationsYes,
		"a_length_current_medication_two_five_months": aTwoToFiveMonthsLength,
		"a_na_allergic_medications":                   aAllergicMedicationsNo,
		"a_yes_prev_skin_diagnosis":                   aPrevSkinConditionDiagnosisYes,
		"a_acne_skin_diagnosis":                       aListPrevSkinConditionDiagnosisAcne,
		"a_psoriasis_skin_diagnosis":                  aListPrevSkinConditionDiagnosisPsoriasis,
		"a_other_condition_acne_none":                 aNoneOfTheAboveOtherConditions,
	}
)

var (
// questionIds = map[questionTag]int64{
// 	qAcneOnset:                      28,
// 	qAcneWorse:                      4,
// 	qAcneChangesWorse:               6,
// 	qAcneSymptoms:                   29,
// 	qAcneWorsePeriod:                30,
// 	qSkinDescription:                32,
// 	qAcnePrevTreatmentTypes:         7,
// 	qAcnePrevTreatmentList:          8,
// 	qUsingTreatment:                 25,
// 	qEffectiveTreatment:             24,
// 	qTreatmentIrritateSkin:          30,
// 	qLengthTreatment:                26,
// 	qAnythingElseAcne:               9,
// 	qAcneLocation:                   18,
// 	qPregnancyPlanning:              10,
// 	qCurrentMedications:             46,
// 	qCurrentMedicationsEntry:        13,
// 	qLengthCurrentMedication:        41,
// 	qAllergicMedications:            11,
// 	qPrevSkinConditionDiagnosis:     15,
// 	qListPrevSkinConditionDiagnosis: 17,
// 	qOtherConditionsAcne:            34,
// }

// answerIds = map[answerTag]int64{
// 	aSixToTwelveMonths:                       142,
// 	aAcneWorseYes:                            8,
// 	aDiscoloration:                           86,
// 	aScarring:                                85,
// 	aPainfulToTouch:                          84,
// 	aCysts:                                   116,
// 	aAcneWorsePeriodNo:                       88,
// 	aSkinDescriptionOily:                     92,
// 	aPrevTreatmentsTypeOTC:                   13,
// 	aUsingTreatmentYes:                       73,
// 	aSomewhatEffectiveTreatment:              71,
// 	aIrritateSkinYes:                         118,
// 	aLengthTreatmentLessThanMonth:            76,
// 	aAcneLocationChest:                       60,
// 	aAcneLocationFace:                        59,
// 	aAcneLocationNeck:                        132,
// 	aCurrentlyPregnant:                       120,
// 	aCurrentMedicationsYes:                   143,
// 	aTwoToFiveMonthsLength:                   125,
// 	aAllergicMedicationsNo:                   21,
// 	aPrevSkinConditionDiagnosisYes:           27,
// 	aListPrevSkinConditionDiagnosisAcne:      30,
// 	aListPrevSkinConditionDiagnosisPsoriasis: 32,
// 	aNoneOfTheAboveOtherConditions:           130,
// }

)

func populatePatientIntake(questionIds map[questionTag]int64, answerIds map[potentialAnswerTag]int64) []*AnswerToQuestionItem {

	return []*AnswerToQuestionItem{
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcneOnset],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aSixToTwelveMonths],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcneWorse],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aAcneWorseYes],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcneChangesWorse],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					AnswerText: "This is a demo.",
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcneSymptoms],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aDiscoloration],
				},
				&AnswerItem{
					PotentialAnswerId: answerIds[aScarring],
				},
				&AnswerItem{
					PotentialAnswerId: answerIds[aCysts],
				},
				&AnswerItem{
					PotentialAnswerId: answerIds[aPainfulToTouch],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcneWorsePeriod],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aAcneWorsePeriodNo],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qSkinDescription],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aSkinDescriptionOily],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcnePrevTreatmentTypes],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aPrevTreatmentsTypeOTC],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcnePrevTreatmentList],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					AnswerText: "Proactiv",
					SubQuestionAnswerIntakes: []*SubQuestionAnswerIntake{
						&SubQuestionAnswerIntake{
							QuestionId: questionIds[qUsingTreatment],
							AnswerIntakes: []*AnswerItem{
								&AnswerItem{
									PotentialAnswerId: answerIds[aUsingTreatmentYes],
								},
							},
						},
						&SubQuestionAnswerIntake{
							QuestionId: questionIds[qEffectiveTreatment],
							AnswerIntakes: []*AnswerItem{
								&AnswerItem{
									PotentialAnswerId: answerIds[aSomewhatEffectiveTreatment],
								},
							},
						},
						&SubQuestionAnswerIntake{
							QuestionId: questionIds[qTreatmentIrritateSkin],
							AnswerIntakes: []*AnswerItem{
								&AnswerItem{
									PotentialAnswerId: answerIds[aIrritateSkinYes],
								},
							},
						},
						&SubQuestionAnswerIntake{
							QuestionId: questionIds[qLengthTreatment],
							AnswerIntakes: []*AnswerItem{
								&AnswerItem{
									PotentialAnswerId: answerIds[aLengthTreatmentLessThanMonth],
								},
							},
						},
					},
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAnythingElseAcne],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					AnswerText: "This is a demo. This is where patient will enter anything they'd like to share with us",
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAcneLocation],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aAcneLocationChest],
				},
				&AnswerItem{
					PotentialAnswerId: answerIds[aAcneLocationFace],
				},
				&AnswerItem{
					PotentialAnswerId: answerIds[aAcneLocationNeck],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qPregnancyPlanning],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aCurrentlyPregnant],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qCurrentMedications],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aCurrentMedicationsYes],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qCurrentMedicationsEntry],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					AnswerText: "Clyndamycin",
					SubQuestionAnswerIntakes: []*SubQuestionAnswerIntake{
						&SubQuestionAnswerIntake{
							QuestionId: questionIds[qLengthCurrentMedication],
							AnswerIntakes: []*AnswerItem{
								&AnswerItem{
									PotentialAnswerId: answerIds[aTwoToFiveMonthsLength],
								},
							},
						},
					},
				},
				&AnswerItem{
					AnswerText: "Tretinoin Topical",
					SubQuestionAnswerIntakes: []*SubQuestionAnswerIntake{
						&SubQuestionAnswerIntake{
							QuestionId: questionIds[qLengthCurrentMedication],
							AnswerIntakes: []*AnswerItem{
								&AnswerItem{
									PotentialAnswerId: answerIds[aTwoToFiveMonthsLength],
								},
							},
						},
					},
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qAllergicMedications],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aAllergicMedicationsNo],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qPrevSkinConditionDiagnosis],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aPrevSkinConditionDiagnosisYes],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qListPrevSkinConditionDiagnosis],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aListPrevSkinConditionDiagnosisAcne],
				},
				&AnswerItem{
					PotentialAnswerId: answerIds[aListPrevSkinConditionDiagnosisPsoriasis],
				},
			},
		},
		&AnswerToQuestionItem{
			QuestionId: questionIds[qOtherConditionsAcne],
			AnswerIntakes: []*AnswerItem{
				&AnswerItem{
					PotentialAnswerId: answerIds[aNoneOfTheAboveOtherConditions],
				},
			},
		},
	}
}

func (c *CreateDemoPatientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// ensure that this is the doctor that we are dealing with
	doctorId, err := c.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get doctor based on the account id: "+err.Error())
		return
	}

	// ensure that are not working with a non-prod environment
	if c.Environment == "prod" {
		WriteDeveloperError(w, http.StatusBadRequest, "Cannot work in the production environment")
		return
	}

	// ensure that the doctor is on the demo whitelist
	if !(c.DataApi.IsDoctorOnDemoWhitelist(doctorId)) {
		WriteUserError(w, http.StatusBadRequest, "Cannot create demo visit for doctor that is not on demo account")
		return
	}

	// Create a demo user account
	urlValues := url.Values{}
	urlValues.Set("first_name", "Demo")
	urlValues.Set("last_name", "User")
	urlValues.Set("dob", "11/08/1987")
	urlValues.Set("gender", "female")
	urlValues.Set("zip_code", "94115")
	urlValues.Set("phone", "2068773590")
	urlValues.Set("password", "12345")
	urlValues.Set("email", fmt.Sprintf("%d%d@example.com", time.Now().UnixNano(), doctorId))
	urlValues.Set("doctor_id", fmt.Sprintf("%d", doctorId))
	httpClient := http.Client{}
	signupPatientRequest, err := http.NewRequest("POST", "http://localhost:8080/v1/patient", bytes.NewBufferString(urlValues.Encode()))
	signupPatientRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(signupPatientRequest)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to signup random patient: "+err.Error())
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse body of response: "+err.Error())
		return
	}

	signupResponse := &PatientSignedupResponse{}
	err = json.Unmarshal(body, signupResponse)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to unmarshal response body into object: "+err.Error())
		return
	}

	// TODO Pick Pharmacy for patient

	// create patient visit
	createPatientVisitRequest, err := http.NewRequest("POST", "http://localhost:8080/v1/visit", nil)
	createPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
	resp, err = httpClient.Do(createPatientVisitRequest)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new patient visit: "+err.Error())
		return
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to read response")
	}

	patientVisitResponse := &PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to unmarshal response into patient visit response: "+err.Error())
		return
	}

	// answer questions

	// populate questionIds for the questionTags
	questionIds := make(map[questionTag]int64)
	questionTagsForLookup := make([]string, 0)
	for questionTagString, _ := range questionTags {
		questionTagsForLookup = append(questionTagsForLookup, questionTagString)
	}

	questionInfos, err := c.DataApi.GetQuestionInfoForTags(questionTagsForLookup, api.EN_LANGUAGE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to lookup ids based on question tags: "+err.Error())
		return
	}

	for _, questionInfoItem := range questionInfos {
		questionIds[questionTags[questionInfoItem.QuestionTag]] = questionInfoItem.Id
	}

	answerIds := make(map[potentialAnswerTag]int64)
	answerTagsForLookup := make([]string, 0)
	for answerTagString, _ := range answerTags {
		answerTagsForLookup = append(answerTagsForLookup, answerTagString)
	}

	answerInfos, err := c.DataApi.GetAnswerInfoForTags(answerTagsForLookup, api.EN_LANGUAGE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to lookup answer infos based on tags: "+err.Error())
		return
	}

	for _, answerInfoItem := range answerInfos {
		answerIds[answerTags[answerInfoItem.AnswerTag]] = answerInfoItem.PotentialAnswerId
	}

	answersToQuestions := populatePatientIntake(questionIds, answerIds)

	answerIntakeRequestBody := &AnswerIntakeRequestBody{
		PatientVisitId: patientVisitResponse.PatientVisitId,
		Questions:      answersToQuestions,
	}
	jsonData, err := json.Marshal(answerIntakeRequestBody)
	answerQuestionsRequest, err := http.NewRequest("POST", "http://localhost:8080/v1/answer", bytes.NewBuffer(jsonData))
	answerQuestionsRequest.Header.Set("Content-Type", "application/json")
	answerQuestionsRequest.Header.Set("Authorization", "token "+signupResponse.Token)

	_, err = httpClient.Do(answerQuestionsRequest)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store answers for patient in patient visit: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

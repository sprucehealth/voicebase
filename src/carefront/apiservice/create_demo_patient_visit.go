package apiservice

import (
	"bytes"
	"carefront/api"
	"carefront/libs/pharmacy"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type CreateDemoPatientVisitHandler struct {
	Environment     string
	DataApi         api.DataAPI
	CloudStorageApi api.CloudStorageAPI
	AWSRegion       string
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
	qFacePhotoIntake
	qNeckPhotoIntake
	qChestPhotoIntake
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
		"q_face_photo_intake":                  qFacePhotoIntake,
		"q_neck_photo_intake":                  qNeckPhotoIntake,
		"q_chest_photo_intake":                 qChestPhotoIntake,
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
	aFaceFrontPhotoIntake
	aProfileRightPhotoIntake
	aProfileLeftPhotoIntake
	aChestPhotoIntake
	aNeckPhotoIntake
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
		"a_face_front_phota_intake":                   aFaceFrontPhotoIntake,
		"a_face_right_phota_intake":                   aProfileRightPhotoIntake,
		"a_face_left_phota_intake":                    aProfileLeftPhotoIntake,
		"a_chest_phota_intake":                        aChestPhotoIntake,
		"a_neck_photo_intake":                         aNeckPhotoIntake,
	}
)

const (
	signupPatientUrl         = "http://127.0.0.1:8080/v1/patient"
	updatePatientPharmacyUrl = "http://127.0.0.1:8080/v1/patient/pharmacy"
	patientVisitUrl          = "http://127.0.0.1:8080/v1/visit"
	answerQuestionsUrl       = "http://127.0.0.1:8080/v1/answer"
	photoIntakeUrl           = "http://127.0.0.1:8080/v1/answer/photo"
	DemoPhotosBucketFormat   = "%s-carefront-demo"
	frontPhoto               = "profile_front.jpg"
	profileRightPhoto        = "profile_right.jpg"
	profileLeftPhoto         = "profile_left.jpg"
	neckPhoto                = "neck.jpg"
	chestPhoto               = "chest.jpg"
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

func startPatientIntakeSubmission(answersToQuestions []*AnswerToQuestionItem, patientVisitId int64, patientAuthToken string, signal chan int) {

	go func() {

		answerIntakeRequestBody := &AnswerIntakeRequestBody{
			PatientVisitId: patientVisitId,
			Questions:      answersToQuestions,
		}

		jsonData, _ := json.Marshal(answerIntakeRequestBody)
		answerQuestionsRequest, err := http.NewRequest("POST", answerQuestionsUrl, bytes.NewBuffer(jsonData))
		answerQuestionsRequest.Header.Set("Content-Type", "application/json")
		answerQuestionsRequest.Header.Set("Authorization", "token "+patientAuthToken)

		httpClient := http.Client{}
		resp, err := httpClient.Do(answerQuestionsRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			signal <- 0
			return
			//return fmt.Errorf("Unable to store answers for patient in patient visit: " + err.Error())
		}
		signal <- 1
	}()
}

func (c *CreateDemoPatientVisitHandler) startPhotoSubmissionForPatient(questionId, answerId, patientVisitId int64, photoKey, patientAuthToken string, signal chan int) {

	go func() {
		// get the image
		imageData, _, err := c.CloudStorageApi.GetObjectAtLocation(fmt.Sprintf(DemoPhotosBucketFormat, c.Environment), photoKey, c.AWSRegion)
		if err != nil {
			signal <- 0
			return
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// uploading any file as a photo for now
		part, err := writer.CreateFormFile("photo", photoKey)
		if err != nil {
			signal <- 0
			return
			//return fmt.Errorf("Unable to create a form file with a sample file")
		}

		_, err = io.Copy(part, bytes.NewBuffer(imageData))
		if err != nil {
			signal <- 0
			return
			//return fmt.Errorf("Unable to copy contents of file into multipart form data: " + err.Error())
		}

		writer.WriteField("question_id", strconv.FormatInt(questionId, 10))
		writer.WriteField("potential_answer_id", strconv.FormatInt(answerId, 10))
		writer.WriteField("patient_visit_id", strconv.FormatInt(patientVisitId, 10))

		err = writer.Close()
		if err != nil {
			signal <- 0
			return
			//return fmt.Errorf("Unable to create multi-form data. Error when trying to close writer: " + err.Error())
		}

		photoIntakeRequest, err := http.NewRequest("POST", photoIntakeUrl, body)
		photoIntakeRequest.Header.Set("Content-Type", writer.FormDataContentType())
		photoIntakeRequest.Header.Set("Authorization", "token "+patientAuthToken)
		httpClient := http.Client{}
		resp, err := httpClient.Do(photoIntakeRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			signal <- 0
			return
			//return fmt.Errorf("Unable to store photo for patient in patient visit", err.Error())
		}
		signal <- 1
	}()
}

func (c *CreateDemoPatientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

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

	// ********** CREATE RANDOM PATIENT **********
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
	signupPatientRequest, err := http.NewRequest("POST", signupPatientUrl, bytes.NewBufferString(urlValues.Encode()))
	signupPatientRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(signupPatientRequest)
	if err != nil || resp.StatusCode != http.StatusOK {
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

	// ********** ASSIGN PHARMACY TO PATIENT **********

	pharmacyDetails := &pharmacy.PharmacyData{
		Id:      "CoQBdgAAAIU6I2DXvwyyql2HTtAdaMrZ_AEgvKsD1O_V4mePQw3NNgntSwDlCKoCdd47DZdZbPOMEXMWSPyno1qekMr0A0ghV2rWGpVbVjLeM-ehKZH1gxMtTVlon47ktbVi2uUKCyuzpZh5hI7gjQChUPkkGoxnpKoLeAcCnzEeC5m4YGRFEhALIHQkJ_E13vByzK_t9xjlGhSDLIpV9QxTHgTwoESfAKHkMIzuxQ",
		Address: "116 New Montgomery St",
		Name:    "Walgreens Pharmacies",
		City:    "San Francisco",
		State:   "CA",
		Source:  pharmacy.PHARMACY_SOURCE_GOOGLE,
	}

	jsonData, err := json.Marshal(pharmacyDetails)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to marshal pharmacy details")
	}

	updatePatientPharmacyRequest, err := http.NewRequest("POST", updatePatientPharmacyUrl, bytes.NewBuffer(jsonData))
	updatePatientPharmacyRequest.Header.Set("Content-Type", "application/json")
	updatePatientPharmacyRequest.Header.Set("Authorization", "token "+signupResponse.Token)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new http request: "+err.Error())
		return
	}

	_, err = httpClient.Do(updatePatientPharmacyRequest)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update pharmacy for patient: "+err.Error())
		return
	}

	// ********** CREATE PATIENT VISIT **********

	// create patient visit
	createPatientVisitRequest, err := http.NewRequest("POST", patientVisitUrl, nil)
	createPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
	resp, err = httpClient.Do(createPatientVisitRequest)
	if err != nil || resp.StatusCode != http.StatusOK {
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

	// ********** SIMULATE PATIENT INTAKE **********

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

	// use a buffered channel so that the goroutines don't block
	// until the receiver reads off the channel
	signal := make(chan int, 6)
	numRequestsWaitingFor := 6

	startPatientIntakeSubmission(answersToQuestions, patientVisitResponse.PatientVisitId, signupResponse.Token, signal)

	c.startPhotoSubmissionForPatient(questionIds[qFacePhotoIntake],
		answerIds[aFaceFrontPhotoIntake], patientVisitResponse.PatientVisitId, frontPhoto, signupResponse.Token, signal)

	c.startPhotoSubmissionForPatient(questionIds[qFacePhotoIntake],
		answerIds[aProfileRightPhotoIntake], patientVisitResponse.PatientVisitId, profileRightPhoto, signupResponse.Token, signal)

	c.startPhotoSubmissionForPatient(questionIds[qFacePhotoIntake],
		answerIds[aProfileLeftPhotoIntake], patientVisitResponse.PatientVisitId, profileLeftPhoto, signupResponse.Token, signal)

	c.startPhotoSubmissionForPatient(questionIds[qNeckPhotoIntake],
		answerIds[aNeckPhotoIntake], patientVisitResponse.PatientVisitId, neckPhoto, signupResponse.Token, signal)

	c.startPhotoSubmissionForPatient(questionIds[qChestPhotoIntake],
		answerIds[aChestPhotoIntake], patientVisitResponse.PatientVisitId, chestPhoto, signupResponse.Token, signal)

	// wait for all requests to finish
	for numRequestsWaitingFor > 0 {
		result := <-signal
		if result == 0 {
			WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong when tryign to submit patient visit intake. Patient visit not successfully submitted")
			return
		}
		numRequestsWaitingFor--
	}

	// ********** SUBMIT CASE TO DOCTOR **********
	submitPatientVisitRequest, err := http.NewRequest("PUT", patientVisitUrl, bytes.NewBufferString(fmt.Sprintf("patient_visit_id=%d", patientVisitResponse.PatientVisitId)))
	submitPatientVisitRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	submitPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new request to submit patient visit: "+err.Error())
		return
	}

	resp, err = httpClient.Do(submitPatientVisitRequest)
	if err != nil || resp.StatusCode != http.StatusOK {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to make successful request to submit patient visit")
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

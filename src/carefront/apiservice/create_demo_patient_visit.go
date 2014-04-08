package apiservice

import (
	"bytes"
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/golog"
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

	"github.com/gorilla/schema"
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
	demoPhotosBucketFormat   = "%s-carefront-demo"
	frontPhoto               = "profile_front.jpg"
	profileRightPhoto        = "profile_right.jpg"
	profileLeftPhoto         = "profile_left.jpg"
	neckPhoto                = "neck.jpg"
	chestPhoto               = "chest.jpg"
	failure                  = 0
	success                  = 1
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
		answerQuestionsRequest, err := http.NewRequest("POST", answerQuestionsUrl, bytes.NewReader(jsonData))
		answerQuestionsRequest.Header.Set("Content-Type", "application/json")
		answerQuestionsRequest.Header.Set("Authorization", "token "+patientAuthToken)

		resp, err := http.DefaultClient.Do(answerQuestionsRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Error while submitting patient intake: %+v", err)
			signal <- failure
			return
		}
		signal <- success
	}()
}

func (c *CreateDemoPatientVisitHandler) startPhotoSubmissionForPatient(questionId, answerId, patientVisitId int64, photoKey, patientAuthToken string, signal chan int) {

	go func() {
		// get the image
		imageData, _, err := c.CloudStorageApi.GetObjectAtLocation(fmt.Sprintf(demoPhotosBucketFormat, c.Environment), photoKey, c.AWSRegion)
		if err != nil {
			golog.Errorf("Error while getting picture at location: %+v", err)
			signal <- failure
			return
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// uploading any file as a photo for now
		part, err := writer.CreateFormFile("photo", photoKey)
		if err != nil {
			golog.Errorf("Error while trying to create form file for photo submission: %+v", err)
			signal <- failure
			return
		}

		_, err = io.Copy(part, bytes.NewReader(imageData))
		if err != nil {
			golog.Errorf("Error while trying to copy image data: %+v", err)
			signal <- failure
			return
		}

		writer.WriteField("question_id", strconv.FormatInt(questionId, 10))
		writer.WriteField("potential_answer_id", strconv.FormatInt(answerId, 10))
		writer.WriteField("patient_visit_id", strconv.FormatInt(patientVisitId, 10))

		err = writer.Close()
		if err != nil {
			golog.Errorf("Error while trying to create form data for submission: %+v", err)
			signal <- failure
			return
		}

		photoIntakeRequest, err := http.NewRequest("POST", photoIntakeUrl, body)
		photoIntakeRequest.Header.Set("Content-Type", writer.FormDataContentType())
		photoIntakeRequest.Header.Set("Authorization", "token "+patientAuthToken)
		resp, err := http.DefaultClient.Do(photoIntakeRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Error while trying submit photo for intake: %+v", err)
			signal <- failure
			return
		}
		signal <- success
	}()
}

type CreateDemoPatientVisitRequestData struct {
	ToCreateSurescriptsPatients bool `schema:"surescripts"`
}

func (c *CreateDemoPatientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &CreateDemoPatientVisitRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
		return
	}

	if requestData.ToCreateSurescriptsPatients {
		ciLi := common.Patient{
			FirstName: "Ci",
			LastName:  "Li",
			Gender:    "Male",
			Dob: encoding.Dob{
				Year:  1923,
				Month: 10,
				Day:   18,
			},
			ZipCode: "94115",
			PhoneNumbers: []*common.PhoneInformation{&common.PhoneInformation{
				Phone:     "2068773590",
				PhoneType: "Home",
			},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "12345 Main Street",
				AddressLine2: "Apt 1112",
				City:         "San Francisco",
				State:        "California",
				ZipCode:      "94115",
			},
		}

		howardPlower := common.Patient{
			Prefix:    "Mr",
			FirstName: "Howard",
			LastName:  "Plower",
			Gender:    "Male",
			Dob: encoding.Dob{
				Year:  1923,
				Month: 10,
				Day:   18,
			},
			ZipCode: "19102",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "215-988-6728",
					PhoneType: "Home",
				},
				&common.PhoneInformation{
					Phone:     "4137762738",
					PhoneType: "Cell",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "76 Deerlake Road",
				City:         "Philadelphia",
				State:        "Pennsylvania",
				ZipCode:      "19102",
			},
		}

		karaWhiteside := common.Patient{
			FirstName: "Kara",
			LastName:  "Whiteside",
			Gender:    "Female",
			Dob: encoding.Dob{
				Year:  1952,
				Month: 10,
				Day:   11,
			},
			ZipCode: "44306",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "3305547754",
					PhoneType: "Home",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "23230 Seaport",
				City:         "Akron",
				State:        "Ohio",
				ZipCode:      "44306",
			},
		}

		debraTucker := common.Patient{
			Prefix:    "Ms",
			FirstName: "Debra",
			LastName:  "Tucker",
			Gender:    "Female",
			Dob: encoding.Dob{
				Year:  1980,
				Month: 11,
				Day:   01,
			},
			ZipCode: "44103",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "4408450398",
					PhoneType: "Home",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "8331 Everwood Dr.",
				AddressLine2: "Apt 342",
				City:         "Cleveland",
				State:        "Ohio",
				ZipCode:      "44103",
			},
		}

		feliciaFlounders := common.Patient{
			Prefix:     "Ms",
			FirstName:  "Felicia",
			LastName:   "Flounders",
			MiddleName: "Ann",
			Gender:     "Female",
			Dob: encoding.Dob{
				Year:  1980,
				Month: 11,
				Day:   01,
			},
			ZipCode: "20187",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "3108620035x2345",
					PhoneType: "Home",
				},
				&common.PhoneInformation{
					Phone:     "3019289283",
					PhoneType: "Cell",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "6715 Swanson Ave",
				AddressLine2: "Apt 102",
				City:         "Bethesda",
				State:        "Maryland",
				ZipCode:      "20187",
			},
		}

		douglasRichardson := common.Patient{
			FirstName:  "Douglas",
			LastName:   "Richardson",
			MiddleName: "R",
			Gender:     "Male",
			Dob: encoding.Dob{
				Year:  1994,
				Month: 9,
				Day:   29,
			},
			ZipCode: "01040",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "4137760938",
					PhoneType: "Home",
				},
				&common.PhoneInformation{
					Phone:     "4137762738",
					PhoneType: "Cell",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "23 Trumble Dr",
				AddressLine2: "Apt 101",
				City:         "Holyoke",
				State:        "Massachusetts",
				ZipCode:      "01040-2239",
			},
		}

		davidThrower := common.Patient{
			FirstName: "David",
			LastName:  "Thrower",
			Gender:    "Male",
			Dob: encoding.Dob{
				Year:  1933,
				Month: 2,
				Day:   22,
			},
			ZipCode: "34737",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "3526685547",
					PhoneType: "Home",
				},
				&common.PhoneInformation{
					Phone:     "4137762738",
					PhoneType: "Cell",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "64 Violet Lane",
				AddressLine2: "Apt 101",
				City:         "Hower In The Hills",
				State:        "Florida",
				ZipCode:      "34737",
			},
		}

		maxLengthPatient := common.Patient{
			Prefix:     "Patient II",
			FirstName:  "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
			LastName:   "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
			MiddleName: "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
			Suffix:     "Junior iii",
			Gender:     "Male",
			Dob: encoding.Dob{
				Year:  1948,
				Month: 1,
				Day:   1,
			},
			ZipCode: "34737",
			PhoneNumbers: []*common.PhoneInformation{
				&common.PhoneInformation{
					Phone:     "5719212122x1234567890444",
					PhoneType: "Home",
				},
				&common.PhoneInformation{
					Phone:     "7034445523x4473",
					PhoneType: "Cell",
				},
				&common.PhoneInformation{
					Phone:     "7034445524x4474",
					PhoneType: "Work",
				},
				&common.PhoneInformation{
					Phone:     "7034445522x4472",
					PhoneType: "Work",
				},
				&common.PhoneInformation{
					Phone:     "7034445526x4476",
					PhoneType: "Home",
				},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
				AddressLine2: "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
				City:         "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
				State:        "Colorado",
				ZipCode:      "94115",
			},
		}

		topLevelSignal := make(chan int, 8)
		c.createNewDemoPatient(&ciLi, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&howardPlower, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&karaWhiteside, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&debraTucker, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&feliciaFlounders, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&douglasRichardson, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&davidThrower, doctorId, topLevelSignal)
		time.Sleep(500 * time.Millisecond)
		c.createNewDemoPatient(&maxLengthPatient, doctorId, topLevelSignal)

		numberPatientsWaitingFor := 8
		for numberPatientsWaitingFor > 0 {
			result := <-topLevelSignal
			if result == failure {
				WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong while trying to create demo patient")
				return
			}
			numberPatientsWaitingFor--
		}

	} else {
		demoPatientToCreate := common.Patient{
			FirstName: "Demo",
			LastName:  "User",
			Gender:    "female",
			Dob: encoding.Dob{
				Year:  1987,
				Month: 11,
				Day:   8,
			},
			ZipCode: "94115",
			PhoneNumbers: []*common.PhoneInformation{&common.PhoneInformation{
				Phone:     "2068773590",
				PhoneType: "Home",
			},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     "47731",
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "12345 Main Street",
				AddressLine2: "Apt 1112",
				City:         "San Francisco",
				State:        "California",
				ZipCode:      "94115",
			},
		}
		topLevelSignal := make(chan int)
		c.createNewDemoPatient(&demoPatientToCreate, doctorId, topLevelSignal)
		result := <-topLevelSignal
		if result == failure {
			WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong while trying to create demo patient")
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

func (c *CreateDemoPatientVisitHandler) createNewDemoPatient(patient *common.Patient, doctorId int64, topLevelSignal chan int) {
	go func() {
		// ********** CREATE RANDOM PATIENT **********
		// Note that once this random patient is created, we will use the patientId and the accountId
		// to update the patient information. The reason to go through this flow instead of directly
		// adding the patient to the database is to avoid the work of assigning a care team to the patient
		// and setting a patient up with an account
		urlValues := url.Values{}
		urlValues.Set("first_name", patient.FirstName)
		urlValues.Set("last_name", patient.LastName)
		urlValues.Set("dob", patient.Dob.String())
		urlValues.Set("gender", patient.Gender)
		urlValues.Set("zip_code", patient.ZipCode)
		urlValues.Set("phone", patient.PhoneNumbers[0].Phone)
		urlValues.Set("password", "12345")
		urlValues.Set("email", fmt.Sprintf("%d%d@example.com", time.Now().UnixNano(), doctorId))
		urlValues.Set("doctor_id", fmt.Sprintf("%d", doctorId))
		signupPatientRequest, err := http.NewRequest("POST", signupPatientUrl, bytes.NewBufferString(urlValues.Encode()))
		signupPatientRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(signupPatientRequest)
		if err != nil {
			golog.Errorf("Unable to signup random patient:%+v", err)
			topLevelSignal <- failure
			return
		}

		if resp.StatusCode != http.StatusOK {
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				golog.Errorf("Unable to signup random patient and unable to read body of response: %+v", err)
				topLevelSignal <- failure
				return
			}
			golog.Errorf("Status %d when trying to signup random patient: %+v", resp.StatusCode, string(respBody))
			topLevelSignal <- failure
			return
		}

		signupResponse := &PatientSignedupResponse{}
		err = json.NewDecoder(resp.Body).Decode(&signupResponse)
		defer resp.Body.Close()

		if err != nil {
			golog.Errorf("Unable to unmarshal response body into object: %+v", err)
			topLevelSignal <- failure
			return
		}

		// ********** UPDATE PATIENT DEMOGRAPHIC INFORMATION AS THOUGH A DOCTOR WERE UPDATING IT **********
		patient.PatientId = signupResponse.Patient.PatientId
		patient.AccountId = signupResponse.Patient.AccountId
		patient.Email = signupResponse.Patient.Email
		err = c.DataApi.UpdatePatientInformationFromDoctor(patient)
		if err != nil {
			golog.Errorf("Unable to update patient information:%+v", err)
			topLevelSignal <- failure
			return
		}

		err = c.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patient.Pharmacy)
		if err != nil {
			golog.Errorf("Unable to update patients preferred pharmacy:%+v", err)
			topLevelSignal <- failure
			return
		}

		// ********** CREATE PATIENT VISIT **********

		// create patient visit
		createPatientVisitRequest, err := http.NewRequest("POST", patientVisitUrl, nil)
		createPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
		resp, err = http.DefaultClient.Do(createPatientVisitRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Unable to create new patient visit: %+v", err)
			topLevelSignal <- failure
			return
		}

		patientVisitResponse := &PatientVisitResponse{}
		err = json.NewDecoder(resp.Body).Decode(&patientVisitResponse)
		defer resp.Body.Close()
		if err != nil {
			golog.Errorf("Unable to unmarshal response into patient visit response: %+v", err.Error())
			topLevelSignal <- failure
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
			golog.Errorf("Unable to lookup ids based on question tags:%+v", err.Error())
			topLevelSignal <- failure
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
			golog.Errorf("Unable to lookup answer infos based on tags:%+v", err.Error())
			topLevelSignal <- failure
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
			if result == failure {
				golog.Errorf("Something went wrong when tryign to submit patient visit intake. Patient visit not successfully submitted")
				topLevelSignal <- failure
				return

			}
			numRequestsWaitingFor--
		}

		// ********** SUBMIT CASE TO DOCTOR **********
		submitPatientVisitRequest, err := http.NewRequest("PUT", patientVisitUrl, bytes.NewBufferString(fmt.Sprintf("patient_visit_id=%d", patientVisitResponse.PatientVisitId)))
		submitPatientVisitRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		submitPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
		if err != nil {
			golog.Errorf("Unable to create new request to submit patient visit:%+v", err)
			topLevelSignal <- failure
			return
		}

		resp, err = http.DefaultClient.Do(submitPatientVisitRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Unable to make successful request to submit patient visit")
			topLevelSignal <- failure
			return
		}

		topLevelSignal <- success
	}()

}

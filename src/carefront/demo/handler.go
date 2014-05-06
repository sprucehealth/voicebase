package demo

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/golog"
	"carefront/libs/pharmacy"
	patientApiService "carefront/patient"
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

type Handler struct {
	environment     string
	dataApi         api.DataAPI
	cloudStorageApi api.CloudStorageAPI
	awsRegion       string
}

func NewHandler(dataApi api.DataAPI, cloudStorageApi api.CloudStorageAPI, awsRegion, environment string) *Handler {
	return &Handler{
		environment:     environment,
		dataApi:         dataApi,
		cloudStorageApi: cloudStorageApi,
		awsRegion:       awsRegion,
	}
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

func populatePatientIntake(questionIds map[questionTag]int64, answerIds map[potentialAnswerTag]int64) []*apiservice.AnswerToQuestionItem {

	return []*apiservice.AnswerToQuestionItem{
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcneOnset],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aSixToTwelveMonths],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcneWorse],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aAcneWorseYes],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcneChangesWorse],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					AnswerText: "I've starting working out again so wonder if sweat could be a contributing factor?",
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcneSymptoms],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aDiscoloration],
				},
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aScarring],
				},
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aCysts],
				},
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aPainfulToTouch],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcneWorsePeriod],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aAcneWorsePeriodNo],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qSkinDescription],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aSkinDescriptionOily],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcnePrevTreatmentTypes],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aPrevTreatmentsTypeOTC],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcnePrevTreatmentList],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					AnswerText: "Proactiv",
					SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
						&apiservice.SubQuestionAnswerIntake{
							QuestionId: questionIds[qUsingTreatment],
							AnswerIntakes: []*apiservice.AnswerItem{
								&apiservice.AnswerItem{
									PotentialAnswerId: answerIds[aUsingTreatmentYes],
								},
							},
						},
						&apiservice.SubQuestionAnswerIntake{
							QuestionId: questionIds[qEffectiveTreatment],
							AnswerIntakes: []*apiservice.AnswerItem{
								&apiservice.AnswerItem{
									PotentialAnswerId: answerIds[aSomewhatEffectiveTreatment],
								},
							},
						},
						&apiservice.SubQuestionAnswerIntake{
							QuestionId: questionIds[qTreatmentIrritateSkin],
							AnswerIntakes: []*apiservice.AnswerItem{
								&apiservice.AnswerItem{
									PotentialAnswerId: answerIds[aIrritateSkinYes],
								},
							},
						},
						&apiservice.SubQuestionAnswerIntake{
							QuestionId: questionIds[qLengthTreatment],
							AnswerIntakes: []*apiservice.AnswerItem{
								&apiservice.AnswerItem{
									PotentialAnswerId: answerIds[aLengthTreatmentLessThanMonth],
								},
							},
						},
					},
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAnythingElseAcne],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					AnswerText: "I've noticed that my acne flares up when I wait longer between changing razor blades. Also, my acne typically concentrates around my lips.",
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAcneLocation],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aAcneLocationChest],
				},
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aAcneLocationFace],
				},
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aAcneLocationNeck],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qPregnancyPlanning],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aCurrentlyPregnant],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qCurrentMedications],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aCurrentMedicationsYes],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qCurrentMedicationsEntry],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					AnswerText: "Clyndamycin",
					SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
						&apiservice.SubQuestionAnswerIntake{
							QuestionId: questionIds[qLengthCurrentMedication],
							AnswerIntakes: []*apiservice.AnswerItem{
								&apiservice.AnswerItem{
									PotentialAnswerId: answerIds[aTwoToFiveMonthsLength],
								},
							},
						},
					},
				},
				&apiservice.AnswerItem{
					AnswerText: "Tretinoin Topical",
					SubQuestionAnswerIntakes: []*apiservice.SubQuestionAnswerIntake{
						&apiservice.SubQuestionAnswerIntake{
							QuestionId: questionIds[qLengthCurrentMedication],
							AnswerIntakes: []*apiservice.AnswerItem{
								&apiservice.AnswerItem{
									PotentialAnswerId: answerIds[aTwoToFiveMonthsLength],
								},
							},
						},
					},
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qAllergicMedications],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aAllergicMedicationsNo],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qPrevSkinConditionDiagnosis],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aPrevSkinConditionDiagnosisYes],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qListPrevSkinConditionDiagnosis],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aListPrevSkinConditionDiagnosisAcne],
				},
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aListPrevSkinConditionDiagnosisPsoriasis],
				},
			},
		},
		&apiservice.AnswerToQuestionItem{
			QuestionId: questionIds[qOtherConditionsAcne],
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: answerIds[aNoneOfTheAboveOtherConditions],
				},
			},
		},
	}
}

func startPatientIntakeSubmission(answersToQuestions []*apiservice.AnswerToQuestionItem, patientVisitId int64, patientAuthToken string, signal chan int) {

	go func() {

		answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{
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

func (c *Handler) startPhotoSubmissionForPatient(questionId, answerId, patientVisitId int64, photoKey, patientAuthToken string, signal chan int) {

	go func() {
		// get the image
		imageData, _, err := c.cloudStorageApi.GetObjectAtLocation(fmt.Sprintf(demoPhotosBucketFormat, c.environment), photoKey, c.awsRegion)
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

func (c *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	doctorId, err := c.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get doctor based on the account id: "+err.Error())
		return
	}

	// ensure that are not working with a non-prod environment
	if c.environment == "prod" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Cannot work in the production environment")
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &CreateDemoPatientVisitRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
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
					Phone:     "215-988-6723",
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
				Year:  1970,
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
				Year:  1968,
				Month: 9,
				Day:   1,
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
				AddressLine1: "2556 Lane Rd",
				AddressLine2: "Apt 101",
				City:         "Smittyville",
				State:        "Virginia",
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
				City:         "Howey In The Hills",
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
				apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong while trying to create demo patient")
				return
			}
			numberPatientsWaitingFor--
		}

	} else {
		demoPatientToCreate := common.Patient{
			FirstName: "Kunal",
			LastName:  "Jham",
			Gender:    "male",
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
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong while trying to create demo patient")
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (c *Handler) createNewDemoPatient(patient *common.Patient, doctorId int64, topLevelSignal chan int) {
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

		signupResponse := &patientApiService.PatientSignedupResponse{}
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
		err = c.dataApi.UpdatePatientInformation(patient, false)
		if err != nil {
			golog.Errorf("Unable to update patient information:%+v", err)
			topLevelSignal <- failure
			return
		}

		err = c.dataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patient.Pharmacy)
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

		patientVisitResponse := &apiservice.PatientVisitResponse{}
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

		questionInfos, err := c.dataApi.GetQuestionInfoForTags(questionTagsForLookup, api.EN_LANGUAGE_ID)
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
		answerInfos, err := c.dataApi.GetAnswerInfoForTags(answerTagsForLookup, api.EN_LANGUAGE_ID)
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

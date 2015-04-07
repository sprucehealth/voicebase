package test_intake

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/test/test_integration"
)

var (
	otherLocationPhotoSectionTag = "q_other_location_photo_section"
	facePhotoSectionTag          = "q_face_photo_section"
	chestPhotoSectionTag         = "q_chest_photo_section"
	backPhotoSectionTag          = "q_back_photo_section"
)

func TestPhotoIntake(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoID, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataAPI.GetQuestionInfo(otherLocationPhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	// ensure that the photos now exist for this question for the patient
	photoIntakeSections, err := testData.DataAPI.PatientPhotoSectionsForQuestionIDs([]int64{questionInfo.QuestionID}, patientID, patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if pIntakeSection, ok := photoIntakeSections[questionInfo.QuestionID][0].(*common.PhotoIntakeSection); !ok {
		t.Fatalf("Expected PhotoIntakeSection instead got type %T", photoIntakeSections[0])
	} else if len(pIntakeSection.Photos) != 1 {
		t.Fatalf("Expected 1 photo in the section instead got %d", len(pIntakeSection.Photos))
	} else if pIntakeSection.Name != "Testing" {
		t.Fatalf("Expected name %s for section instead got %s", "Testing", pIntakeSection.Name)
	} else if pIntakeSection.Photos[0].Name != "Other" {
		t.Fatalf("Expected name %s for photo slot in the answer instead got %s", "Other", pIntakeSection.Photos[0].Name)
	}
}

func TestPhotoIntake_AllSections(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoIds := make([]int64, 5)
	var err error
	for i := 0; i < 5; i++ {
		// Need a better way to generate a temp URL
		tempurl := "s3://us-east-1/test-spruce-storage/media/media-b" + strconv.Itoa(i)
		photoIds[i], err = testData.DataAPI.AddMedia(patient.PersonID, tempurl, "image/jpeg")
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	// get the question that represents the other location photo section
	questionInfos, err := testData.DataAPI.GetQuestionInfoForTags([]string{otherLocationPhotoSectionTag, facePhotoSectionTag, chestPhotoSectionTag, backPhotoSectionTag}, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: make([]*patient_visit.PhotoAnswerIntakeQuestionItem, 4),
	}

	for i, questionInfo := range questionInfos {
		// get the photo slots associated with this question
		photoSlots, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, api.LanguageIDEnglish)
		if err != nil {
			t.Fatal(err.Error())
		}

		requestData.PhotoQuestions[i] = &patient_visit.PhotoAnswerIntakeQuestionItem{
			QuestionID: questionInfo.QuestionID,
			PhotoSections: []*common.PhotoIntakeSection{
				&common.PhotoIntakeSection{
					Name: "Testing",
					Photos: []*common.PhotoIntakeSlot{
						&common.PhotoIntakeSlot{
							PhotoID: photoIds[i],
							SlotID:  photoSlots[0].ID,
							Name:    "Slot1",
						},
					},
				},
			},
		}
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	// now try to get the patient visit for this patient via the api to ensure that the photos are filled in
	patientVisitResponse = test_integration.GetPatientVisitForPatient(patientID, testData, t)

	// go through the visit intake layout to ensure that photos are present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionType == info_intake.QuestionTypePhotoSection {
					if len(question.Answers) != 1 {
						t.Fatalf("Expected question to have 1 answered section but instead it has %d", len(question.Answers))
					} else if photoIntakeSection, ok := question.Answers[0].(*common.PhotoIntakeSection); !ok {
						t.Fatalf("Expected answer of type PhotoIntakeSection instead got type %T", question.Answers[0])
					} else if len(photoIntakeSection.Photos) != 1 {
						t.Fatalf("Expected question to have 1 photo in the section but instead it has %d", len(photoIntakeSection.Photos))
					} else if photoIntakeSection.Photos[0].PhotoURL == "" {
						t.Fatalf("Expected photo url to exist instead it was empty")
					}
				}
			}
		}
	}
}

func TestPhotoIntake_MultipleSectionsForSameQuestion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoID, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}
	photoID2, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost/2", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataAPI.GetQuestionInfo(otherLocationPhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
					&common.PhotoIntakeSection{
						Name: "Testing2",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other2",
								PhotoID: photoID2,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	photoIntakeSections, err := testData.DataAPI.PatientPhotoSectionsForQuestionIDs([]int64{questionInfo.QuestionID}, patientID, patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections[questionInfo.QuestionID]) != 2 {
		t.Fatalf("Expected 2 photo section instead got back %d", len(photoIntakeSections))
	}
}

func TestPhotoIntake_MultiplePhotos(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoID, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}
	photoID2, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost/2", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataAPI.GetQuestionInfo(otherLocationPhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID,
								SlotID:  photoSlots[0].ID,
							},
							&common.PhotoIntakeSlot{
								Name:    "Other2",
								PhotoID: photoID2,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	photoIntakeSections, err := testData.DataAPI.PatientPhotoSectionsForQuestionIDs([]int64{questionInfo.QuestionID}, patientID, patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if pIntakeSection, ok := photoIntakeSections[questionInfo.QuestionID][0].(*common.PhotoIntakeSection); !ok {
		t.Fatalf("Expected PhotoIntakeSection instead got type %T", pIntakeSection)
	} else if len(pIntakeSection.Photos) != 2 {
		t.Fatalf("Expected 1 photo slot in the section instead got back %d", len(pIntakeSection.Photos))
	}
}

func TestPhotoIntake_AnswerInvalidation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoID, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	photoID2, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost/2", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataAPI.GetQuestionInfo(otherLocationPhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID,
								SlotID:  photoSlots[0].ID,
							},
							&common.PhotoIntakeSlot{
								Name:    "Other2",
								PhotoID: photoID2,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	// now lets go ahead and change the answer for the section to have 1 photo
	photoID3, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost/2", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData = &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing3",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other3",
								PhotoID: photoID3,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	photoIntakeSections, err := testData.DataAPI.PatientPhotoSectionsForQuestionIDs([]int64{questionInfo.QuestionID}, patientID, patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if pIntakeSection, ok := photoIntakeSections[questionInfo.QuestionID][0].(*common.PhotoIntakeSection); !ok {
		t.Fatalf("Expected PhotoIntakeSection instead got type %T", pIntakeSection)
	} else if len(pIntakeSection.Photos) != 1 {
		t.Fatalf("Expected 1 photo slot in the section instead got back %d", len(pIntakeSection.Photos))
	} else if pIntakeSection.Photos[0].PhotoID != photoID3 {
		t.Fatalf("Expected photo id %d for image but got back %d", photoID3, pIntakeSection.Photos[0].PhotoID)
	}
}

func TestPhotoIntake_MultiplePhotoQuestions(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoID, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}
	photoID2, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	questionInfo, err := testData.DataAPI.GetQuestionInfo(otherLocationPhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	questionInfo2, err := testData.DataAPI.GetQuestionInfo(facePhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	photoSlots, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}
	photoSlots2, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo2.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID,
								SlotID:  photoSlots[0].ID,
							},
						},
					},
				},
			},
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo2.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID2,
								SlotID:  photoSlots2[0].ID,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountID.Int64(), requestData, testData, t)

	// ensure that the answers exist for both questions
	photoIntakeSections, err := testData.DataAPI.PatientPhotoSectionsForQuestionIDs([]int64{questionInfo.QuestionID}, patientID, patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if pIntakeSection, ok := photoIntakeSections[questionInfo.QuestionID][0].(*common.PhotoIntakeSection); !ok {
		t.Fatalf("Expected PhotoIntakeSection instead got type %T", photoIntakeSections[0])
	} else if len(pIntakeSection.Photos) != 1 {
		t.Fatalf("Expected 1 photo in the section instead got %d", len(pIntakeSection.Photos))
	}

	photoIntakeSections, err = testData.DataAPI.PatientPhotoSectionsForQuestionIDs([]int64{questionInfo2.QuestionID}, patientID, patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if pIntakeSection, ok := photoIntakeSections[questionInfo2.QuestionID][0].(*common.PhotoIntakeSection); !ok {
		t.Fatalf("Expected PhotoIntakeSection instead got type %T", photoIntakeSections[0])
	} else if len(pIntakeSection.Photos) != 1 {
		t.Fatalf("Expected 1 photo in the section instead got %d", len(pIntakeSection.Photos))
	}
}

func TestPhotoIntake_MistmatchedSlotId(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patient := pr.Patient
	patientID := patient.PatientID.Int64()
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientID, testData, t)

	// simulate photo upload
	photoID, err := testData.DataAPI.AddMedia(patient.PersonID, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	questionInfo, err := testData.DataAPI.GetQuestionInfo(otherLocationPhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	questionInfo2, err := testData.DataAPI.GetQuestionInfo(facePhotoSectionTag, api.LanguageIDEnglish, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	photoSlots2, err := testData.DataAPI.GetPhotoSlotsInfo(questionInfo2.QuestionID, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}

	// this request is badly formed because the slot id represents that of the face photo section
	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID: questionInfo.QuestionID,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoID: photoID,
								SlotID:  photoSlots2[0].ID,
							},
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatal(err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientVisitPhotoAnswerURLPath, "application/json", bytes.NewReader(jsonData), patient.AccountID.Int64())
	if err != nil {
		t.Fatal(err.Error())
	} else if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected response code %d for photo intake but got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

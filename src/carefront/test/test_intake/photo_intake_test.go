package test_intake

import (
	"carefront/api"
	"carefront/common"
	"carefront/patient_visit"
	"carefront/test/test_integration"
	"testing"
)

var (
	otherLocationPhotoSectionTag = "q_other_location_photo_section"
)

func TestPhotoIntake(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientId, testData, t)

	// insert a photo item to simulate uploading of a photo
	photoId, err := testData.DataApi.AddPhoto(patient.PersonId, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataApi.GetQuestionInfo(otherLocationPhotoSectionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataApi.GetPhotoSlots(questionInfo.Id, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// create a request to add 1 photo slot with a particular name for the section
	// to the "Other Location" section
	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitId: patientVisitResponse.PatientVisitId,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionId: questionInfo.Id,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoId: photoId,
								SlotId:  photoSlots[0].Id,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountId.Int64(), requestData, testData, t)

	// ensure that the photos now exist for this question for the patient
	photoIntakeSections, err := testData.DataApi.GetPhotoSectionsCreatedByPatientForQuestion(questionInfo.Id, patientId, patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if len(photoIntakeSections[0].Photos) != 1 {
		t.Fatalf("Expected 1 photo in the section instead got %d", len(photoIntakeSections[0].Photos))
	} else if photoIntakeSections[0].Name != "Testing" {
		t.Fatalf("Expected name %s for section instead got %s", "Testing", photoIntakeSections[0].Name)
	} else if photoIntakeSections[0].Photos[0].Name != "Other" {
		t.Fatalf("Expected name %s for photo slot in the answer instead got %s", "Other", photoIntakeSections[0].Photos[0].Name)
	}
}

func TestPhotoIntake_MultipleSectionsForSameQuestion(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientId, testData, t)

	// insert a photo item to simulate uploading of a photo
	photoId, err := testData.DataApi.AddPhoto(patient.PersonId, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	photoId2, err := testData.DataApi.AddPhoto(patient.PersonId, "http://localhost/2", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataApi.GetQuestionInfo(otherLocationPhotoSectionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataApi.GetPhotoSlots(questionInfo.Id, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// create a request to add 1 photo slot with a particular name for the section
	// to the "Other Location" section
	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitId: patientVisitResponse.PatientVisitId,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionId: questionInfo.Id,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoId: photoId,
								SlotId:  photoSlots[0].Id,
							},
						},
					},
					&common.PhotoIntakeSection{
						Name: "Testing2",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other2",
								PhotoId: photoId2,
								SlotId:  photoSlots[0].Id,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountId.Int64(), requestData, testData, t)

	photoIntakeSections, err := testData.DataApi.GetPhotoSectionsCreatedByPatientForQuestion(questionInfo.Id, patientId, patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 2 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	}
}

func TestPhotoIntake_MultiplePhotos(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	pr := test_integration.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patient := pr.Patient
	patientId := patient.PatientId.Int64()

	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientId, testData, t)

	// insert a photo item to simulate uploading of a photo
	photoId, err := testData.DataApi.AddPhoto(patient.PersonId, "http://localhost", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	photoId2, err := testData.DataApi.AddPhoto(patient.PersonId, "http://localhost/2", "image/jpeg")
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the question that represents the other location photo section
	questionInfo, err := testData.DataApi.GetQuestionInfo(otherLocationPhotoSectionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// get the photo slots associated with this question
	photoSlots, err := testData.DataApi.GetPhotoSlots(questionInfo.Id, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// create a request to add 1 photo slot with a particular name for the section
	// to the "Other Location" section
	requestData := &patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitId: patientVisitResponse.PatientVisitId,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionId: questionInfo.Id,
				PhotoSections: []*common.PhotoIntakeSection{
					&common.PhotoIntakeSection{
						Name: "Testing",
						Photos: []*common.PhotoIntakeSlot{
							&common.PhotoIntakeSlot{
								Name:    "Other",
								PhotoId: photoId,
								SlotId:  photoSlots[0].Id,
							},
							&common.PhotoIntakeSlot{
								Name:    "Other2",
								PhotoId: photoId2,
								SlotId:  photoSlots[0].Id,
							},
						},
					},
				},
			},
		},
	}

	test_integration.SubmitPhotoSectionsForQuestionInPatientVisit(patient.AccountId.Int64(), requestData, testData, t)

	photoIntakeSections, err := testData.DataApi.GetPhotoSectionsCreatedByPatientForQuestion(questionInfo.Id, patientId, patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err.Error())
	} else if len(photoIntakeSections) != 1 {
		t.Fatalf("Expected 1 photo section instead got back %d", len(photoIntakeSections))
	} else if len(photoIntakeSections[0].Photos) != 2 {
		t.Fatalf("Expected 1 photo slot in the section instead got back %d", len(photoIntakeSections[0].Photos))
	}
}

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
}

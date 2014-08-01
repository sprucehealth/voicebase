package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

func TestVisitDiagnosis(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	testData.StartAPIServer(t)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err)
	}

	pr, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	diagnosisQuestionId := getQuestionIdForQuestionTag("q_acne_diagnosis", testData, t)
	acneTypeQuestionId := getQuestionIdForQuestionTag("q_acne_type", testData, t)
	rosaceaTypeQuestionId := getQuestionIdForQuestionTag("q_acne_rosacea_type", testData, t)
	describeConditionQuestionId := getQuestionIdForQuestionTag("q_diagnosis_describe_condition", testData, t)

	answerIntakeRequestBody := setupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_inflammatory"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	// at this point the diagnosis on the case should be set
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != "Inflammatory Acne" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Inflammatory Acne", patientVisit.Diagnosis)
	}

	// let's just update the diagnosis type to ensure that the case diagnosis gets updated
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_comedonal"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientVisit, err = testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != "Comedonal Acne" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Comedonal Acne", patientVisit.Diagnosis)
	}

	// now lets try picking multiple types to describe a combination of an acne type
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_comedonal", "a_acne_inflammatory"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientVisit, err = testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != "Comedonal and Inflammatory Acne" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Comedonal and Inflammatory Acne", patientVisit.Diagnosis)
	}

	// lets try a different diagnosis category
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId:   []string{"a_doctor_acne_rosacea"},
		rosaceaTypeQuestionId: []string{"a_acne_papulopstular_rosacea"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientVisit, err = testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != "Papulopustular Rosacea" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Papulopustular Rosacea", patientVisit.Diagnosis)
	}

	// let's try multiple typed picked for this category
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId:   []string{"a_doctor_acne_rosacea"},
		rosaceaTypeQuestionId: []string{"a_acne_papulopstular_rosacea", "a_acne_erythematotelangiectatic_rosacea", "a_acne_ocular_rosacea"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientVisit, err = testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != "Papulopustular, Erythematotelangiectatic and Ocular Rosacea" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Papulopustular, Erythematotelangiectatic and Ocular Rosacea", patientVisit.Diagnosis)
	}

	// lets try another category where we don't pick a diagnosis type
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_perioral_dermatitis"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientVisit, err = testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != "Perioral dermatitis" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Perioral Dermatitis", patientVisit.Diagnosis)
	}

	// now lets try describing a custom condition
	customCondition := "Ingrown hair"
	answerIntakeRequestBody = &apiservice.AnswerIntakeRequestBody{
		PatientVisitId: pr.PatientVisitId,
		Questions: []*apiservice.AnswerToQuestionItem{
			&apiservice.AnswerToQuestionItem{
				QuestionId: describeConditionQuestionId,
				AnswerIntakes: []*apiservice.AnswerItem{
					&apiservice.AnswerItem{
						AnswerText: customCondition,
					},
				},
			},
		},
	}
	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientVisit, err = testData.DataApi.GetPatientVisitFromId(pr.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientVisit.Diagnosis != customCondition {
		t.Fatalf("Expected diagnosis to be %s but got %s", customCondition, patientVisit.Diagnosis)
	}

}

func getQuestionIdForQuestionTag(questionTag string, testData *TestData, t *testing.T) int64 {
	qi, err := testData.DataApi.GetQuestionInfo(questionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err)
	}

	return qi.QuestionId
}

func setupAnswerIntakeForDiagnosis(questionIdToAnswerTagMapping map[int64][]string, patientVisitId int64, testData *TestData, t *testing.T) *apiservice.AnswerIntakeRequestBody {
	answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitId

	i := 0
	answerIntakeRequestBody.Questions = make([]*apiservice.AnswerToQuestionItem, len(questionIdToAnswerTagMapping))
	for questionId, answerTags := range questionIdToAnswerTagMapping {
		answerInfoList, err := testData.DataApi.GetAnswerInfoForTags(answerTags, api.EN_LANGUAGE_ID)
		if err != nil {
			t.Fatal(err)
		}
		answerIntakeRequestBody.Questions[i] = &apiservice.AnswerToQuestionItem{}
		answerIntakeRequestBody.Questions[i].QuestionId = questionId
		answerIntakeRequestBody.Questions[i].AnswerIntakes = make([]*apiservice.AnswerItem, len(answerInfoList))
		for j, answerInfoItem := range answerInfoList {
			answerIntakeRequestBody.Questions[i].AnswerIntakes[j] = &apiservice.AnswerItem{PotentialAnswerId: answerInfoItem.AnswerId}
		}
		i++
	}
	return answerIntakeRequestBody
}

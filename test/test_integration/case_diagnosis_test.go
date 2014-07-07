package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

func TestCaseDiagnosis(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal(err)
	}

	pr, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	diagnosisQuestionId := getQuestionIdForQuestionTag("q_acne_diagnosis", testData, t)
	acneTypeQuestionId := getQuestionIdForQuestionTag("q_acne_type", testData, t)
	describeConditionQuestionId := getQuestionIdForQuestionTag("q_diagnosis_describe_condition", testData, t)

	answerIntakeRequestBody := setupAnswerIntakeForDiagnosis(map[int64]string{
		diagnosisQuestionId: "a_doctor_acne_vulgaris",
		acneTypeQuestionId:  "a_acne_inflammatory",
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	// at this point the diagnosis on the case should be set
	patientCase, err := testData.DataApi.GetPatientCaseFromId(treatmentPlan.PatientCaseId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Diagnosis != "Inflammatory Acne" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Inflammatory Acne", patientCase.Diagnosis)
	}

	// let's just update the diagnosis type to ensure that the case diagnosis gets updated
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64]string{
		diagnosisQuestionId: "a_doctor_acne_vulgaris",
		acneTypeQuestionId:  "a_acne_comedonal",
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientCase, err = testData.DataApi.GetPatientCaseFromId(treatmentPlan.PatientCaseId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Diagnosis != "Comedonal Acne" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Comedonal Acne", patientCase.Diagnosis)
	}

	// lets try a different diagnosis category
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64]string{
		diagnosisQuestionId: "a_doctor_acne_rosacea",
		acneTypeQuestionId:  "a_acne_papulopstular_rosacea",
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientCase, err = testData.DataApi.GetPatientCaseFromId(treatmentPlan.PatientCaseId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Diagnosis != "Papulopustular Rosacea" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Papulopustular Rosacea", patientCase.Diagnosis)
	}

	// lets try another category where we don't pick a diagnosis type
	answerIntakeRequestBody = setupAnswerIntakeForDiagnosis(map[int64]string{
		diagnosisQuestionId: "a_doctor_acne_perioral_dermatitis",
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	patientCase, err = testData.DataApi.GetPatientCaseFromId(treatmentPlan.PatientCaseId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Diagnosis != "Perioral dermatitis" {
		t.Fatalf("Expected diagnosis to be %s but got %s", "Perioral Dermatitis", patientCase.Diagnosis)
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

	patientCase, err = testData.DataApi.GetPatientCaseFromId(treatmentPlan.PatientCaseId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Diagnosis != customCondition {
		t.Fatalf("Expected diagnosis to be %s but got %s", customCondition, patientCase.Diagnosis)
	}

}

func getQuestionIdForQuestionTag(questionTag string, testData *TestData, t *testing.T) int64 {
	qi, err := testData.DataApi.GetQuestionInfo(questionTag, api.EN_LANGUAGE_ID)
	if err != nil {
		t.Fatal(err)
	}

	return qi.QuestionId
}

func setupAnswerIntakeForDiagnosis(questionIdToAnswerTagMapping map[int64]string, patientVisitId int64, testData *TestData, t *testing.T) *apiservice.AnswerIntakeRequestBody {
	answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitId

	i := 0
	answerIntakeRequestBody.Questions = make([]*apiservice.AnswerToQuestionItem, len(questionIdToAnswerTagMapping))
	for questionId, answerTag := range questionIdToAnswerTagMapping {
		answerInfo, err := testData.DataApi.GetAnswerInfoForTags([]string{answerTag}, api.EN_LANGUAGE_ID)
		if err != nil {
			t.Fatal(err)
		}
		answerIntakeRequestBody.Questions[i] = &apiservice.AnswerToQuestionItem{}
		answerIntakeRequestBody.Questions[i].QuestionId = questionId
		answerIntakeRequestBody.Questions[i].AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: answerInfo[0].AnswerId}}
		i++
	}
	return answerIntakeRequestBody
}

package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/test"
)

func TestVisitDiagnosis(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	test.OK(t, err)

	pr, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	diagnosisQuestionId := GetQuestionIdForQuestionTag("q_acne_diagnosis", testData, t)
	acneTypeQuestionId := GetQuestionIdForQuestionTag("q_acne_type", testData, t)
	rosaceaTypeQuestionId := GetQuestionIdForQuestionTag("q_acne_rosacea_type", testData, t)
	describeConditionQuestionId := GetQuestionIdForQuestionTag("q_diagnosis_describe_condition", testData, t)

	answerIntakeRequestBody := SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_inflammatory"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	// at this point the diagnosis on the case should be set
	diagnosis, err := testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, "Inflammatory Acne", diagnosis)

	// let's just update the diagnosis type to ensure that the case diagnosis gets updated
	answerIntakeRequestBody = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_comedonal"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	diagnosis, err = testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, "Comedonal Acne", diagnosis)

	// now lets try picking multiple types to describe a combination of an acne type
	answerIntakeRequestBody = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_comedonal", "a_acne_inflammatory"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	diagnosis, err = testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, "Comedonal and Inflammatory Acne", diagnosis)

	// lets try a different diagnosis category
	answerIntakeRequestBody = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId:   []string{"a_doctor_acne_rosacea"},
		rosaceaTypeQuestionId: []string{"a_acne_papulopstular_rosacea"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	diagnosis, err = testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, "Papulopustular Rosacea", diagnosis)

	// let's try multiple typed picked for this category
	answerIntakeRequestBody = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId:   []string{"a_doctor_acne_rosacea"},
		rosaceaTypeQuestionId: []string{"a_acne_papulopstular_rosacea", "a_acne_erythematotelangiectatic_rosacea", "a_acne_ocular_rosacea"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	diagnosis, err = testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, "Papulopustular, Erythematotelangiectatic and Ocular Rosacea", diagnosis)

	// lets try another category where we don't pick a diagnosis type
	answerIntakeRequestBody = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_perioral_dermatitis"},
	}, pr.PatientVisitId, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitId, doctor.AccountId.Int64(), answerIntakeRequestBody, testData, t)

	diagnosis, err = testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, "Perioral dermatitis", diagnosis)

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

	diagnosis, err = testData.DataApi.DiagnosisForVisit(pr.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, customCondition, diagnosis)
}

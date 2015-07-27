package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/test"
)

func TestVisitDiagnosis(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	pr, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	diagnosisQuestionID := GetQuestionIDForQuestionTag("q_acne_diagnosis", 1, testData, t)
	acneTypeQuestionID := GetQuestionIDForQuestionTag("q_acne_type", 1, testData, t)
	rosaceaTypeQuestionID := GetQuestionIDForQuestionTag("q_acne_rosacea_type", 1, testData, t)
	describeConditionQuestionID := GetQuestionIDForQuestionTag("q_diagnosis_describe_condition", 1, testData, t)

	intakeData := SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionID:  []string{"a_acne_inflammatory"},
	}, pr.PatientVisitID, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	// at this point the diagnosis on the case should be set
	diagnosis, err := testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "Inflammatory Acne", diagnosis)

	// let's just update the diagnosis type to ensure that the case diagnosis gets updated
	intakeData = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionID:  []string{"a_acne_comedonal"},
	}, pr.PatientVisitID, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	diagnosis, err = testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "Comedonal Acne", diagnosis)

	// now lets try picking multiple types to describe a combination of an acne type
	intakeData = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionID:  []string{"a_acne_comedonal", "a_acne_inflammatory"},
	}, pr.PatientVisitID, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	diagnosis, err = testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "Comedonal and Inflammatory Acne", diagnosis)

	// lets try a different diagnosis category
	intakeData = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID:   []string{"a_doctor_acne_rosacea"},
		rosaceaTypeQuestionID: []string{"a_acne_papulopstular_rosacea"},
	}, pr.PatientVisitID, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	diagnosis, err = testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "Papulopustular Rosacea", diagnosis)

	// let's try multiple typed picked for this category
	intakeData = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID:   []string{"a_doctor_acne_rosacea"},
		rosaceaTypeQuestionID: []string{"a_acne_papulopstular_rosacea", "a_acne_erythematotelangiectatic_rosacea", "a_acne_ocular_rosacea"},
	}, pr.PatientVisitID, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	diagnosis, err = testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "Papulopustular, Erythematotelangiectatic and Ocular Rosacea", diagnosis)

	// lets try another category where we don't pick a diagnosis type
	intakeData = SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID: []string{"a_doctor_acne_perioral_dermatitis"},
	}, pr.PatientVisitID, testData, t)

	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	diagnosis, err = testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "Perioral dermatitis", diagnosis)

	// now lets try describing a custom condition
	customCondition := "Ingrown hair"
	intakeData = &apiservice.IntakeData{
		PatientVisitID: pr.PatientVisitID,
		Questions: []*apiservice.QuestionAnswerItem{
			&apiservice.QuestionAnswerItem{
				QuestionID: describeConditionQuestionID,
				AnswerIntakes: []*apiservice.AnswerItem{
					&apiservice.AnswerItem{
						AnswerText: customCondition,
					},
				},
			},
		},
	}
	SubmitPatientVisitDiagnosisWithIntake(pr.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	diagnosis, err = testData.DataAPI.DiagnosisForVisit(pr.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, customCondition, diagnosis)
}

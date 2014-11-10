package test_ma

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestMAAccess_PatientInfo(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	// MA should be able to get patient information
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorPatientInfoURLPath+"?patient_id="+strconv.FormatInt(pr.Patient.PatientId.Int64(), 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should not be able to update patient information
	jsonData, err := json.Marshal(map[string]interface{}{"patient": pr.Patient})
	test.OK(t, err)
	res, err = testData.AuthPut(testData.APIServer.URL+router.DoctorPatientInfoURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

}

func TestMAAccess_VisitReview(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// MA should be able to review patient's visit information
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorVisitReviewURLPath+"?patient_visit_id="+strconv.FormatInt(pv.PatientVisitId, 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// The case should not get claimed by the MA opening the visit
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)
	test.Equals(t, common.PCStatusUnclaimed, patientCase.Status)

	// MA should be able to review patient's visit information even for a case that is currently claimed by another doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	pv, _ = test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorVisitReviewURLPath+"?patient_visit_id="+strconv.FormatInt(pv.PatientVisitId, 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
}

func TestMAAccess_Diagnosis(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	// MA should be able to get diagnosis information for a patient visit
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitId, doctor, testData, t)
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorVisitDiagnosisURLPath+"?patient_visit_id="+strconv.FormatInt(pv.PatientVisitId, 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should not be able to modify diagnosis information
	answerRequest := test_integration.PrepareAnswersForDiagnosis(testData, t, pv.PatientVisitId)
	jsonData, err := json.Marshal(answerRequest)
	test.OK(t, err)
	res, err = testData.AuthPost(testData.APIServer.URL+router.DoctorVisitDiagnosisURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)
}

func TestMAAccess_TreatmentPlan(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// MA should be able to view a list of treatment plans
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorTreatmentPlansListURLPath+"?patient_id="+strconv.FormatInt(tp.PatientId, 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should be able to view treatment plans
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(tp.Id.Int64(), 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should not be able to start a treatment plan
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentId:   tp.Id,
			ParentType: common.TPParentTypeTreatmentPlan,
		},
	})
	test.OK(t, err)

	res, err = testData.AuthPost(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

	// MA should not be able to update a treatment plan
	// (first lets get the doctor to start a new version of the treatment plan; then we will try getting the MA to update it)
	res, err = testData.AuthPost(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	tpResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.NewDecoder(res.Body).Decode(tpResponse)
	test.OK(t, err)

	// MA should not be able to add medication
	treatment1 := &common.Treatment{
		DrugInternalName: "DrugName (DrugRoute - DrugForm)",
		DosageStrength:   "10 mg",
		DispenseValue:    1,
		DispenseUnitId:   encoding.NewObjectId(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		OTC:                 true,
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient insturctions",
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}
	jsonData, err = json.Marshal(&common.TreatmentList{
		Treatments: []*common.Treatment{treatment1},
	})
	test.OK(t, err)
	res, err = testData.AuthPost(testData.APIServer.URL+router.DoctorVisitTreatmentsURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

	// MA should not be able to update regimen plan
	regimenPlan := &common.RegimenPlan{TreatmentPlanId: tpResponse.TreatmentPlan.Id}
	jsonData, err = json.Marshal(regimenPlan)
	test.OK(t, err)
	res, err = testData.AuthPost(testData.APIServer.URL+router.DoctorRegimenURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

	//MA should not be able to submit visit
	jsonData, err = json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: tpResponse.TreatmentPlan.Id.Int64(),
	})
	res, err = testData.AuthPut(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)
}

func TestMAAccess_CaseMessages(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	// MA should be able to view message thread
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseId.Int64(),
		Message: "foo",
	})

	res, err := testData.AuthGet(testData.APIServer.URL+router.CaseMessagesListURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseId.Int64(), 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should be able to post to message thread
	test_integration.PostCaseMessage(t, testData, ma.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseId.Int64(),
		Message: "foo1",
	})

	// MA should be able to view all messages when both patient and doctor have sent messages
	test_integration.PostCaseMessage(t, testData, patient.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseId.Int64(),
		Message: "foo2",
	})

	res, err = testData.AuthGet(testData.APIServer.URL+router.CaseMessagesListURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseId.Int64(), 10), ma.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	listResponse := &messages.ListResponse{}
	err = json.NewDecoder(res.Body).Decode(listResponse)
	test.OK(t, err)
	test.Equals(t, 3, len(listResponse.Items))
	test.Equals(t, 3, len(listResponse.Participants))

}

func TestMAAccess_RXError(t *testing.T) {
	// TODO
}

func TestMAAccess_RefillRx(t *testing.T) {
	// TODO
}

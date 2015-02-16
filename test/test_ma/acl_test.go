package test_ma

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestMAAccess_PatientInfo(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	// MA should be able to get patient information
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorPatientInfoURLPath+"?patient_id="+strconv.FormatInt(pr.Patient.PatientID.Int64(), 10), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should not be able to update patient information
	jsonData, err := json.Marshal(map[string]interface{}{"patient": pr.Patient})
	test.OK(t, err)
	res, err = testData.AuthPut(testData.APIServer.URL+apipaths.DoctorPatientInfoURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

}

func TestMAAccess_VisitReview(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// MA should be able to review patient's visit information
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorVisitReviewURLPath+"?patient_visit_id="+strconv.FormatInt(pv.PatientVisitID, 10), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// The case should not get claimed by the MA opening the visit
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, false, patientCase.Claimed)

	_, err = testData.DataAPI.GetTempClaimedCaseInQueue(patientCase.ID.Int64())
	test.Equals(t, true, api.IsErrNotFound(err))

	// The status of the visit should not change to reviewing when the MA opens the patient visit
	visit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, visit.Status)

	// MA should be able to review patient's visit information even for a case that is currently claimed by another doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	pv, _ = test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.DoctorVisitReviewURLPath+"?patient_visit_id="+strconv.FormatInt(pv.PatientVisitID, 10), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
}

func TestMAAccess_Diagnosis(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	// MA should be able to get diagnosis information for a patient visit
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitID, doctor, testData, t)
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorVisitDiagnosisURLPath+"?patient_visit_id="+strconv.FormatInt(pv.PatientVisitID, 10), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should not be able to modify diagnosis information
	answerRequest := test_integration.PrepareAnswersForDiagnosis(testData, t, pv.PatientVisitID)
	jsonData, err := json.Marshal(answerRequest)
	test.OK(t, err)
	res, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorVisitDiagnosisURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)
}

func TestMAAccess_TreatmentPlan(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	// MA should be able to view a list of treatment plans
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorTreatmentPlansListURLPath+"?patient_id="+strconv.FormatInt(tp.PatientID, 10), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should be able to view treatment plans
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(tp.ID.Int64(), 10), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// MA should not be able to start a treatment plan
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentID:   tp.ID,
			ParentType: common.TPParentTypeTreatmentPlan,
		},
	})
	test.OK(t, err)

	res, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

	// MA should not be able to update a treatment plan
	// (first lets get the doctor to start a new version of the treatment plan; then we will try getting the MA to update it)
	res, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
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
		DispenseUnitID:   encoding.NewObjectID(26),
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
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}
	jsonData, err = json.Marshal(&common.TreatmentList{
		Treatments: []*common.Treatment{treatment1},
	})
	test.OK(t, err)
	res, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorVisitTreatmentsURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

	// MA should not be able to update regimen plan
	regimenPlan := &common.RegimenPlan{TreatmentPlanID: tpResponse.TreatmentPlan.ID}
	jsonData, err = json.Marshal(regimenPlan)
	test.OK(t, err)
	res, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorRegimenURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)

	//MA should not be able to submit visit
	jsonData, err = json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanID: tpResponse.TreatmentPlan.ID.Int64(),
	})
	res, err = testData.AuthPut(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusForbidden, res.StatusCode)
}

func TestMAAccess_CaseMessages(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	// MA should be able to view message thread
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	doctorCli := test_integration.DoctorClient(testData, t, dr.DoctorID)
	maCli := test_integration.DoctorClient(testData, t, ma.DoctorID.Int64())
	patientCli := test_integration.PatientClient(testData, t, patient.PatientID.Int64())

	_, err = doctorCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)
	_, _, err = maCli.ListCaseMessages(tp.PatientCaseID.Int64())
	test.OK(t, err)
	_, err = maCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo2", nil)
	test.OK(t, err)
	_, err = patientCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo1", nil)
	test.OK(t, err)

	// MA should be able to view all messages when both patient and doctor have sent messages
	msgs, part, err := maCli.ListCaseMessages(tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, 3, len(msgs))
	test.Equals(t, 3, len(part))
}

func TestMAAccess_RXError(t *testing.T) {
	// TODO
}

func TestMAAccess_RefillRx(t *testing.T) {
	// TODO
}

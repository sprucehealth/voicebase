package test_patient_visit

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// Test to ensure that the same user can start multiple visits
// for different pathways with the constraint being that just a single
// open case can exist per pathway
func TestPatientVisit_MultiplePathways(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create a doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// create a patient
	pr := test_integration.SignupRandomTestPatient(t, testData)
	pc := test_integration.PatientClient(testData, t, pr.Patient.PatientID.Int64())

	// create new pathway
	p1 := &common.Pathway{
		Name:           "test1",
		MedicineBranch: "test",
		Tag:            "test1",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(p1))

	// upload layouts for pathway
	uploadLayoutPairForPathway(p1.Tag, testData, t)

	// register doctor in CA for this new pathway
	careProvidingStateID, err := testData.DataAPI.AddCareProvidingState("CA", "California", p1.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr.DoctorID))

	// start a visit for the pathway
	_, err = pc.CreatePatientVisit(p1.Tag, dr.DoctorID, setupTestHeaders())
	test.OK(t, err)

	// create another pathway
	p2 := &common.Pathway{
		Name:           "test2",
		MedicineBranch: "test",
		Tag:            "test2",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(p2))
	// upload layouts for pathway
	uploadLayoutPairForPathway(p2.Tag, testData, t)
	// register doctor in CA for this new pathway
	careProvidingStateID, err = testData.DataAPI.AddCareProvidingState("CA", "California", p2.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr.DoctorID))

	// start a visit for the pathway
	_, err = pc.CreatePatientVisit(p2.Tag, dr.DoctorID, setupTestHeaders())
	test.OK(t, err)

	// at this point the patient should have 2 open cases
	cases, err := testData.DataAPI.GetCasesForPatient(pr.Patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
	test.Equals(t, common.PCStatusUnclaimed, cases[0].Status)
	test.Equals(t, common.PCStatusUnclaimed, cases[1].Status)

}

func setupTestHeaders() http.Header {
	headers := http.Header(make(map[string][]string))
	headers.Set("S-Version", "Patient;Feature;1.0.0;000105")
	headers.Set("S-OS", "iOS;7.1.1")
	headers.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	headers.Set("S-Device-ID", "12345678-1234-1234-1234-123456789abc")
	return headers

}

func uploadLayoutPairForPathway(pathwayTag string, testData *test_integration.TestData, t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// read in the intake layout and modify the pathway tag
	data, err := ioutil.ReadFile(test_integration.IntakeFileLocation)
	test.OK(t, err)
	var intakeJsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(data, &intakeJsonMap))
	intakeJsonMap["health_condition"] = pathwayTag
	intakeJsonData, err := json.Marshal(intakeJsonMap)
	test.OK(t, err)

	// read in the review layout and modify the pathway tag
	data, err = ioutil.ReadFile(test_integration.ReviewFileLocation)
	test.OK(t, err)
	var reviewJsonMap map[string]interface{}
	test.OK(t, json.Unmarshal(data, &reviewJsonMap))
	reviewJsonMap["health_condition"] = pathwayTag
	reviewJsonData, err := json.Marshal(reviewJsonMap)
	test.OK(t, err)

	// now write the intake and review files to the multipart writer
	part, err := writer.CreateFormFile("intake", "intake-1-0-0.json")
	test.OK(t, err)
	_, err = part.Write(intakeJsonData)
	test.OK(t, err)
	part, err = writer.CreateFormFile("review", "review-1-0-0.json")
	test.OK(t, err)
	_, err = part.Write(reviewJsonData)
	test.OK(t, err)

	// specify the app versions and the platform information
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)

	test.OK(t, writer.Close())

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

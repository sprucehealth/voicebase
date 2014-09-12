package test_demo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTrainingCase(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Upload the latest versions of the review and intake
	latestIntakeVersion := determineLatestVersionedFile("intake", t)
	latestReviewVersion := determineLatestVersionedFile("review", t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", latestIntakeVersion, "../../info_intake/"+latestIntakeVersion, t)
	test_integration.AddFileToMultipartWriter(writer, "review", latestReviewVersion, "../../info_intake/"+latestReviewVersion, t)
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)
	err := writer.Close()
	test.OK(t, err)
	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// have the demo worker run ones to create the training cases
	demo.LocalServerURL = testData.APIServer.URL
	demo.StartWorker(testData.DataApi, "www.spruce.local", "us-east-1", 24*60*60)

	// wait until the training cases have been created
	time.Sleep(2 * time.Second)

	// check for number of pending training cases. It should be greater than 0
	pendingTrainingCases, err := testData.DataApi.TrainingCaseSetCount(common.TCSStatusPending)
	test.OK(t, err)
	test.Equals(t, true, pendingTrainingCases > 0)

	// lets get a doctor to claim 1 training case set
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	resp, err = testData.AuthGet(testData.APIServer.URL+router.TrainingCasesURLPath, doctor.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now the doctor should have non-zero number of pending cases in their inbox
	pendingVisits, err := testData.DataApi.GetPendingItemsInDoctorQueue(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, true, len(pendingVisits) > 0)

	// now lets go ahead and try to diagnose one of those cases up until the point of visit submission
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(pendingVisits[0].ItemId)
	test.OK(t, err)
	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, patientVisit.PatientCaseId.Int64())
	test_integration.StartReviewingPatientVisit(patientVisit.PatientVisitId.Int64(), doctor, testData, t)
	test_integration.SubmitPatientVisitDiagnosis(patientVisit.PatientVisitId.Int64(), doctor, testData, t)
	tp := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   patientVisit.PatientVisitId,
		ParentType: common.TPParentTypePatientVisit,
	}, nil, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.TreatmentPlan.Id.Int64(), doctor, testData, t)

	// now attempt to add an FTP for the doctor using the endpoint
	jsonData, err := json.Marshal(map[string]interface{}{
		"tag": "doxy_and_tretinoin",
	})
	test.OK(t, err)

	resp, err = testData.AuthPost(testData.APIServer.URL+router.CreateDemoFTPURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func determineLatestVersionedFile(prefix string, t *testing.T) string {
	files, err := ioutil.ReadDir("../../info_intake/")
	test.OK(t, err)

	var fileNamesToCompare []string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) {
			fileNamesToCompare = append(fileNamesToCompare, f.Name())
		}
	}

	if len(fileNamesToCompare) > 0 {
		sort.Strings(fileNamesToCompare)
	} else {
		t.Fatalf("File with prefix %s not found", prefix)
	}

	return fileNamesToCompare[len(fileNamesToCompare)-1]
}

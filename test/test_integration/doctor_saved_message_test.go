package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"strconv"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoctorSavedMessage(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(apiservice.NewDoctorSavedMessageHandler(testData.DataApi))
	defer ts.Close()

	// Check that the doctor can retrieve a saved message
	initialMsg := `{"message":""}`
	res, err := testData.AuthGet(ts.URL, doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != initialMsg {
		t.Fatalf("Expected %q got %q", initialMsg, s)
	}
	
	// Save a default saved message
	defaultMsg := `{"message":"foo"}`
	res, err = testData.AuthPut(ts.URL, "application/json", bytes.NewReader([]byte(defaultMsg)), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	// Check that the doctor can retrieve default saved message
	res, err = testData.AuthGet(ts.URL, doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != defaultMsg {
		t.Fatalf("Expected %q got %q", defaultMsg, s)
	}
}

func TestDoctorUpdateTreatmentPlanMessage(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(apiservice.NewDoctorSavedMessageHandler(testData.DataApi))
	defer ts.Close()


	// Create a patient treatment plan, and save a draft message
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	tpMessage := `{"message":"Dear foo, this is my message"}`
	requestData := apiservice.DoctorSavedMessagePutRequest{
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message: "Dear foo, this is my message",
	}
	jsonData, err := json.Marshal(requestData)

	if err != nil {
		t.Fatal(err)
	}

	res, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	// Retreieve treatment plan message
	res, err = testData.AuthGet(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != tpMessage {
		t.Fatalf("Expected %q got %q", tpMessage, s)
	}

	// Update treatment plan message
	newTpMessage := `{"message":"Dear foo, I have changed my mind"}`
	requestData = apiservice.DoctorSavedMessagePutRequest{
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message: "Dear foo, I have changed my mind",
	}
	jsonData, err = json.Marshal(requestData)

	if err != nil {
		t.Fatal(err)
	}

	res, err = testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	res, err = testData.AuthGet(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != newTpMessage {
		t.Fatalf("Expected %q got %q", newTpMessage, s)
	}
}

func TestDoctorMultipleTreatmentPlans(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	
	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(apiservice.NewDoctorSavedMessageHandler(testData.DataApi))
	defer ts.Close()

	// Create default message
	newJS := `{"message":"default message"}`
	res, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader([]byte(newJS)), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	// Create patient, save message to their treatment plan, and retrieve it
	tpMessage := `{"message":"Dear patient, this is not a default message"}`
	pv, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	requestData := apiservice.DoctorSavedMessagePutRequest{
		DoctorID: doctor.AccountId.Int64(),
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message: "Dear patient, this is not a default message",
	}
	jsonData, err := json.Marshal(requestData)

	if err != nil {
		t.Fatal(err)
	}
	res, err = testData.AuthPut(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(treatmentPlan.Id.Int64(), 10), "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	res, err = testData.AuthGet(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != tpMessage {
		t.Fatalf("Expected %q got %q", newJS, s)
	}

	// Choose a different treatment plan for the same patient, and retrieve message from new treatment plan
	tp := PickATreatmentPlanForPatientVisit(pv.PatientVisitId, doctor, nil, testData, t).TreatmentPlan
	res, err = testData.AuthGet(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(tp.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != newJS {
		t.Fatalf("Expected %q got %q", newJS, s)
	}

}

func TestDoctorSubmitTreatmentPlan(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	
	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(apiservice.NewDoctorSavedMessageHandler(testData.DataApi))
	defer ts.Close()

	// Create default message
	message := `{"message":"default message"}`
	res, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader([]byte(message)), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	// Create patient, save message to their treatment plan
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	requestData := apiservice.DoctorSavedMessagePutRequest{
		DoctorID: doctor.AccountId.Int64(),
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message: "Dear patient, this is not a default message",
	}
	jsonData, err := json.Marshal(requestData)

	if err != nil {
		t.Fatal(err)
	}
	res, err = testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	erxStatusQueue := &common.SQSQueue{}
	erxStatusQueue.QueueService = &sqs.StubSQS{}
	erxStatusQueue.QueueUrl = "local-erx"
	submitTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(
		testData.DataApi,
		testData.ERxAPI,
		erxStatusQueue,
		false)

	ts3 := httptest.NewServer(submitTreatmentPlanHandler)
	defer ts3.Close()

	jsonData, err = json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: treatmentPlan.Id,
		Message: "Dear patient, this is not a default message",
	})

	if err != nil {
		t.Fatal(err)
	}

	//Submit treatment plan, then check that the treament plan message draft is deleted. The message returned should be the doctor's default message.
	resp, err := testData.AuthPut(ts3.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	defer resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected %d but got %d: %s", http.StatusOK, resp.StatusCode, string(b))
	}
	time.Sleep(time.Second)
	res, err = testData.AuthGet(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != message {
		t.Fatalf("Expected %q got %q", message, s)
	}
}

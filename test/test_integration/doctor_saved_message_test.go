package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/test"
)

func TestDoctorSavedMessage(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)
	// Check that the doctor can retrieve a saved message
	initialMsg := `{"message":""}`
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != initialMsg {
		t.Fatalf("Expected %q got %q", initialMsg, s)
	}

	// Save a default saved message
	defaultMsg := `{"message":"foo"}`
	res, err = testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader([]byte(defaultMsg)), doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	// Check that the doctor can retrieve default saved message
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != defaultMsg {
		t.Fatalf("Expected %q got %q", defaultMsg, s)
	}
}

func TestDoctorUpdateTreatmentPlanMessage(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	// Create a patient treatment plan, and save a draft message
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	tpMessage := `{"message":"Dear foo, this is my message"}`
	requestData := doctor_treatment_plan.DoctorSavedMessageRequestData{
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message:         "Dear foo, this is my message",
	}
	jsonData, err := json.Marshal(requestData)

	test.OK(t, err)

	res, err := testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	defer res.Body.Close()
	test.OK(t, err)

	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	// Retreieve treatment plan message
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	test.OK(t, err)

	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != tpMessage {
		t.Fatalf("Expected %q got %q", tpMessage, s)
	}

	// Update treatment plan message
	newTpMessage := `{"message":"Dear foo, I have changed my mind"}`
	requestData = doctor_treatment_plan.DoctorSavedMessageRequestData{
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message:         "Dear foo, I have changed my mind",
	}
	jsonData, err = json.Marshal(requestData)

	test.OK(t, err)

	res, err = testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)

	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	test.OK(t, err)

	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != newTpMessage {
		t.Fatalf("Expected %q got %q", newTpMessage, s)
	}
}

func TestDoctorMultipleTreatmentPlans(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	// Create default message
	newJS := `{"message":"default message"}`
	res, err := testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader([]byte(newJS)), doctor.AccountId.Int64())
	defer res.Body.Close()
	test.OK(t, err)

	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	// Create patient, save message to their treatment plan, and retrieve it
	tpMessage := `{"message":"Dear patient, this is not a default message"}`
	pv, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	requestData := doctor_treatment_plan.DoctorSavedMessageRequestData{
		DoctorID:        doctor.AccountId.Int64(),
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message:         "Dear patient, this is not a default message",
	}
	jsonData, err := json.Marshal(requestData)

	test.OK(t, err)
	res, err = testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)

	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	test.OK(t, err)

	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != tpMessage {
		t.Fatalf("Expected %q got %q", newJS, s)
	}

	// Choose a different treatment plan for the same patient, and retrieve message from new treatment plan
	tp := PickATreatmentPlanForPatientVisit(pv.PatientVisitId, doctor, nil, testData, t).TreatmentPlan
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath+"?treatment_plan_id="+strconv.FormatInt(tp.Id.Int64(), 10), doctor.AccountId.Int64())
	test.OK(t, err)

	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != newJS {
		t.Fatalf("Expected %q got %q", newJS, s)
	}

}

func TestDoctorSubmitTreatmentPlan(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	// Create default message
	message := `{"message":"default message"}`
	res, err := testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader([]byte(message)), doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	// Create patient, save message to their treatment plan
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	requestData := doctor_treatment_plan.DoctorSavedMessageRequestData{
		DoctorID:        doctor.AccountId.Int64(),
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message:         "Dear patient, this is not a default message",
	}
	jsonData, err := json.Marshal(requestData)

	test.OK(t, err)
	res, err = testData.AuthPut(testData.APIServer.URL+router.DoctorSavedMessagesURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}

	jsonData, err = json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: treatmentPlan.Id.Int64(),
		Message:         "Dear patient, this is not a default message",
	})

	test.OK(t, err)

	//Submit treatment plan, then check that the treament plan message draft is deleted. The message returned should be the doctor's default message.
	resp, err := testData.AuthPut(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected %d but got %d: %s", http.StatusOK, resp.StatusCode, string(b))
	}
	time.Sleep(time.Second)
	res, err = testData.AuthGet(testData.APIServer.URL+router.DoctorSavedMessagesURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != message {
		t.Fatalf("Expected %q got %q", message, s)
	}
}

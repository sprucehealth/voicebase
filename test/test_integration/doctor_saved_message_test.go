package test_integration

import (
	"bytes"
	"encoding/json"
	"github.com/sprucehealth/backend/apiservice"
	"strconv"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
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

	initialJS := `{"message":""}`
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	res, err := testData.AuthGet(ts.URL + "?treatment_plan_id=" + strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64()) //TODO find a more robust way to choose a random treatment_plan_id
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if b, err := ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	} else if s := strings.TrimSpace(string(b)); s != initialJS {
		t.Fatalf("Expected %q got %q", initialJS, s)
	}

	newJS := `{"message":"foo"}`
	res, err = testData.AuthPut(ts.URL, "application/json", bytes.NewReader([]byte(newJS)), doctor.AccountId.Int64())
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
	} else if s := strings.TrimSpace(string(b)); s != newJS {
		t.Fatalf("Expected %q got %q", newJS, s)
	}

	newJS = `{"message":"Dear foo, this is my message"}`
	requestData := apiservice.DoctorSavedMessagePutRequest{
		DoctorID: doctor.AccountId.Int64(),
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message: "Dear foo, this is my message",
	}
	jsonData, err := json.Marshal(requestData)

	if err != nil {
		t.Fatalf("Unable to marshal favorited treatment plan %s", err)
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
	} else if s := strings.TrimSpace(string(b)); s != newJS {
		t.Fatalf("Expected %q got %q", newJS, s)
	}

	newJS = `{"message":"Dear foo, I have changed my mind"}`
	requestData = apiservice.DoctorSavedMessagePutRequest{
		DoctorID: doctor.AccountId.Int64(),
		TreatmentPlanID: treatmentPlan.Id.Int64(),
		Message: "Dear foo, I have changed my mind",
	}
	jsonData, err = json.Marshal(requestData)

	if err != nil {
		t.Fatalf("Unable to marshal favorited treatment plan %s", err)
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
	} else if s := strings.TrimSpace(string(b)); s != newJS {
		t.Fatalf("Expected %q got %q", newJS, s)
	}
}

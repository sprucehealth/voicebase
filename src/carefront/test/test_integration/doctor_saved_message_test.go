package test_integration

import (
	"bytes"
	"carefront/apiservice"
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
	res, err := testData.AuthGet(ts.URL, doctor.AccountId.Int64())
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

	res, err = testData.AuthGet(ts.URL, doctor.AccountId.Int64())
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

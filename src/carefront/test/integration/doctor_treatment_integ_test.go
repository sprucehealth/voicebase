package integration

import (
	"carefront/apiservice"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestMedicationStrengthSearch(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	erx := setupErxAPI(t)
	medicationStrengthSearchHandler := &apiservice.MedicationStrengthSearchHandler{ERxApi: erx}
	ts := httptest.NewServer(medicationStrengthSearchHandler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "?drug_internal_name=" + url.QueryEscape("Benzoyl Peroxide Topical (topical - cream)"))
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication strength api: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the medication strength api for the doctor: "+string(body), t)
	medicationStrengthResponse := &apiservice.MedicationStrengthSearchResponse{}
	err = json.Unmarshal(body, medicationStrengthResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if medicationStrengthResponse.MedicationStrengths == nil || len(medicationStrengthResponse.MedicationStrengths) == 0 {
		t.Fatal("Expected a list of medication strengths from the api but got none")
	}
}

func TestMedicationSelectSearch(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	erx := setupErxAPI(t)
	medicationSelectHandler := &apiservice.MedicationSelectHandler{ERxApi: erx}
	ts := httptest.NewServer(medicationSelectHandler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "?drug_internal_name=" + url.QueryEscape("Lisinopril (oral - tablet)") + "&medication_strength=" + url.QueryEscape("10 mg"))
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication strength api: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the medication strength api for the doctor: "+string(body), t)
	medicationSelectResponse := &apiservice.MedicationSelectResponse{}
	err = json.Unmarshal(body, medicationSelectResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if medicationSelectResponse.DrugId == 0 {
		t.Fatal("Expected a drug id but got none")
	}

	if medicationSelectResponse.AdditionalDrugDBIds == nil || len(medicationSelectResponse.AdditionalDrugDBIds) == 0 {
		t.Fatal("Expected additional drug db ids to be returned from api but none were")
	}

	if medicationSelectResponse.AdditionalDrugDBIds[erx.LexiDrugSynId] == "0" || medicationSelectResponse.AdditionalDrugDBIds[erx.LexiSynonymTypeId] == "0" {
		t.Fatal("Expected additional drug db ids not set (lexi_drug_syn_id and lexi_synonym_type_id")
	}
}

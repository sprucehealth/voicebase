package test_integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

func TestPatientCareProvidingEllgibility(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	resp, err := http.Get(testData.APIServer.URL + apipaths.CheckEligibilityURLPath + "?zip_code=94115")
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// should be marked as available
	var j map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		t.Fatal(err)
	} else if !j["available"].(bool) {
		t.Fatal("Expected this state to be eligible but it wasnt")
	}

	// when the state code is provided, should skip resolving of zipcode to state
	stubAddressValidationService := testData.Config.AddressValidator.(*address.StubAddressValidationService)
	stubAddressValidationService.CityStateToReturn = nil
	resp, err = http.Get(testData.APIServer.URL + apipaths.CheckEligibilityURLPath + "?state_code=CA")
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
	j = nil
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		t.Fatal(err)
	} else if !j["available"].(bool) {
		t.Fatal("Expected this state to be eligible but it wasnt")
	}

	// when state and zipcode is provided, should still skip resolving of zipcode to state
	resp, err = http.Get(testData.APIServer.URL + apipaths.CheckEligibilityURLPath + "?state_code=CA&zip_code=94115")
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// should be marked as unavailable
	stubAddressValidationService.CityStateToReturn = &address.CityState{
		City:              "Aventura",
		State:             "Florida",
		StateAbbreviation: "FL",
	}
	resp, err = testData.AuthGet(testData.APIServer.URL+apipaths.CheckEligibilityURLPath+"?zip_code=33180", 0)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		t.Fatal(err)
	} else if j["available"].(bool) {
		t.Fatal("Expected this state to be ineligible but it wasnt")
	}
}

func TestSpruceAvailableInState(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := SignupRandomTestDoctor(t, testData)

	// create pathway
	pathway := &common.Pathway{
		Tag:            "test",
		Name:           "Test",
		MedicineBranch: "Derm",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(pathway))

	// register this doctor to see patients in FL
	stateID, err := testData.DataAPI.AddCareProvidingState("FL", "Florida", pathway.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(stateID, dr.DoctorID))

	// spruce should be available in FL
	isAvailable, err := testData.DataAPI.SpruceAvailableInState("FL")
	test.OK(t, err)
	test.Equals(t, true, isAvailable)

	// spruce is not available in PA
	isAvailable, err = testData.DataAPI.SpruceAvailableInState("PA")
	test.OK(t, err)
	test.Equals(t, false, isAvailable)
}

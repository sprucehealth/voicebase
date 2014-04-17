package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/address_validation"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoctorUpdateToPatientAddress(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "1234",
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	patientAddress := &common.Address{
		AddressLine1: "12345 Main Street",
		AddressLine2: "Apt 1212",
		City:         "San Francisco",
		State:        "CA",
		ZipCode:      "94115",
	}
	signedupPatientResponse.Patient.PatientAddress = patientAddress

	stubErxApi := &erx.StubErxService{}

	stubAddressValidationService := address_validation.StubAddressValidationService{
		CityStateToReturn: address_validation.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi:              testData.DataApi,
		ErxApi:               stubErxApi,
		AddressValidationApi: stubAddressValidationService,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&apiservice.DoctorPatientUpdateHandlerRequestResponse{
			Patient: signedupPatientResponse.Patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal patient object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get back patient information from database: " + err.Error())
	}

	if patient.PatientAddress == nil {
		t.Fatal("Expected patient to have address information: ")
	}

	if patient.PatientAddress.AddressLine1 != patientAddress.AddressLine1 || patient.PatientAddress.AddressLine2 != patientAddress.AddressLine2 ||
		patient.PatientAddress.State != "California" || patient.PatientAddress.City != patientAddress.City ||
		patient.PatientAddress.ZipCode != patientAddress.ZipCode {
		t.Fatal("Patient address did not updated to match what was entered")
	}

}

func TestDoctorFailedUpdate(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	// ensure that an update does not go through if we remove the patient address
	// or the dob or phone numbers
	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	signedupPatientResponse.Patient.PhoneNumbers = nil
	stubErxApi := &erx.StubErxService{}
	stubAddressValidationService := address_validation.StubAddressValidationService{
		CityStateToReturn: address_validation.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi:              testData.DataApi,
		ErxApi:               stubErxApi,
		AddressValidationApi: stubAddressValidationService,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&apiservice.DoctorPatientUpdateHandlerRequestResponse{
			Patient: signedupPatientResponse.Patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal patient object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected a %d request due to remove of phone numbers, instead got %d", http.StatusBadRequest, resp.StatusCode)
	}

	signedupPatientResponse.Patient.PhoneNumbers = []*common.PhoneInformation{&common.PhoneInformation{
		Phone:     "1241515",
		PhoneType: "Home",
	}}

	// now lets try no address
	resp, err = authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a failed request due to remove of phone address")
	}

	// now lets try no dob
	signedupPatientResponse.Patient.Dob = encoding.Dob{Month: 11, Day: 8, Year: 1987}
	resp, err = authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a failed request due to remove of date of birth")
	}

}

func TestDoctorUpdateToPhoneNumbers(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "1234",
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	patientAddress := &common.Address{
		AddressLine1: "12345 Main Street",
		AddressLine2: "Apt 1212",
		City:         "San Francisco",
		State:        "CA",
		ZipCode:      "94115",
	}
	signedupPatientResponse.Patient.PatientAddress = patientAddress

	// lets go ahead and modify current phone number list

	phoneNumbers := []*common.PhoneInformation{&common.PhoneInformation{
		Phone:     "7348465522",
		PhoneType: "Home",
	},
		&common.PhoneInformation{
			Phone:     "7348465522",
			PhoneType: "Work",
		},
		&common.PhoneInformation{
			Phone:     "7348465522",
			PhoneType: "Work",
		},
	}
	signedupPatientResponse.Patient.PhoneNumbers = phoneNumbers

	stubErxApi := &erx.StubErxService{}
	stubAddressValidationService := address_validation.StubAddressValidationService{
		CityStateToReturn: address_validation.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}
	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi:              testData.DataApi,
		ErxApi:               stubErxApi,
		AddressValidationApi: stubAddressValidationService,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&apiservice.DoctorPatientUpdateHandlerRequestResponse{
			Patient: signedupPatientResponse.Patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal patient object: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get back patient information from database: " + err.Error())
	}

	if len(patient.PhoneNumbers) != len(phoneNumbers) {
		t.Fatal("Did not get back expected number of phone numbers for patient")
	}

	for i, phoneNumber := range phoneNumbers {
		if phoneNumber.Phone != patient.PhoneNumbers[i].Phone || phoneNumber.PhoneType != patient.PhoneNumbers[i].PhoneType {
			t.Fatal("Expected the phone numbers modified to be the same ones returned")
		}
	}
}

func TestDoctorUpdateToTopLevelInformation(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "1234",
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	patientAddress := &common.Address{
		AddressLine1: "12345 Main Street",
		AddressLine2: "Apt 1212",
		City:         "San Francisco",
		State:        "CA",
		ZipCode:      "94115",
	}
	signedupPatientResponse.Patient.PatientAddress = patientAddress

	signedupPatientResponse.Patient.FirstName = "Test"
	signedupPatientResponse.Patient.LastName = "Test again"
	signedupPatientResponse.Patient.Suffix = "m"
	signedupPatientResponse.Patient.Prefix = "n"
	signedupPatientResponse.Patient.MiddleName = "aaaa"
	signedupPatientResponse.Patient.Gender = "Unknown"
	signedupPatientResponse.Patient.Dob = encoding.Dob{Day: 11, Month: 9, Year: 1987}

	stubErxApi := &erx.StubErxService{}
	stubAddressValidationService := address_validation.StubAddressValidationService{
		CityStateToReturn: address_validation.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}
	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi:              testData.DataApi,
		ErxApi:               stubErxApi,
		AddressValidationApi: stubAddressValidationService,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&apiservice.DoctorPatientUpdateHandlerRequestResponse{
			Patient: signedupPatientResponse.Patient,
		},
	)
	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get back patient information from database: " + err.Error())
	}

	if patient.FirstName != signedupPatientResponse.Patient.FirstName ||
		patient.LastName != signedupPatientResponse.Patient.LastName ||
		patient.MiddleName != signedupPatientResponse.Patient.MiddleName ||
		patient.Suffix != signedupPatientResponse.Patient.Suffix ||
		patient.Prefix != signedupPatientResponse.Patient.Prefix ||
		patient.Dob.Day != signedupPatientResponse.Patient.Dob.Day ||
		patient.Dob.Year != signedupPatientResponse.Patient.Dob.Year ||
		patient.Dob.Month != signedupPatientResponse.Patient.Dob.Month {
		t.Fatal("Patient data incorrectly updated")
	}
}

func TestDoctorUpdatePatientInformationForbidden(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "1234",
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err := testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}
	stubAddressValidationService := address_validation.StubAddressValidationService{
		CityStateToReturn: address_validation.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi:              testData.DataApi,
		ErxApi:               &erx.StubErxService{},
		AddressValidationApi: stubAddressValidationService,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&apiservice.DoctorPatientUpdateHandlerRequestResponse{
			Patient: signedupPatientResponse.Patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal json object: " + err.Error())
	}

	doctor, err := testData.DataApi.GetDoctorFromId(signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("unable to get doctor from id: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successfull call to upte patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatal("Expected the doctor to be forbidden from updating the patient information given that it is not the patient's primary doctor but this was not the case")
	}

}

func TestDoctorPatientPharmacyUpdateHandler(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}
	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "1234",
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	updatedPatientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "12345",
		AddressLine1: "1231515 Updated main street",
		AddressLine2: "124151515 apt",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115325151",
	}

	doctorUpdatePatientPharmacy := &apiservice.DoctorUpdatePatientPharmacyHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorUpdatePatientPharmacy)
	defer ts.Close()

	requestData := &apiservice.DoctorUpdatePatientPharmacyRequestData{
		PatientId: signedupPatientResponse.Patient.PatientId,
		Pharmacy:  updatedPatientPharmacy,
	}

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marhsal data: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful call to update patient information")
	}

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient based on id: " + err.Error())
	}

	if patient.Pharmacy.AddressLine1 != updatedPatientPharmacy.AddressLine1 ||
		patient.Pharmacy.AddressLine2 != updatedPatientPharmacy.AddressLine2 ||
		patient.Pharmacy.City != updatedPatientPharmacy.City ||
		patient.Pharmacy.State != updatedPatientPharmacy.State ||
		patient.Pharmacy.Postal != updatedPatientPharmacy.Postal {
		t.Fatalf("Patient pharmacy not successfully updated: %+v %+v", patient.Pharmacy, updatedPatientPharmacy)
	}
}

func TestDoctorPharmacyUpdateForbidden(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     "1234",
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err := testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	doctorUpdatePatientPharmacy := &apiservice.DoctorUpdatePatientPharmacyHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorUpdatePatientPharmacy)
	defer ts.Close()

	requestData := &apiservice.DoctorUpdatePatientPharmacyRequestData{
		PatientId: signedupPatientResponse.Patient.PatientId,
		Pharmacy:  patientPharmacy,
	}

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marhsal data: " + err.Error())
	}

	doctor, err := testData.DataApi.GetDoctorFromId(signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("unable to get doctor from id: " + err.Error())
	}

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successfull call to upte patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatal("Expected the doctor to be forbidden from updating the patient pharmacy information given that it is not the patient's primary doctor but this was not the case")
	}
}

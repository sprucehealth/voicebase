package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxApi,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(signedupPatientResponse.Patient)

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
		patient.PatientAddress.State != patientAddress.State || patient.PatientAddress.City != patientAddress.City ||
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

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxApi,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(signedupPatientResponse.Patient)

	resp, err := authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a failed request due to remove of phone numbers")
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
	signedupPatientResponse.Patient.Dob = time.Time{}
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
		Phone:     "123456",
		PhoneType: "Home",
	},
		&common.PhoneInformation{
			Phone:     "1231515",
			PhoneType: "Work",
		},
		&common.PhoneInformation{
			Phone:     "12341515",
			PhoneType: "Work",
		},
	}
	signedupPatientResponse.Patient.PhoneNumbers = phoneNumbers

	stubErxApi := &erx.StubErxService{}

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxApi,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(signedupPatientResponse.Patient)

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
	signedupPatientResponse.Patient.Dob = time.Now()

	stubErxApi := &erx.StubErxService{}

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := &apiservice.DoctorPatientUpdateHandler{
		DataApi: testData.DataApi,
		ErxApi:  stubErxApi,
	}

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(signedupPatientResponse.Patient)

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
		patient.Dob.Day() != signedupPatientResponse.Patient.Dob.Day() ||
		patient.Dob.Year() != signedupPatientResponse.Patient.Dob.Year() ||
		patient.Dob.Month() != signedupPatientResponse.Patient.Dob.Month() {
		t.Fatal("Patient data incorrectly updated")
	}
}

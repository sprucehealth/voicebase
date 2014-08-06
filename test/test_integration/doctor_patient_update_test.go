package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/pharmacy"
)

type requestData struct {
	Patient *common.Patient `json:"patient"`
}

func TestDoctorUpdateToPatientAddress(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	// the only way a doctor can update a patient's information is if they are assigned to them. and the only way
	// to currently be assigned to them is to grab the item from the unclaimed queue by opening the patient visit
	patientVisitResponse, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     1234,
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	patient, err := testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patientPharmacy)
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
	patient.PatientAddress = patientAddress

	stubErxApi := &erx.StubErxService{}

	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	// removing the accountId before sending it to the update handler because it should work without it even
	patient.AccountId = encoding.ObjectId{}

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := patient_file.NewDoctorPatientHandler(testData.DataApi, stubErxApi, stubAddressValidationService)

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&requestData{
			Patient: patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal patient object: " + err.Error())
	}

	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
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

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	// ensure that an update does not go through if we remove the patient address
	// or the dob or phone numbers
	signedupPatientResponse := SignupRandomTestPatient(t, testData)
	signedupPatientResponse.Patient.PhoneNumbers = nil
	stubErxApi := &erx.StubErxService{}
	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := patient_file.NewDoctorPatientHandler(testData.DataApi, stubErxApi, stubAddressValidationService)

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&requestData{
			Patient: signedupPatientResponse.Patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal patient object: " + err.Error())
	}

	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected a %d request due to remove of phone numbers, instead got %d", http.StatusBadRequest, resp.StatusCode)
	}

	signedupPatientResponse.Patient.PhoneNumbers = []*common.PhoneNumber{&common.PhoneNumber{
		Phone: "1241515",
		Type:  "Home",
	}}

	// now lets try no address
	resp, err = testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a failed request due to remove of phone address")
	}

	// now lets try no dob
	signedupPatientResponse.Patient.DOB = encoding.DOB{Month: 11, Day: 8, Year: 1987}
	resp, err = testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a failed request due to remove of date of birth")
	}

}

func TestDoctorUpdateToPhoneNumbers(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	patientVisitResponse, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     1234,
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patientPharmacy)
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
	patient.PatientAddress = patientAddress

	// lets go ahead and modify current phone number list

	phoneNumbers := []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone: "7348465522",
			Type:  "Home",
		},
		&common.PhoneNumber{
			Phone: "7348465522",
			Type:  "Work",
		},
		&common.PhoneNumber{
			Phone: "7348465522",
			Type:  "Work",
		},
	}
	patient.PhoneNumbers = phoneNumbers

	stubErxApi := &erx.StubErxService{}
	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}
	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := patient_file.NewDoctorPatientHandler(
		testData.DataApi,
		stubErxApi,
		stubAddressValidationService,
	)

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&requestData{
			Patient: patient,
		},
	)
	if err != nil {
		t.Fatal("Unable to marshal patient object: " + err.Error())
	}

	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get back patient information from database: " + err.Error())
	}

	if len(patient.PhoneNumbers) != len(phoneNumbers) {
		t.Fatal("Did not get back expected number of phone numbers for patient")
	}

	for i, phoneNumber := range phoneNumbers {
		if phoneNumber.Phone != patient.PhoneNumbers[i].Phone || phoneNumber.Type != patient.PhoneNumbers[i].Type {
			t.Fatal("Expected the phone numbers modified to be the same ones returned")
		}
	}
}

func TestDoctorUpdateToTopLevelInformation(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	patientVisitResponse, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     1234,
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patientPharmacy)
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
	patient.PatientAddress = patientAddress

	patient.FirstName = "Test"
	patient.LastName = "Test again"
	patient.Suffix = "m"
	patient.Prefix = "n"
	patient.MiddleName = "aaaa"
	patient.Gender = "Unknown"
	patient.DOB = encoding.DOB{Day: 11, Month: 9, Year: 1987}

	stubErxApi := &erx.StubErxService{}
	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}
	// lets go ahead and add this address to the patient and we should get back an address when we get the patient information
	doctorPatientHandler := patient_file.NewDoctorPatientHandler(
		testData.DataApi,
		stubErxApi,
		stubAddressValidationService,
	)

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&requestData{
			Patient: patient,
		},
	)
	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful call to update patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	updatedPatient, err := testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get back patient information from database: " + err.Error())
	}

	if patient.FirstName != updatedPatient.FirstName ||
		patient.LastName != updatedPatient.LastName ||
		patient.MiddleName != updatedPatient.MiddleName ||
		patient.Suffix != updatedPatient.Suffix ||
		patient.Prefix != updatedPatient.Prefix ||
		patient.DOB.Day != updatedPatient.DOB.Day ||
		patient.DOB.Year != updatedPatient.DOB.Year ||
		patient.DOB.Month != updatedPatient.DOB.Month {
		t.Fatal("Patient data incorrectly updated")
	}
}

func TestDoctorUpdatePatientInformationForbidden(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     1234,
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err := testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	signedupPatientResponse.Patient.PatientAddress = &common.Address{
		AddressLine1: "1234 Main Street",
		AddressLine2: "Apt 12345",
		City:         "San Francisco",
		State:        "California",
		ZipCode:      "94115",
	}

	stubAddressValidationService := address.StubAddressValidationService{
		CityStateToReturn: &address.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}
	doctorPatientHandler := patient_file.NewDoctorPatientHandler(
		testData.DataApi,
		&erx.StubErxService{},
		stubAddressValidationService,
	)

	ts := httptest.NewServer(doctorPatientHandler)
	defer ts.Close()

	jsonData, err := json.Marshal(
		&requestData{
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

	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successfull call to upte patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatal("Expected the doctor to be forbidden from updating the patient information given that it is not the patient's primary doctor but this was not the case")
	}

}

func TestDoctorPatientPharmacyUpdateHandler(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to read doctor information")
	}

	patientVisitResponse, _ := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     1234,
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	updatedPatientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     12345,
		AddressLine1: "1231515 Updated main street",
		AddressLine2: "124151515 apt",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115325151",
	}

	doctorUpdatePatientPharmacy := patient_file.NewDoctorUpdatePatientPharmacyHandler(testData.DataApi)
	ts := httptest.NewServer(doctorUpdatePatientPharmacy)
	defer ts.Close()

	requestData := &patient_file.DoctorUpdatePatientPharmacyRequestData{
		PatientId: patient.PatientId,
		Pharmacy:  updatedPatientPharmacy,
	}

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatal("Unable to marhsal data: " + err.Error())
	}

	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successfull call to update patient information")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Unable to make successful call to update patient information")
	}

	patient, err = testData.DataApi.GetPatientFromId(patient.PatientId.Int64())
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

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData)
	patientPharmacy := &pharmacy.PharmacyData{
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		SourceId:     1234,
		AddressLine1: "123456 main street",
		City:         "San Francisco",
		State:        "CA",
		Postal:       "94115",
	}

	err := testData.DataApi.UpdatePatientPharmacy(signedupPatientResponse.Patient.PatientId.Int64(), patientPharmacy)
	if err != nil {
		t.Fatal("Unable to add patient's preferred pharmacy")
	}

	doctorUpdatePatientPharmacy := patient_file.NewDoctorUpdatePatientPharmacyHandler(testData.DataApi)
	ts := httptest.NewServer(doctorUpdatePatientPharmacy)
	defer ts.Close()

	requestData := &patient_file.DoctorUpdatePatientPharmacyRequestData{
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

	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successfull call to upte patient information: " + err.Error())
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatal("Expected the doctor to be forbidden from updating the patient pharmacy information given that it is not the patient's primary doctor but this was not the case")
	}
}

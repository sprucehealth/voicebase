package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"strconv"

	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorPatientUpdateHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPatientUpdateHandlerRequestData struct {
	PatientId string `schema:"patient_id,required"`
}

func (d *DoctorPatientUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		d.getPatientInformation(w, r)
	case HTTP_PUT:
		d.updatePatientInformation(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type DoctorPatientUpdateHandlerResponse struct {
	Patient *common.Patient `json:"patient"`
}

func (d *DoctorPatientUpdateHandler) getPatientInformation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := DoctorPatientUpdateHandlerRequestData{}
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	currentDoctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor based on account id: "+err.Error())
		return
	}

	patientId, err := strconv.ParseInt(requestData.PatientId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse patient id: "+err.Error())
		return
	}

	careTeam, err := d.DataApi.GetCareTeamForPatient(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team based on patient id: "+err.Error())
		return
	}

	primaryDoctorId := getPrimaryDoctorIdFromCareTeam(careTeam)

	if currentDoctor.DoctorId.Int64() != primaryDoctorId {
		WriteDeveloperError(w, http.StatusForbidden, "Unable to get the patient information by doctor when this doctor is not the primary doctor for patient")
		return
	}

	patient, err := d.DataApi.GetPatientFromId(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient information from id: "+err.Error())
	}

	patient.Pharmacy, err = d.DataApi.GetPatientPharmacySelection(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get pharmacy selection for patient: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPatientUpdateHandlerResponse{Patient: patient})
}

func (d *DoctorPatientUpdateHandler) updatePatientInformation(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patient := new(common.Patient)
	if err := json.NewDecoder(r.Body).Decode(patient); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input body that is meant to be the patient object: "+err.Error())
		return
	}

	// avoid the doctor from making changes that would de-identify the patient
	if patient.FirstName == "" || patient.LastName == "" || patient.Dob.IsZero() || len(patient.PhoneNumbers) == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "Cannot remove first name, last name, date of birth or phone numbers")
		return
	}

	// TODO : Remove this once we have patient information intake
	// as a requirement
	if patient.PatientAddress == nil {
		patient.PatientAddress = &common.Address{
			AddressLine1: "1234 Main Street",
			AddressLine2: "Apt 12345",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94115",
		}
	}

	currentDoctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	// ensure that this doctor is the primary doctor of the patient
	careTeam, err := d.DataApi.GetCareTeamForPatient(patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient: "+err.Error())
		return
	}

	doctorId := getPrimaryDoctorIdFromCareTeam(careTeam)
	if doctorId != currentDoctor.DoctorId.Int64() {
		WriteDeveloperError(w, http.StatusForbidden, `Unable to move forward to update patient information since this doctor is not the primary doctor for the patient: `)
		return
	}

	// get the erx id for the patient, if it exists in the database
	existingPatientInfo, err := d.DataApi.GetPatientFromId(patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient info from database: "+err.Error())
		return
	}

	patient.ERxPatientId = existingPatientInfo.ERxPatientId

	// get patient's preferred pharmacy
	// TODO: Get patient pharmacy from the database once we start using surecsripts as our backing solution
	patientPreferredPharmacy, err := d.DataApi.GetPatientPharmacySelection(patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient's preferred pharmacy: "+err.Error())
		return
	}

	if patientPreferredPharmacy.Source != pharmacy.PHARMACY_SOURCE_SURESCRIPTS {
		patientPreferredPharmacy = &pharmacy.PharmacyData{
			SourceId:     "47731",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			AddressLine1: "1234 Main Street",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "94103",
		}
	}
	patient.Pharmacy = patientPreferredPharmacy

	if err := d.ErxApi.UpdatePatientInformation(currentDoctor.DoseSpotClinicianId, patient); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, `Unable to update patient information on dosespot. 
			Failing to avoid out of sync issues between surescripts and our platform `+err.Error())
		return
	}

	// update the doseSpot Id for the patient in our system now that we got one
	if existingPatientInfo.ERxPatientId == nil {
		if err := d.DataApi.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), patient.ERxPatientId.Int64()); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the patientId from dosespot: "+err.Error())
			return
		}
	}

	// go ahead and udpate the doctor's information in our system now that dosespot has it
	if err := d.DataApi.UpdatePatientInformationFromDoctor(patient); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient information on our databsae: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/address_validation"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"strconv"

	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
)

type DoctorPatientUpdateHandler struct {
	DataApi              api.DataAPI
	ErxApi               erx.ERxAPI
	AddressValidationApi address_validation.AddressValidationAPI
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

type DoctorPatientUpdateHandlerRequestResponse struct {
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

	patient, err := d.DataApi.GetPatientFromId(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient information from id: "+err.Error())
		return
	}

	if err := VerifyDoctorPatientRelationship(d.DataApi, currentDoctor, patient); err != nil {
		WriteDeveloperError(w, http.StatusForbidden, "Unable to verify doctor-patient relationship: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPatientUpdateHandlerRequestResponse{Patient: patient})
}

func (d *DoctorPatientUpdateHandler) updatePatientInformation(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &DoctorPatientUpdateHandlerRequestResponse{}
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input body that is meant to be the patient object: "+err.Error())
		return
	}

	// TODO : Remove this once we have patient information intake
	// as a requirement
	if requestData.Patient.PatientAddress == nil {
		requestData.Patient.PatientAddress = &common.Address{
			AddressLine1: "1234 Main Street",
			AddressLine2: "Apt 12345",
			City:         "San Francisco",
			State:        "California",
			ZipCode:      "94115",
		}
	}

	err := d.validatePatientInformationAccordingToSurescriptsRequirements(requestData.Patient, d.AddressValidationApi)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	// get the erx id for the patient, if it exists in the database
	existingPatientInfo, err := d.DataApi.GetPatientFromId(requestData.Patient.PatientId.Int64())
	if err != nil {
		WriteUserError(w, http.StatusBadRequest, err.Error())
		return
	}

	currentDoctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	if err := VerifyDoctorPatientRelationship(d.DataApi, currentDoctor, requestData.Patient); err != nil {
		WriteDeveloperError(w, http.StatusForbidden, "Unable to verify doctor-patient relationship: "+err.Error())
		return
	}

	requestData.Patient.ERxPatientId = existingPatientInfo.ERxPatientId

	// TODO: Get patient pharmacy from the database once we start using surecsripts as our backing solution
	if existingPatientInfo.Pharmacy == nil || existingPatientInfo.Pharmacy.Source != pharmacy.PHARMACY_SOURCE_SURESCRIPTS {
		existingPatientInfo.Pharmacy = &pharmacy.PharmacyData{
			SourceId:     "47731",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			AddressLine1: "1234 Main Street",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "94103",
		}
	}
	requestData.Patient.Pharmacy = existingPatientInfo.Pharmacy

	if err := d.ErxApi.UpdatePatientInformation(currentDoctor.DoseSpotClinicianId, requestData.Patient); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, `Unable to update patient information on dosespot. 
			Failing to avoid out of sync issues between surescripts and our platform `+err.Error())
		return
	}

	// update the doseSpot Id for the patient in our system now that we got one
	if !existingPatientInfo.ERxPatientId.IsValid {
		if err := d.DataApi.UpdatePatientWithERxPatientId(requestData.Patient.PatientId.Int64(), requestData.Patient.ERxPatientId.Int64()); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the patientId from dosespot: "+err.Error())
			return
		}
	}

	// go ahead and udpate the doctor's information in our system now that dosespot has it
	if err := d.DataApi.UpdatePatientInformationFromDoctor(requestData.Patient); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient information on our databsae: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

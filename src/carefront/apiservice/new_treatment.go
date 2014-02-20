package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"net/http"

	"github.com/gorilla/schema"
)

type NewTreatmentHandler struct {
	DataApi api.DataAPI
	ERxApi  erx.ERxAPI
}

type NewTreatmentRequestData struct {
	MedicationName     string `schema:"drug_internal_name,required"`
	MedicationStrength string `schema:"medication_strength,required"`
}

type NewTreatmentResponse struct {
	Treatment *common.Treatment `json:"treatment"`
}

func (m *NewTreatmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(NewTreatmentRequestData)
	err := schema.NewDecoder().Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctor, err := m.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	medication, err := m.ERxApi.SelectMedication(doctor.DoseSpotClinicianId, requestData.MedicationName, requestData.MedicationStrength)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get select medication: "+err.Error())
		return
	}

	if medication.IsControlledSubstance {
		WriteUserError(w, HTTP_UNPROCESSABLE_ENTITY, "Unfortunately, we do not support electronic routing of controlled substances using the platform. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
		return
	}

	newTreatmentResponse := &NewTreatmentResponse{
		Treatment: medication,
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, newTreatmentResponse)

	// TODO make sure to return the predefined additional instructions for the drug based on the drug name here.
}

package apiservice

import (
	"carefront/common"
	"carefront/libs/erx"
	"github.com/gorilla/schema"
	"net/http"
)

type NewTreatmentHandler struct {
	ERxApi erx.ERxAPI
}

type NewTreatmentRequestData struct {
	MedicationName     string `schema:"drug_internal_name,required"`
	MedicationStrength string `schema:"medication_strength,required"`
}

type NewTreatmentResponse struct {
	Treatment *common.Treatment `json:"treatment"`
}

func (m *NewTreatmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(NewTreatmentRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	medication, err := m.ERxApi.SelectMedication(requestData.MedicationName, requestData.MedicationStrength)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get select medication: "+err.Error())
		return
	}

	newTreatmentResponse := &NewTreatmentResponse{}
	newTreatmentResponse.Treatment = &common.Treatment{}
	newTreatmentResponse.Treatment.DrugDBIds = medication.DrugDBIds
	newTreatmentResponse.Treatment.DispenseUnitId = int64(medication.DispenseUnitId)
	newTreatmentResponse.Treatment.DispenseUnitDescription = medication.DispenseUnitDescription
	newTreatmentResponse.Treatment.OTC = medication.OTC
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, newTreatmentResponse)

	// TODO make sure to return the predefined additional instructions for the drug based on the drug name here.
}

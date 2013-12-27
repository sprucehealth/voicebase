package apiservice

import (
	"carefront/libs/erx"
	"github.com/gorilla/schema"
	"net/http"
)

type MedicationSelectHandler struct {
	ERxApi erx.ERxAPI
}

type MedicationSelectRequestData struct {
	MedicationName     string `schema:"drug_internal_name,required"`
	MedicationStrength string `schema:"medication_strength,required"`
}

type MedicationSelectResponse struct {
	Medication *erx.Medication `json:"medication"`
}

func (m *MedicationSelectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(MedicationSelectRequestData)
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

	medicationSelectResponse := &MedicationSelectResponse{}
	medicationSelectResponse.Medication = medication
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, medicationSelectResponse)
}

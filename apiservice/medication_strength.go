package apiservice

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/erx"
	"net/http"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
)

type MedicationStrengthSearchHandler struct {
	ERxApi  erx.ERxAPI
	DataApi api.DataAPI
}

type MedicationStrengthRequestData struct {
	MedicationName string `schema:"drug_internal_name,required"`
}

type MedicationStrengthSearchResponse struct {
	MedicationStrengths []string `json:"dosage_strength_options"`
}

func (m *MedicationStrengthSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(MedicationStrengthRequestData)
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

	medicationStrengths, err := m.ERxApi.SearchForMedicationStrength(doctor.DoseSpotClinicianId, requestData.MedicationName)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get medication strength results for given drug: "+err.Error())
		return
	}

	medicationStrengthResponse := &MedicationStrengthSearchResponse{MedicationStrengths: medicationStrengths}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, medicationStrengthResponse)
}

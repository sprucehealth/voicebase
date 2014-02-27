package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/pharmacy"
	"encoding/json"
	"net/http"
)

type DoctorUpdatePatientPharmacyHandler struct {
	DataApi api.DataAPI
}

type DoctorUpdatePatientPharmacyRequestData struct {
	PatientId *common.ObjectId       `json:"patient_id"`
	Pharmacy  *pharmacy.PharmacyData `json:"pharmacy"`
}

func (d *DoctorUpdatePatientPharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_PUT {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &DoctorUpdatePatientPharmacyRequestData{}
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patient, err := d.DataApi.GetPatientFromId(requestData.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from id: "+err.Error())
		return
	}

	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	careTeam, err := d.DataApi.GetCareTeamForPatient(patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient: "+err.Error())
		return
	}

	primaryDoctorId := getPrimaryDoctorIdFromCareTeam(careTeam)
	if primaryDoctorId != doctor.DoctorId.Int64() {
		WriteDeveloperError(w, http.StatusForbidden, "Unable to move forawrd to update patient's pharmacy address as this doctor is not the patient's primary doctor: ")
		return
	}

	if err := d.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), requestData.Pharmacy); err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient pharmacy by doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

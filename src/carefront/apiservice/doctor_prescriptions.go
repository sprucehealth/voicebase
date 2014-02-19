package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"
	"time"

	"github.com/gorilla/schema"
)

type DoctorPrescriptionsHandler struct {
	DataApi api.DataAPI
}

type DoctorPrescriptionsRequestData struct {
	FromTimeUnix int64 `schema:"from"`
	ToTimeUnix   int64 `schema:"to"`
}

type DoctorPrescriptionsResponse struct {
	TreatmentPlans []*common.TreatmentPlan `json:"treatment_plans"`
}

func (d *DoctorPrescriptionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData DoctorPrescriptionsRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// ensure that from and to date are specified
	if requestData.FromTimeUnix == 0 || requestData.ToTimeUnix == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "From and to times (in time since epoch) need to be specified!")
		return
	}

	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id: "+err.Error())
		return
	}

	fromTime := time.Unix(requestData.FromTimeUnix, 0)
	toTime := time.Unix(requestData.ToTimeUnix, 0)

	treatmentPlans, err := d.DataApi.GetCompletedPrescriptionsForDoctor(fromTime, toTime, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get completed prescriptions for doctor: "+err.Error())
		return
	}

	// find a list of unique patients for which to get information
	uniquePatientIdsBookKeeping := make(map[int64]bool)
	uniquePatientIds := make([]int64, 0)
	for _, treatmentPlan := range treatmentPlans {
		if !uniquePatientIdsBookKeeping[treatmentPlan.PatientId.Int64()] {
			uniquePatientIds = append(uniquePatientIds, treatmentPlan.PatientId.Int64())
			uniquePatientIdsBookKeeping[treatmentPlan.PatientId.Int64()] = true
		}
	}

	patients, err := d.DataApi.GetPatientsForIds(uniquePatientIds)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the patients based on ids: "+err.Error())
		return
	}

	pharmacies, err := d.DataApi.GetPharmacySelectionForPatients(uniquePatientIds)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacies for patients based on idsL "+err.Error())
		return
	}

	for _, pharmacySelection := range pharmacies {
		for _, patient := range patients {
			if patient.PatientId.Int64() == pharmacySelection.PatientId {
				patient.Pharmacy = pharmacySelection
			}
		}
	}

	for _, treatmentPlan := range treatmentPlans {
		for _, patient := range patients {
			if patient.PatientId == treatmentPlan.PatientId {
				treatmentPlan.PatientInfo = patient
			}
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionsResponse{TreatmentPlans: treatmentPlans})

}

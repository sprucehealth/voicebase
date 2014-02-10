package apiservice

import (
	"carefront/api"
	"carefront/common"
	"github.com/gorilla/schema"
	"net/http"
	"time"
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
	r.ParseForm()
	requestData := new(DoctorPrescriptionsRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
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
	uniquePatientIds := make(map[int64]bool)
	for _, treatmentPlan := range treatmentPlans {
		uniquePatientIds[treatmentPlan.PatientId] = true
	}

	// TODO: It's better to batch these queries into a single query based on the patientId as opposed to making 2 queries per patient for
	// patient information and pharmacy information
	patients := make([]*common.Patient, 0)
	for patientId, _ := range uniquePatientIds {
		patient, err := d.DataApi.GetPatientFromId(patientId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from id: "+err.Error())
			return
		}
		pharmacySelection, err := d.DataApi.GetPatientPharmacySelection(patientId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient's pharmacy selection : "+err.Error())
			return
		}
		patient.Pharmacy = pharmacySelection
		patients = append(patients, patient)
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

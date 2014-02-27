package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/pharmacy"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorPrescriptionErrorHandler struct {
	DataApi api.DataAPI
}

type DoctorPrescriptionErrorRequestData struct {
	TreatmentId string `schema:"treatment_id"`
}

type DoctorPrescriptionErrorResponse struct {
	Treatment *common.Treatment            `json:"treatment,omitempty"`
	Patient   *common.Patient              `json:"patient,omitempty"`
	Pharmacy  *pharmacy.PharmacyData       `json:"pharmacy,omitempty"`
	RxHistory []*common.PrescriptionStatus `json:"erx_history,omitempty"`
	Doctor    *common.Doctor               `json:"doctor,omitempty"`
}

func (d *DoctorPrescriptionErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &DoctorPrescriptionErrorRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	treatmentId, err := strconv.ParseInt(requestData.TreatmentId, 10, 64)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatmentId: "+err.Error())
		return
	}

	patient, err := d.DataApi.GetPatientFromTreatmentId(treatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on treatment id: "+err.Error())
		return
	}

	patient.Pharmacy, err = d.DataApi.GetPatientPharmacySelection(patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient's preferred pharmacy: "+err.Error())
		return
	}

	treatment, err := d.DataApi.GetTreatmentFromId(treatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on treatment id: "+err.Error())
		return
	}

	pharmacy, err := d.DataApi.GetPharmacyFromId(treatment.PharmacyLocalId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacy based on treatment information: "+err.Error())
		return
	}

	doctor, err := d.DataApi.GetDoctorFromTreatmentId(treatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from treatment id: "+err.Error())
		return
	}

	rxHistory, err := d.DataApi.GetPrescriptionStatusEventsForTreatment(treatmentId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get rx histor for treatment: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
		Treatment: treatment,
		Patient:   patient,
		Pharmacy:  pharmacy,
		RxHistory: rxHistory,
		Doctor:    doctor,
	})
}

package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorPatientTreatmentsHandler struct {
	DataApi api.DataAPI
}

type DoctorPatientTreatmentsRequestData struct {
	PatientId string `schema:"patient_id,required"`
}

type DoctorPatientTreatmentsResponse struct {
	Treatments             []*common.Treatment         `json:"treatments,omitempty"`
	UnlinkedDNTFTreatments []*common.Treatment         `json:"unlinked_dntf_treatments,omitempty"`
	RefillRequests         []*common.RefillRequestItem `json:"refill_requests,omitempty"`
}

func (d *DoctorPatientTreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request parameters: "+err.Error())
		return
	}

	requestData := DoctorPatientTreatmentsRequestData{}
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
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on id: "+err.Error())
		return
	}

	if !patient.IsUnlinked {
		careTeam, err := d.DataApi.GetCareTeamForPatient(patientId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team based on patient id: "+err.Error())
			return
		}

		primaryDoctorId := GetPrimaryDoctorIdFromCareTeam(careTeam)

		if currentDoctor.DoctorId.Int64() != primaryDoctorId {
			WriteDeveloperError(w, http.StatusForbidden, "Unable to get the patient information by doctor when this doctor is not the primary doctor for patient")
			return
		}
	}

	treatments, err := d.DataApi.GetTreatmentsForPatient(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments for patient: "+err.Error())
		return
	}

	refillRequests, err := d.DataApi.GetRefillRequestsForPatient(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill requests for patient: "+err.Error())
		return
	}

	unlinkedDNTFTreatments, err := d.DataApi.GetUnlinkedDNTFTreatmentsForPatient(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get unlinked dntf treatments for patient: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPatientTreatmentsResponse{
		Treatments:             treatments,
		RefillRequests:         refillRequests,
		UnlinkedDNTFTreatments: unlinkedDNTFTreatments,
	})
}

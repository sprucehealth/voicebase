package apiservice

import (
	"carefront/api"
	"carefront/common"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorPrescriptionErrorHandler struct {
	DataApi api.DataAPI
}

type DoctorPrescriptionErrorRequestData struct {
	TreatmentId             string `schema:"treatment_id"`
	UnlinkedDNTFTreatmentId string `schema:"unlinked_dntf_treatment_id"`
}

type DoctorPrescriptionErrorResponse struct {
	Treatment *common.Treatment `json:"treatment,omitempty"`
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

	if requestData.TreatmentId != "" {
		treatmentId, err := strconv.ParseInt(requestData.TreatmentId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatmentId: "+err.Error())
			return
		}

		treatment, err := d.DataApi.GetTreatmentFromId(treatmentId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on treatment id: "+err.Error())
			return
		}

		if err := verifyDoctorPatientRelationship(d.DataApi, treatment.Doctor, treatment.Patient); err != nil {
			WriteDeveloperError(w, http.StatusForbidden, "Unable to verify patient-doctor relationship: "+err.Error())
			return
		}

		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
			Treatment: treatment,
		})
	} else if requestData.UnlinkedDNTFTreatmentId != "" {
		treatmentId, err := strconv.ParseInt(requestData.UnlinkedDNTFTreatmentId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatmentId: "+err.Error())
			return
		}

		treatment, err := d.DataApi.GetUnlinkedDNTFTreatment(treatmentId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on treatment id: "+err.Error())
			return
		}

		if err := verifyDoctorPatientRelationship(d.DataApi, treatment.Doctor, treatment.Patient); err != nil {
			WriteDeveloperError(w, http.StatusForbidden, "Unable to verify patient-doctor relationship: "+err.Error())
			return
		}

		WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
			Treatment: treatment,
		})
	}
}

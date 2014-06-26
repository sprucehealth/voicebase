package apiservice

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
)

type DoctorPrescriptionErrorHandler struct {
	DataApi api.DataAPI
}

type DoctorPrescriptionErrorRequestData struct {
	TreatmentId             string `schema:"treatment_id"`
	UnlinkedDNTFTreatmentId string `schema:"unlinked_dntf_treatment_id"`
}

type DoctorPrescriptionErrorResponse struct {
	Treatment             *common.Treatment `json:"treatment,omitempty"`
	UnlinkedDNTFTreatment *common.Treatment `json:"unlinked_dntf_treatment,omitempty"`
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

	var treatment *common.Treatment
	if requestData.TreatmentId != "" {
		treatmentId, err := strconv.ParseInt(requestData.TreatmentId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatmentId: "+err.Error())
			return
		}

		treatment, err = d.DataApi.GetTreatmentFromId(treatmentId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on treatment id: "+err.Error())
			return
		}
	} else if requestData.UnlinkedDNTFTreatmentId != "" {
		treatmentId, err := strconv.ParseInt(requestData.UnlinkedDNTFTreatmentId, 10, 64)
		if err != nil {
			WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatmentId: "+err.Error())
			return
		}

		treatment, err = d.DataApi.GetUnlinkedDNTFTreatment(treatmentId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on treatment id: "+err.Error())
			return
		}
	}

	if treatment != nil && !treatment.Patient.IsUnlinked {
		if err := ValidateDoctorAccessToPatientFile(treatment.Doctor.DoctorId.Int64(), treatment.PatientId.Int64(), d.DataApi); err != nil {
			WriteError(err, w, r)
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
		Treatment: treatment,
	})

}

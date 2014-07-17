package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type DoctorPrescriptionErrorHandler struct {
	DataApi api.DataAPI
}

type DoctorPrescriptionErrorRequestData struct {
	TreatmentId             int64 `schema:"treatment_id"`
	UnlinkedDNTFTreatmentId int64 `schema:"unlinked_dntf_treatment_id"`
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

	requestData := &DoctorPrescriptionErrorRequestData{}
	if err := DecodeRequestData(requestData, r); err != nil {
		WriteError(err, w, r)
		return
	}

	var treatment *common.Treatment
	var err error
	if requestData.TreatmentId != 0 {
		treatment, err = d.DataApi.GetTreatmentFromId(requestData.TreatmentId)
		if err != nil {
			WriteError(err, w, r)
			return
		}
	} else if requestData.UnlinkedDNTFTreatmentId != 0 {
		treatment, err = d.DataApi.GetUnlinkedDNTFTreatment(requestData.UnlinkedDNTFTreatmentId)
		if err != nil {
			WriteError(err, w, r)
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

package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type doctorPrescriptionErrorHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorPrescriptionErrorHandler(dataAPI api.DataAPI) http.Handler {
	return &doctorPrescriptionErrorHandler{
		dataAPI: dataAPI,
	}
}

type DoctorPrescriptionErrorRequestData struct {
	TreatmentId             int64 `schema:"treatment_id"`
	UnlinkedDNTFTreatmentId int64 `schema:"unlinked_dntf_treatment_id"`
}

type DoctorPrescriptionErrorResponse struct {
	Treatment             *common.Treatment `json:"treatment,omitempty"`
	UnlinkedDNTFTreatment *common.Treatment `json:"unlinked_dntf_treatment,omitempty"`
}

func (d *doctorPrescriptionErrorHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := GetContext(r)

	requestData := &DoctorPrescriptionErrorRequestData{}
	if err := DecodeRequestData(requestData, r); err != nil {
		return false, NewValidationError(err.Error(), r)
	}

	var treatment *common.Treatment
	var err error
	if requestData.TreatmentId != 0 {
		treatment, err = d.dataAPI.GetTreatmentFromId(requestData.TreatmentId)
		if err != nil {
			return false, err
		}
	} else if requestData.UnlinkedDNTFTreatmentId != 0 {
		treatment, err = d.dataAPI.GetUnlinkedDNTFTreatment(requestData.UnlinkedDNTFTreatmentId)
		if err != nil {
			return false, err
		}
	}

	if treatment != nil && !treatment.Patient.IsUnlinked {
		if err := ValidateDoctorAccessToPatientFile(treatment.Doctor.DoctorId.Int64(), treatment.PatientId.Int64(), d.dataAPI); err != nil {
			return false, err
		}
	}

	ctxt.RequestCache[RequestData] = requestData
	ctxt.RequestCache[Treatment] = treatment

	return true, nil
}

func (d *doctorPrescriptionErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	treatment := GetContext(r).RequestCache[Treatment]
	if treatment == nil {
		WriteResourceNotFoundError("no treatment found", w, r)
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
		Treatment: treatment.(*common.Treatment),
	})

}

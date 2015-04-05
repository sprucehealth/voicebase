package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type prescriptionErrorHandler struct {
	dataAPI api.DataAPI
}

func NewPrescriptionErrorHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.AuthorizationRequired(&prescriptionErrorHandler{
		dataAPI: dataAPI,
	})
}

type DoctorPrescriptionErrorRequestData struct {
	TreatmentID             int64 `schema:"treatment_id"`
	UnlinkedDNTFTreatmentID int64 `schema:"unlinked_dntf_treatment_id"`
}

type DoctorPrescriptionErrorResponse struct {
	Treatment             *common.Treatment `json:"treatment,omitempty"`
	UnlinkedDNTFTreatment *common.Treatment `json:"unlinked_dntf_treatment,omitempty"`
}

func (d *prescriptionErrorHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &DoctorPrescriptionErrorRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	var treatment *common.Treatment
	var err error
	if requestData.TreatmentID != 0 {
		treatment, err = d.dataAPI.GetTreatmentFromID(requestData.TreatmentID)
		if err != nil {
			return false, err
		}
	} else if requestData.UnlinkedDNTFTreatmentID != 0 {
		treatment, err = d.dataAPI.GetUnlinkedDNTFTreatment(requestData.UnlinkedDNTFTreatmentID)
		if err != nil {
			return false, err
		}
	}

	if treatment != nil {
		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, treatment.Doctor.DoctorID.Int64(), treatment.PatientID.Int64(), d.dataAPI); err != nil {
			return false, err
		}
	}

	ctxt.RequestCache[apiservice.RequestData] = requestData
	ctxt.RequestCache[apiservice.Treatment] = treatment

	return true, nil
}

func (d *prescriptionErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != httputil.Get {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	treatment := apiservice.GetContext(r).RequestCache[apiservice.Treatment]
	if treatment == nil {
		apiservice.WriteResourceNotFoundError("no treatment found", w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
		Treatment: treatment.(*common.Treatment),
	})
}

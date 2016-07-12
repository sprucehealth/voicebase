package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type prescriptionErrorHandler struct {
	dataAPI api.DataAPI
}

func NewPrescriptionErrorHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&prescriptionErrorHandler{
				dataAPI: dataAPI,
			})),
		httputil.Get)
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
	requestCache := apiservice.MustCtxCache(r.Context())
	account := apiservice.MustCtxAccount(r.Context())

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
		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, account.Role, treatment.Doctor.ID.Int64(), treatment.PatientID, d.dataAPI); err != nil {
			return false, err
		}
	}

	requestCache[apiservice.CKRequestData] = requestData
	requestCache[apiservice.CKTreatment] = treatment

	return true, nil
}

func (d *prescriptionErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	treatment := apiservice.MustCtxCache(r.Context())[apiservice.CKTreatment]
	if treatment == nil {
		apiservice.WriteResourceNotFoundError("no treatment found", w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
		Treatment: treatment.(*common.Treatment),
	})
}

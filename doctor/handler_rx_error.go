package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type prescriptionErrorHandler struct {
	dataAPI api.DataAPI
}

func NewPrescriptionErrorHandler(dataAPI api.DataAPI) httputil.ContextHandler {
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

func (d *prescriptionErrorHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

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

func (d *prescriptionErrorHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	treatment := apiservice.MustCtxCache(ctx)[apiservice.CKTreatment]
	if treatment == nil {
		apiservice.WriteResourceNotFoundError(ctx, "no treatment found", w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorPrescriptionErrorResponse{
		Treatment: treatment.(*common.Treatment),
	})
}

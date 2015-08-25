package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type doctorTokensHandler struct {
	dataAPI api.DataAPI
}

type doctorTokensRequest struct {
	PatientID common.PatientID `schema:"patient_id,required"`
}

type tokenItem struct {
	Token       string `json:"token"`
	Replaced    string `json:"replaced"`
	Description string `json:"description"`
}

type doctorTokensResponse struct {
	Tokens []*tokenItem `json:"tokens"`
}

func NewDoctorTokensHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(&doctorTokensHandler{
					dataAPI: dataAPI,
				})),
			api.RoleDoctor,
		),
		httputil.Get)
}

func (d *doctorTokensHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	requestCache := apiservice.MustCtxCache(ctx)

	requestData := &doctorTokensRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if requestData.PatientID.Int64() == 0 {
		return false, apiservice.NewValidationError("patient_id is required")
	}
	requestCache[apiservice.CKRequestData] = requestData

	doctor, err := d.dataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctor] = doctor

	patient, err := d.dataAPI.Patient(requestData.PatientID, true)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method,
		account.Role,
		doctor.ID.Int64(),
		patient.ID,
		d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *doctorTokensHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patient := requestCache[apiservice.CKPatient].(*common.Patient)
	doctor := requestCache[apiservice.CKDoctor].(*common.Doctor)

	t := newPatientDoctorTokenizer(patient, doctor)
	res := &doctorTokensResponse{
		Tokens: make([]*tokenItem, 0, len(t.tokens)),
	}

	for _, tItem := range t.tokens {
		res.Tokens = append(res.Tokens, &tokenItem{
			Token:       string(t.startDelimiter) + string(tItem.tokenType) + string(t.endDelimiter),
			Replaced:    tItem.replacer,
			Description: tItem.description,
		})
	}

	httputil.JSONResponse(w, http.StatusOK, res)
}

package patient

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type applePayHandler struct {
	dataAPI              api.DataAPI
	paymentAPI           apiservice.StripeClient
	addressValidationAPI address.AddressValidationAPI
	dispatcher           *dispatch.Dispatcher
}

type ApplePayRequest struct {
	VisitID int64       `json:"patient_visit_id,string"`
	Card    common.Card `json:"apple_pay_card"`
}

func NewApplePayHandler(dataAPI api.DataAPI, paymentAPI apiservice.StripeClient, addressValidationAPI address.AddressValidationAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return &applePayHandler{
		dataAPI:              dataAPI,
		paymentAPI:           paymentAPI,
		addressValidationAPI: addressValidationAPI,
		dispatcher:           dispatcher,
	}
}

func (h *applePayHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctx := apiservice.GetContext(r)
	if ctx.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (h *applePayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	var req ApplePayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	req.Card.ApplePay = true
	req.Card.IsDefault = false

	patient, err := h.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err == api.NoRowsError {
		apiservice.WriteResourceNotFoundError("no patient found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := addCardForPatient(r, h.dataAPI, h.paymentAPI, h.addressValidationAPI, &req.Card, patient); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	visit, err := submitVisit(r, h.dataAPI, h.dispatcher, patient, req.VisitID, req.Card.ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &PatientVisitSubmittedResponse{
		PatientVisitId: visit.PatientVisitId.Int64(),
		Status:         visit.Status,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

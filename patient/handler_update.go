package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/surescripts"
)

// UpdateHandler handles requests related to patient record updates
type UpdateHandler struct {
	dataAPI          api.DataAPI
	addressValidator address.Validator
}

// PhoneNumber represents the valid forms of phone number input from the client
type PhoneNumber struct {
	Type   string `json:"phone_type,omitempty"`
	Number string `json:"phone"`
}

// UpdateRequest represents the expected data associated with a successful PUT request
type UpdateRequest struct {
	PhoneNumbers []PhoneNumber   `json:"phone_numbers"`
	Address      *common.Address `json:"address"`
}

func (r *UpdateRequest) isZero() bool {
	return (r == nil || (len(r.PhoneNumbers) == 0 && r.Address == nil))
}

func (r *UpdateRequest) transformRequestToUpdate(dataAPI api.DataAPI, validator address.Validator) (*api.PatientUpdate, error) {
	var update api.PatientUpdate
	var err error

	if len(r.PhoneNumbers) > 0 {
		update.PhoneNumbers, err = transformPhoneNumbers(r.PhoneNumbers)
		if err != nil {
			return nil, err
		}
	}

	if r.Address != nil {
		if err := surescripts.ValidateAddress(r.Address, validator, dataAPI); err != nil {
			return nil, err
		}

		update.Address = r.Address
	}

	return &update, nil
}

// NewUpdateHandler returns an initialized instance of UpdateHandler
func NewUpdateHandler(dataAPI api.DataAPI, addressValidator address.Validator) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&UpdateHandler{
				dataAPI:          dataAPI,
				addressValidator: addressValidator,
			}),
			api.RolePatient,
		), httputil.Post, httputil.Put)
}

func (h *UpdateHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}

	req := &UpdateRequest{}
	if err := apiservice.DecodeRequestData(req, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	patientID, err := h.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// TODO: implement DELETE
	switch r.Method {
	case "POST", "PUT":
		// For now treat these the same because we don't support more than one phone number
		// for the patient which is the only this this endpoint currently supports.
		h.postOrPUT(ctx, w, r, patientID, req)
	}
}

func (h *UpdateHandler) postOrPUT(ctx context.Context, w http.ResponseWriter, r *http.Request, patientID common.PatientID, req *UpdateRequest) {
	if req.isZero() {
		apiservice.WriteJSONSuccess(w)
		return
	}

	update, err := req.transformRequestToUpdate(h.dataAPI, h.addressValidator)
	if err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	if err := h.dataAPI.UpdatePatient(patientID, update, false); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

func transformPhoneNumbers(pn []PhoneNumber) ([]*common.PhoneNumber, error) {
	var numbers []*common.PhoneNumber
	for _, phone := range pn {
		num, err := common.ParsePhone(phone.Number)
		if err != nil {
			return nil, err
		}
		phoneNumberType, err := common.ParsePhoneNumberType(phone.Type)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, &common.PhoneNumber{
			Phone: num,
			Type:  phoneNumberType,
		})
	}
	return numbers, nil
}

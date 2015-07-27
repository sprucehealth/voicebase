package home

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type signUpAPIHAndler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type signUpAPIRequest struct {
	Email       string        `json:"email"`
	Password    string        `json:"password"`
	State       string        `json:"state"`
	FirstName   string        `json:"first_name"`
	LastName    string        `json:"last_name"`
	DOB         encoding.Date `json:"dob"`
	Gender      string        `json:"gender"`
	MobilePhone common.Phone  `json:"mobile_phone"`
}

type signUpAPIResponse struct{}

func (r *signUpAPIRequest) Validate(states []*common.State) (bool, string) {
	if r.Email == "" {
		return false, "Email is required"
	}
	if !email.IsValidEmail(r.Email) {
		return false, "The email provided is invalid"
	}
	if r.Password == "" {
		return false, "Password is required"
	}
	// TODO: for now prevent anyone under 18 from using this endpoint
	if r.DOB.Age() < 18 {
		return false, "Must be over 18 or over"
	}
	if r.State == "" {
		return false, "State is required"
	}
	r.State = strings.ToUpper(r.State)
	var validState bool
	for _, s := range states {
		if r.State == s.Abbreviation {
			validState = true
			break
		}
	}
	if !validState {
		return false, "A valid US state is required"
	}
	if r.FirstName == "" {
		return false, "First name is required"
	}
	if r.LastName == "" {
		return false, "Last name is required"
	}
	if r.Gender != "male" && r.Gender != "female" {
		return false, "Gender is required"
	}
	return true, ""
}

func newSignUpAPIHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&signUpAPIHAndler{
		dataAPI: dataAPI,
		authAPI: authAPI,
	}, httputil.Post)
}

func (h *signUpAPIHAndler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req signUpAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	states, err := h.dataAPI.AvailableStates()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	if ok, reason := req.Validate(states); !ok {
		www.APIGeneralError(w, r, "invalid_request", reason)
		return
	}

	accountID, err := h.authAPI.CreateAccount(req.Email, req.Password, api.RolePatient)
	if err == api.ErrLoginAlreadyExists {
		// TODO: should check here if there's the account role is PATIENT but there's no
		// patient attached to the account. This can happen because the creation of an account
		// and the creation of a patient don't happen in the same transaction. Should allow
		// the creation to proceed if that's the case.
		httputil.JSONResponse(w, www.HTTPStatusAPIError, &www.APIErrorResponse{
			Error: www.APIError{
				Message: "An account already exists with the provided email address",
				Type:    "account_exists",
			},
		})
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		DOB:       req.DOB,
		Gender:    req.Gender,
		PhoneNumbers: []*common.PhoneNumber{
			{
				Phone:    req.MobilePhone,
				Type:     api.PhoneCell,
				Status:   api.StatusActive,
				Verified: false,
			},
		},
	}
	if err := h.dataAPI.RegisterPatient(patient); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	token, err := h.authAPI.CreateToken(accountID, api.Web, 0)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	http.SetCookie(w, www.NewAuthCookie(token, r))
	httputil.JSONResponse(w, http.StatusOK, signUpAPIResponse{})
}

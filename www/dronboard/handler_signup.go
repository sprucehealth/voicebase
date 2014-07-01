package dronboard

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
	"github.com/sprucehealth/backend/www"
)

type signupHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type signupRequest struct {
	FirstName  string
	LastName   string
	Email      string
	CellNumber string
	Password1  string
	Password2  string
}

// Validate returns an error message for each field that doesn't match. If
// the request has no validation errors then the size of the map is 0.
func (r *signupRequest) Validate() map[string]string {
	errors := map[string]string{}
	if r.FirstName == "" {
		errors["FirstName"] = "FirstName is required"
	}
	if r.LastName == "" {
		errors["LastName"] = "LastName is required"
	}
	if r.Email == "" {
		errors["Email"] = "Email is required"
	} else if !validator.IsValidEmail(r.Email) {
		errors["Email"] = "Email does not appear to be valid"
	}
	if r.CellNumber == "" {
		errors["CellNumber"] = "Cell phone number is required"
	}
	if len(r.Password1) < api.MinimumPasswordLength {
		errors["Password1"] = fmt.Sprintf("Password must be a minimum of %d characters", api.MinimumPasswordLength)
	} else if r.Password1 != r.Password2 {
		errors["Password2"] = "Passwords do not match"
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}

func NewSignupHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) http.Handler {
	return www.SupportedMethodsFilter(&signupHandler{
		dataAPI: dataAPI,
		authAPI: authAPI,
	}, []string{"GET", "POST"})
}

func (h *signupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	var errors map[string]string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		if err := schema.NewDecoder().Decode(&req, r.PostForm); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		errors = req.Validate()
		if len(errors) == 0 {
			accountID, token, err := h.authAPI.SignUp(req.Email, req.Password1, api.DOCTOR_ROLE)
			if err == api.LoginAlreadyExists {
				errors = map[string]string{
					"Email": "An account with the provided email already exists.",
				}
			} else if err != nil {
				www.InternalServerError(w, r, err)
				return
			} else {
				// TODO
				_ = token
				doctor := &common.Doctor{
					AccountId: encoding.NewObjectId(accountID),
					FirstName: req.FirstName,
					LastName:  req.LastName,
				}
				if _, err := h.dataAPI.RegisterDoctor(doctor); err != nil {
					www.InternalServerError(w, r, err)
					return
				}
			}
		}
	}

	www.TemplateResponse(w, http.StatusOK, signupTemplate, &www.BaseTemplateContext{
		Title: "Doctor Sign Up | Spruce",
		SubContext: &signupTemplateContext{
			Form:       &req,
			FormErrors: errors,
		},
	})
}

// func (h *signupHandler) signUpDoctor(w http.ResponseWriter, r *http.Request, req *signupRequest) {
// 	// h.dataAPI.RegisterDoctor(doctor)
// }

package dronboard

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

var (
	dobSeparators = []rune{'-', '/'}
)

type signupHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type signupRequest struct {
	FirstName  string
	LastName   string
	Gender     string
	DOB        string
	Email      string
	CellNumber string
	Password1  string
	Password2  string

	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	ZipCode      string

	dob encoding.DOB
}

// Validate returns an error message for each field that doesn't match. If
// the request has no validation errors then the size of the map is 0.
func (r *signupRequest) Validate() map[string]string {
	errors := map[string]string{}
	if r.FirstName == "" {
		errors["FirstName"] = "First name is required"
	}
	if r.LastName == "" {
		errors["LastName"] = "Last name is required"
	}
	if r.Gender == "" {
		errors["Gender"] = "Gender is required"
	}
	if r.DOB == "" {
		errors["DOB"] = "Date of birth is required"
	} else {
		// Browsers supporting HTML5 forms will return YYYY-MM-DD, but otherwrise
		// the field is treated as text and people will enter MM-DD-YYY. Support
		// both formats since there's no chance they'll collide.
		dob, err := encoding.ParseDOB(r.DOB, "YMD", dobSeparators)
		if err != nil {
			dob, err = encoding.ParseDOB(r.DOB, "MDY", dobSeparators)
		}
		if err != nil {
			errors["DOB"] = "Date of birth is invalid"
		} else {
			r.dob = dob
		}
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
	if r.AddressLine1 == "" {
		errors["AddressLine1"] = "Address is required"
	}
	if r.City == "" {
		errors["City"] = "City is required"
	}
	if r.State == "" {
		errors["State"] = "State is required"
	}
	if r.ZipCode == "" {
		errors["ZipCode"] = "ZipCode is required"
	}
	return errors
}

func NewSignupHandler(router *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI) http.Handler {
	return www.SupportedMethodsHandler(&signupHandler{
		router:  router,
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
				address := &common.Address{
					AddressLine1: req.AddressLine1,
					AddressLine2: req.AddressLine2,
					City:         req.City,
					State:        req.State,
					ZipCode:      req.ZipCode,
					Country:      "USA",
				}
				doctor := &common.Doctor{
					AccountId:     encoding.NewObjectId(accountID),
					FirstName:     req.FirstName,
					LastName:      req.LastName,
					DOB:           req.dob,
					Gender:        req.Gender,
					CellPhone:     req.CellNumber,
					DoctorAddress: address,
				}

				// TODO ??? DoseSpotClinicianId int64

				if _, err := h.dataAPI.RegisterDoctor(doctor); err != nil {
					www.InternalServerError(w, r, err)
					return
				}

				http.SetCookie(w, www.NewAuthCookie(token, r))
				if u, err := h.router.Get("doctor-register-credentials").URLPath(); err != nil {
					www.InternalServerError(w, r, err)
				} else {
					http.Redirect(w, r, u.String(), http.StatusSeeOther)
				}
				return
			}
		}
	}

	states, err := h.dataAPI.ListStates()
	if err != nil {
		www.InternalServerError(w, r, err)
	}

	www.TemplateResponse(w, http.StatusOK, signupTemplate, &www.BaseTemplateContext{
		Title: "Doctor Registration | Spruce",
		SubContext: &signupTemplateContext{
			Form:       &req,
			FormErrors: errors,
			States:     states,
		},
	})
}

// func (h *signupHandler) signUpDoctor(w http.ResponseWriter, r *http.Request, req *signupRequest) {
// 	// h.dataAPI.RegisterDoctor(doctor)
// }

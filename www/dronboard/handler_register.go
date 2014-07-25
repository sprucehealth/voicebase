package dronboard

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/dchest/validator"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

var (
	dobSeparators = []rune{'-', '/'}
)

type registerHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	authAPI  api.AuthAPI
	signer   *common.Signer
	nextStep string
}

type registerForm struct {
	FirstName  string
	LastName   string
	Gender     string
	DOB        string
	Email      string
	CellNumber string
	Password1  string
	Password2  string
	// Address
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	ZipCode      string
	// Engagement
	HoursPerWeek string
	TimesActive  string
	JacketSize   string
	Excitement   string
	// Legal
	EBusinessAgree bool

	dob encoding.DOB
}

// Validate returns an error message for each field that doesn't match. If
// the request has no validation errors then the size of the map is 0.
func (r *registerForm) Validate() map[string]string {
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
	if !r.EBusinessAgree {
		errors["EBusinessAgree"] = "Must agree to communicate electronically"
	}
	return errors
}

func NewRegisterHandler(router *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, signer *common.Signer) http.Handler {
	return httputil.SupportedMethods(&registerHandler{
		router:   router,
		dataAPI:  dataAPI,
		authAPI:  authAPI,
		signer:   signer,
		nextStep: "doctor-register-credentials",
	}, []string{"GET", "POST"})
}

func (h *registerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !validateRequestSignature(h.signer, r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var form registerForm
	var errors map[string]string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		if err := schema.NewDecoder().Decode(&form, r.PostForm); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		errors = form.Validate()
		if len(errors) == 0 {
			accountID, token, err := h.authAPI.SignUp(form.Email, form.Password1, api.DOCTOR_ROLE)
			if err == api.LoginAlreadyExists {
				errors = map[string]string{
					"Email": "An account with the provided email already exists.",
				}
			} else if err != nil {
				www.InternalServerError(w, r, err)
				return
			} else {
				address := &common.Address{
					AddressLine1: form.AddressLine1,
					AddressLine2: form.AddressLine2,
					City:         form.City,
					State:        form.State,
					ZipCode:      form.ZipCode,
					Country:      "USA",
				}
				doctor := &common.Doctor{
					AccountId:        encoding.NewObjectId(accountID),
					FirstName:        form.FirstName,
					LastName:         form.LastName,
					ShortDisplayName: fmt.Sprintf("Dr. %s", form.LastName),
					LongDisplayName:  fmt.Sprintf("Dr. %s %s", form.FirstName, form.LastName),
					DOB:              form.dob,
					Gender:           form.Gender,
					CellPhone:        form.CellNumber,
					DoctorAddress:    address,
				}

				doctorID, err := h.dataAPI.RegisterDoctor(doctor)
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}

				attributes := map[string]string{
					api.AttrHoursUsingSprucePerWeek: form.HoursPerWeek,
					api.AttrTimesActiveOnSpruce:     form.TimesActive,
					api.AttrJacketSize:              form.JacketSize,
					api.AttrExcitedAboutSpruce:      form.Excitement,
					api.AttrEBusinessAgreement:      "true",
				}
				if err := h.dataAPI.UpdateDoctorAttributes(doctorID, attributes); err != nil {
					www.InternalServerError(w, r, err)
					return
				}

				http.SetCookie(w, www.NewAuthCookie(token, r))
				if u, err := h.router.Get(h.nextStep).URLPath(); err != nil {
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

	www.TemplateResponse(w, http.StatusOK, registerTemplate, &www.BaseTemplateContext{
		Title: "Doctor Registration | Spruce",
		SubContext: &registerTemplateContext{
			Form:       &form,
			FormErrors: errors,
			States:     states,
		},
	})
}

func validateRequestSignature(signer *common.Signer, r *http.Request) bool {
	sig, err := base64.StdEncoding.DecodeString(r.FormValue("s"))
	if err != nil {
		return false
	}
	expires, err := strconv.ParseInt(r.FormValue("e"), 10, 64)
	if err != nil {
		return false
	}
	nonce := r.FormValue("n")
	now := time.Now().UTC().Unix()
	if nonce == "" || len(sig) == 0 || expires <= now {
		return false
	}
	msg := []byte(fmt.Sprintf("expires=%d&nonce=%s", expires, nonce))
	return signer.Verify(msg, sig)
}

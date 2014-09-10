package dronboard

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
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
	template *template.Template
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

	dob        encoding.DOB
	cellNumber common.Phone
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
	} else if !email.IsValidEmail(r.Email) {
		errors["Email"] = "Email does not appear to be valid"
	}
	if r.CellNumber == "" {
		errors["CellNumber"] = "Cell phone number is required"
	} else {
		cellNumber, err := common.ParsePhone(r.CellNumber)
		if err != nil {
			errors["CellNumber"] = err.Error()
		}
		r.cellNumber = cellNumber
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

func NewRegisterHandler(router *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, signer *common.Signer, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&registerHandler{
		router:   router,
		dataAPI:  dataAPI,
		authAPI:  authAPI,
		signer:   signer,
		template: templateLoader.MustLoadTemplate("dronboard/register.html", "dronboard/base.html", nil),
		nextStep: "doctor-register-saved-message",
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
			accountID, err := h.authAPI.CreateAccount(form.Email, form.Password1, api.DOCTOR_ROLE)
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
					CellPhone:        form.cellNumber,
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

				token, err := h.authAPI.CreateToken(accountID, api.Web, api.RegularAuth)
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}

				if environment.IsProd() {
					if err := registerDoctorInDemo(r); err != nil {
						golog.Errorf("Unable to register doctor in demo environment: %s", err)
					}
				}

				http.SetCookie(w, www.NewAuthCookie(token, r))
				if u, err := h.router.Get(h.nextStep).URLPath(); err != nil {
					www.InternalServerError(w, r, err)
				} else {
					http.Redirect(w, r, u.String(), http.StatusSeeOther)
				}

				dispatch.Default.Publish(&DoctorRegisteredEvent{
					DoctorID: doctorID,
				})

				return
			}
		}
	}

	states, err := h.dataAPI.ListStates()
	if err != nil {
		www.InternalServerError(w, r, err)
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Doctor Registration | Spruce",
		SubContext: &struct {
			Form       *registerForm
			FormErrors map[string]string
			States     []*common.State
		}{
			Form:       &form,
			FormErrors: errors,
			States:     states,
		},
	})
}

// registerDoctorInDemo essentially makes a call to the demo environment
// to register the same doctor so that we can have the doctor use the same credentials
// to login and go through training cases. This is more of a hack in that if the doctor account
// already exists in the demo environment then this wont work.
func registerDoctorInDemo(r *http.Request) error {
	req, err := http.NewRequest("POST",
		"https://demo-www.carefront.net/doctor-register?e=1851894319&n=cMsSRH243pE%3D&s=SgGxU3kYg2s66v4BIiyIpeF2SzY%3D",
		strings.NewReader(r.PostForm.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		golog.Errorf("Error making request to register doctor on the demo portal: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		golog.Errorf("Unable to make successful request to register doctor on demo portal: %d", resp.StatusCode)
	}

	return nil
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

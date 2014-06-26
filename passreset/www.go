package passreset

import (
	"carefront/api"
	"carefront/email"
	"carefront/libs/golog"
	"carefront/www"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

const (
	resetCodeDigits       = 6
	resetCodeMax          = 999999
	minimumPasswordLength = 6
)

type promptHandler struct {
	r            *mux.Router
	dataAPI      api.DataAPI
	authAPI      api.AuthAPI
	emailService email.Service
	supportEmail string
	webSubdomain string
}

type verifyHandler struct {
	r                *mux.Router
	dataAPI          api.DataAPI
	authAPI          api.AuthAPI
	twilioCli        *twilio.Client
	fromNumber       string
	supportEmail     string
	statInvalidToken metrics.Counter
	statExpiredToken metrics.Counter
}

type resetHandler struct {
	r                *mux.Router
	dataAPI          api.DataAPI
	authAPI          api.AuthAPI
	emailService     email.Service
	supportEmail     string
	statInvalidToken metrics.Counter
	statExpiredToken metrics.Counter
}

func RouteResetPassword(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, twilioCli *twilio.Client, fromNumber string, emailService email.Service, supportEmail, webSubdomain string, metricsRegistry metrics.Registry) {
	ph := &promptHandler{
		r:            r,
		dataAPI:      dataAPI,
		authAPI:      authAPI,
		emailService: emailService,
		supportEmail: supportEmail,
		webSubdomain: webSubdomain,
	}

	vh := &verifyHandler{
		r:                r,
		dataAPI:          dataAPI,
		authAPI:          authAPI,
		twilioCli:        twilioCli,
		fromNumber:       fromNumber,
		supportEmail:     supportEmail,
		statInvalidToken: metrics.NewCounter(),
		statExpiredToken: metrics.NewCounter(),
	}
	metricsRegistry.Add("verify/fail/invalid_token", vh.statInvalidToken)
	metricsRegistry.Add("verify/fail/expired_token", vh.statExpiredToken)

	rh := &resetHandler{
		r:                r,
		dataAPI:          dataAPI,
		authAPI:          authAPI,
		emailService:     emailService,
		supportEmail:     supportEmail,
		statInvalidToken: metrics.NewCounter(),
		statExpiredToken: metrics.NewCounter(),
	}
	metricsRegistry.Add("reset/fail/invalid_token", vh.statInvalidToken)
	metricsRegistry.Add("reset/fail/expired_token", vh.statExpiredToken)

	r.Handle("/reset-password", ph).Name("reset-password-prompt")
	r.Handle("/reset-password/verify", vh).Name("reset-password-verify")
	r.Handle("/reset-password/password", rh).Name("reset-password")
}

func (h *promptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: rate-limit this endpoint

	email := r.FormValue("email")
	invalidEmail := false
	if email != "" {
		account, err := h.authAPI.GetAccountForEmail(email)
		if err == api.LoginDoesNotExist {
			invalidEmail = true
		} else if err != nil {
			www.InternalServerError(w, r, err)
			return
		} else if r.Method == "POST" {
			domain := r.Host
			if idx := strings.IndexByte(domain, '.'); idx >= 0 {
				domain = domain[idx+1:]
			}
			domain = fmt.Sprintf("%s.%s", h.webSubdomain, domain)
			if err := SendPasswordResetEmail(h.authAPI, h.emailService, domain, account.ID, email, h.supportEmail); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			www.TemplateResponse(w, http.StatusOK, PromptTemplate, &PromptTemplateContext{
				Email:        email,
				Sent:         true,
				SupportEmail: h.supportEmail,
			})
			return
		}
	} else if r.Method == "POST" {
		invalidEmail = true
	}

	www.TemplateResponse(w, http.StatusOK, PromptTemplate, &PromptTemplateContext{
		Email:        email,
		InvalidEmail: invalidEmail,
		SupportEmail: h.supportEmail,
	})
}

func (h *verifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID, roleType, token, emailAddress, rsent := validateToken(w, r, h.r, h.authAPI, api.LostPassword, h.statInvalidToken, h.statExpiredToken)
	if rsent {
		return
	}

	var toNumber string
	switch roleType {
	case api.DOCTOR_ROLE:
		doctor, err := h.dataAPI.GetDoctorFromAccountId(accountID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		toNumber = doctor.CellPhone
	case api.PATIENT_ROLE:
		patient, err := h.dataAPI.GetPatientFromAccountId(accountID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		for _, phoneNumber := range patient.PhoneNumbers {
			if phoneNumber.PhoneType == api.PHONE_CELL {
				toNumber = patient.PhoneNumbers[0].Phone
				break
			}
		}
	}
	var lastDigits string
	if len(toNumber) >= 2 {
		lastDigits = toNumber[len(toNumber)-2:]
	}

	if r.Method == "POST" {
		action := r.FormValue("action")
		switch action {
		case "send":
			contact := r.FormValue("method")
			if contact == "sms" {
				bigCode, err := rand.Int(rand.Reader, big.NewInt(resetCodeMax))
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				code := bigCode.String()
				for len(code) < resetCodeDigits {
					code = "0" + code
				}
				if _, err := h.authAPI.CreateTempToken(accountID, lostPasswordCodeExpires, api.LostPasswordCode, fmt.Sprintf("%d:%s", accountID, code)); err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				if _, _, err := h.twilioCli.Messages.SendSMS(h.fromNumber, toNumber, fmt.Sprintf("Your Spruce verification code is %s", code)); err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				www.TemplateResponse(w, http.StatusOK, VerifyTemplate, &VerifyTemplateContext{
					Token:         token,
					Email:         emailAddress,
					LastTwoDigits: lastDigits,
					EnterCode:     true,
					SupportEmail:  h.supportEmail,
				})
				return
			}
		case "validate":
			code := r.FormValue("code")
			codeToken := fmt.Sprintf("%d:%s", accountID, code)
			_, _, err := h.authAPI.ValidateTempToken(api.LostPasswordCode, codeToken)
			if err != nil {
				switch err {
				case api.TokenExpired:
					h.statExpiredToken.Inc(1)
				case api.TokenDoesNotExist:
					h.statInvalidToken.Inc(1)
				default:
					www.InternalServerError(w, r, err)
					return
				}
				www.TemplateResponse(w, http.StatusOK, VerifyTemplate, &VerifyTemplateContext{
					Token:         token,
					Email:         emailAddress,
					LastTwoDigits: lastDigits,
					EnterCode:     true,
					Code:          code,
					Errors:        []string{"Code is incorrect. Check to make sure it's typed correctly."},
					SupportEmail:  h.supportEmail,
				})
				return
			}

			if err := h.authAPI.DeleteTempToken(api.LostPassword, token); err != nil {
				golog.Errorf("Failed to delete lost password token: %s", err.Error())
			}
			if err := h.authAPI.DeleteTempToken(api.LostPasswordCode, codeToken); err != nil {
				golog.Errorf("Failed to delete lost password code token: %s", err.Error())
			}

			resetToken, err := h.authAPI.CreateTempToken(accountID, resetPasswordExpires, api.PasswordReset, "")
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			params := url.Values{
				"token": []string{resetToken},
			}
			if emailAddress != "" {
				params.Set("email", emailAddress)
			}
			u, err := h.r.Get("reset-password").URLPath()
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			u.RawQuery = params.Encode()
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, VerifyTemplate, &VerifyTemplateContext{
		Token:         token,
		Email:         emailAddress,
		LastTwoDigits: lastDigits,
		SupportEmail:  h.supportEmail,
	})
}

func (h *resetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID, _, token, emailAddress, rsent := validateToken(w, r, h.r, h.authAPI, api.PasswordReset, h.statInvalidToken, h.statExpiredToken)
	if rsent {
		return
	}

	var errors []string
	var done bool
	if r.Method == "POST" {
		pass1 := r.FormValue("password1")
		pass2 := r.FormValue("password2")
		if len(pass1) < minimumPasswordLength {
			// TODO: further validation of length?
			errors = append(errors, fmt.Sprintf("Password must be longer than %d characters.", minimumPasswordLength-1))
		} else if pass1 != pass2 {
			errors = append(errors, "Passwords do not match.")
		} else {
			if err := h.authAPI.SetPassword(accountID, pass1); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			if err := h.authAPI.DeleteTempToken(api.PasswordReset, token); err != nil {
				golog.Errorf("Failed to delete password reset token: %s", err.Error())
			}
			done = true
			if err := SendPasswordHasBeenResetEmail(h.emailService, emailAddress, h.supportEmail); err != nil {
				golog.Errorf("Failed to send password reset success email: %s", err.Error())
			}
		}
	}
	www.TemplateResponse(w, http.StatusOK, ResetTemplate, &ResetTemplateContext{
		Token:        token,
		Email:        emailAddress,
		Done:         done,
		Errors:       errors,
		SupportEmail: h.supportEmail,
	})
}

func validateToken(w http.ResponseWriter, r *http.Request, router *mux.Router, authAPI api.AuthAPI, purpose string, statInvalidToken, statExpiredToken metrics.Counter) (int64, string, string, string, bool) {
	token := r.FormValue("token")
	emailAddress := r.FormValue("email")
	var accountID int64
	var roleType string
	if token == "" {
		statInvalidToken.Inc(1)
	} else {
		var err error
		accountID, roleType, err = authAPI.ValidateTempToken(purpose, token)
		if err != nil {
			switch err {
			case api.TokenExpired:
				statExpiredToken.Inc(1)
			case api.TokenDoesNotExist:
				statInvalidToken.Inc(1)
			default:
				www.InternalServerError(w, r, err)
				return 0, "", token, emailAddress, true
			}
		}
	}
	if accountID == 0 {
		// If the token is invalid then redirect to the reset-password page where
		// the person can request a new reset email.
		params := url.Values{}
		if emailAddress != "" {
			params.Set("email", emailAddress)
		}
		u, err := router.Get("reset-password-prompt").URLPath()
		if err != nil {
			www.InternalServerError(w, r, err)
			return 0, "", token, emailAddress, true
		}
		u.RawQuery = params.Encode()
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
		return 0, "", token, emailAddress, true
	}
	return accountID, roleType, token, emailAddress, false
}

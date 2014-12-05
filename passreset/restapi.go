package passreset

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/httputil"
)

type ForgotPasswordRequest struct {
	Email string `json:"email" schema:"email,required"`
}

type forgotPasswordHandler struct {
	dataAPI      api.DataAPI
	authAPI      api.AuthAPI
	emailService email.Service
	fromEmail    string
	webDomain    string
}

func NewForgotPasswordHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, emailService email.Service, fromEmail, webDomain string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&forgotPasswordHandler{
				dataAPI:      dataAPI,
				authAPI:      authAPI,
				emailService: emailService,
				fromEmail:    fromEmail,
				webDomain:    webDomain,
			}), []string{"POST"})
}

func (h *forgotPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Failed to decode request: "+err.Error())
		return
	}
	if req.Email == "" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "email is required")
		return
	}

	// TODO: ratelimit this endpoint

	account, err := h.authAPI.GetAccountForEmail(req.Email)
	if err == api.LoginDoesNotExist {
		apiservice.WriteUserError(w, http.StatusOK, "No account with the given email")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := SendPasswordResetEmail(h.authAPI, h.emailService, h.webDomain, account.ID, req.Email, h.fromEmail); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (*forgotPasswordHandler) NonAuthenticated() bool {
	return true
}

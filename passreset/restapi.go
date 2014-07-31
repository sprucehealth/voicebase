package passreset

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/email"
)

type ForgotPasswordRequest struct {
	Email string `json:"email" schema:"email,required"`
}

type forgotPasswordHandler struct {
	dataAPI      api.DataAPI
	authAPI      api.AuthAPI
	emailService email.Service
	fromEmail    string
	webSubdomain string
}

func NewForgotPasswordHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, emailService email.Service, fromEmail, webSubdomain string) http.Handler {
	return &forgotPasswordHandler{
		dataAPI:      dataAPI,
		authAPI:      authAPI,
		emailService: emailService,
		fromEmail:    fromEmail,
		webSubdomain: webSubdomain,
	}
}

func (h *forgotPasswordHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h *forgotPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

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

	domain := r.Host
	if idx := strings.IndexByte(domain, '.'); idx >= 0 {
		domain = domain[idx+1:]
	}
	domain = fmt.Sprintf("%s.%s", h.webSubdomain, domain)
	if err := SendPasswordResetEmail(h.authAPI, h.emailService, domain, account.ID, req.Email, h.fromEmail); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (*forgotPasswordHandler) NonAuthenticated() bool {
	return true
}

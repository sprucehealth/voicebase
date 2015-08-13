package passreset

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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
	webDomain    string
}

func NewForgotPasswordHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, emailService email.Service, webDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&forgotPasswordHandler{
				dataAPI:      dataAPI,
				authAPI:      authAPI,
				emailService: emailService,
				webDomain:    webDomain,
			}), httputil.Post)
}

func (h *forgotPasswordHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}
	if req.Email == "" {
		apiservice.WriteValidationError(ctx, "email is required", w, r)
		return
	}

	// TODO: ratelimit this endpoint

	account, err := h.authAPI.AccountForEmail(req.Email)
	if err == api.ErrLoginDoesNotExist {
		apiservice.WriteJSONSuccess(w)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if err := SendPasswordResetEmail(h.authAPI, h.emailService, h.webDomain, account.ID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}

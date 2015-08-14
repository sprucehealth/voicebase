package home

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type signInAPIHandler struct {
	authAPI api.AuthAPI
}

type signInAPIRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signInAPIResponse struct{}

func (r *signInAPIRequest) Validate() (bool, string) {
	if r.Email == "" {
		return false, "Email is required"
	}
	if r.Password == "" {
		return false, "Password is required"
	}
	return true, ""
}

func newSignInAPIHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&signInAPIHandler{
		authAPI: authAPI,
	}, httputil.Post)
}

func (h *signInAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req signInAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	if ok, reason := req.Validate(); !ok {
		www.APIGeneralError(w, r, "invalid_request", reason)
		return
	}
	acc, err := h.authAPI.Authenticate(req.Email, req.Password)
	if err != nil {
		var e www.APIErrorResponse
		switch err {
		case api.ErrLoginDoesNotExist:
			e.Error.Message = "Invalid email"
			e.Error.Type = "invalid_email"
		case api.ErrInvalidPassword:
			e.Error.Message = "Invalid password"
			e.Error.Type = "invalid_password"
		default:
			www.APIInternalError(w, r, err)
			return
		}
		httputil.JSONResponse(w, www.HTTPStatusAPIError, &e)
		return
	}

	// For now only allow patients to sign in through this endpoint as for doctors
	// and admins we need 2FA which can be implemented in the future.
	if acc.Role != api.RolePatient {
		httputil.JSONResponse(w, www.HTTPStatusAPIError, &www.APIErrorResponse{
			Error: www.APIError{
				Message: "Auth not allowed",
				Type:    "invalid_role",
			},
		})
		return
	}

	token, err := h.authAPI.CreateToken(acc.ID, api.Web, 0)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	http.SetCookie(w, www.NewAuthCookie(token, r))
	httputil.JSONResponse(w, http.StatusOK, signInAPIResponse{})
}

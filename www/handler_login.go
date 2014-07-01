package www

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
)

type loginHandler struct {
	authAPI api.AuthAPI
}

func NewLoginHandler(authAPI api.AuthAPI) http.Handler {
	return SupportedMethodsFilter(&loginHandler{
		authAPI: authAPI,
	}, []string{"GET", "POST"})
}

func (h *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: rate-limit this endpoint

	email := r.FormValue("email")
	next, valid := validateRedirectURL(r.FormValue("next"))
	if !valid {
		next = "/"
	}

	var errorMessage string

	if r.Method == "POST" {
		password := r.FormValue("password")
		_, token, err := h.authAPI.LogIn(email, password)
		if err != nil {
			switch err {
			case api.LoginDoesNotExist, api.InvalidPassword:
				errorMessage = "Email or password is not valid."
			default:
				InternalServerError(w, r, err)
				return
			}
		} else {
			http.SetCookie(w, NewAuthCookie(token, r))
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
	}

	TemplateResponse(w, http.StatusOK, LoginTemplate, &BaseTemplateContext{
		Title: "Login | Spruce",
		SubContext: &LoginTemplateContext{
			Error: errorMessage,
			Email: email,
			Next:  next,
		},
	})
}

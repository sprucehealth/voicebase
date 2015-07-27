package home

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/www"
)

type emailOptoutHandler struct {
	dataAPI  api.DataAPI
	authAPI  api.AuthAPI
	signer   *sig.Signer
	template *template.Template
}

func newEmailOptoutHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, signer *sig.Signer, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	t := templateLoader.MustLoadTemplate("home/email-optout.html", "home/base.html", nil)
	return httputil.ContextSupportedMethods(&emailOptoutHandler{
		dataAPI:  dataAPI,
		authAPI:  authAPI,
		signer:   signer,
		template: t,
	}, httputil.Get, httputil.Post)
}

func (h *emailOptoutHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tmplCtx := &struct {
		Unsubscribed bool
		Email        string
		Error        string
	}{
		Email: r.FormValue("email"),
	}
	switch r.Method {
	case httputil.Get:
		sig := []byte(r.FormValue("sig"))
		if h.signer.Verify([]byte("optout:"+r.FormValue("id")), sig) {
			accountID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
			if err := h.dataAPI.EmailUpdateOptOut(accountID, "all", true); err != nil {
				golog.Errorf(err.Error())
				tmplCtx.Error = "Internal error. Please try again later."
			} else {
				tmplCtx.Unsubscribed = true
			}
		}
	case httputil.Post:
		account, err := h.authAPI.AccountForEmail(strings.ToLower(strings.TrimSpace(tmplCtx.Email)))
		if err == api.ErrLoginDoesNotExist {
			tmplCtx.Error = "No account found for the entered email."
		} else if err != nil {
			golog.Errorf(err.Error())
			tmplCtx.Error = "Internal error. Please try again later."
		} else {
			if err := h.dataAPI.EmailUpdateOptOut(account.ID, "all", true); err != nil {
				golog.Errorf(err.Error())
				tmplCtx.Error = "Internal error. Please try again later."
			} else {
				tmplCtx.Unsubscribed = true
			}
		}
	}
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       "Email Optout | Spruce",
		SubContext: &homeContext{
			SubContext: tmplCtx,
		},
	})
}

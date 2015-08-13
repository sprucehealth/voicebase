package dronboard

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/www"
)

type introHandler struct {
	router   *mux.Router
	nextStep string
	signer   *sig.Signer
	template *template.Template
}

func newIntroHandler(router *mux.Router, signer *sig.Signer, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&introHandler{
		router:   router,
		nextStep: "doctor-register-account",
		template: templateLoader.MustLoadTemplate("dronboard/intro.html", "dronboard/base.html", nil),
		signer:   signer,
	}, httputil.Get)
}

func (h *introHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !validateRequestSignature(h.signer, r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	nextURL, err := h.router.Get(h.nextStep).URLPath()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	nextURL.RawQuery = r.Form.Encode()

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Welcome | Doctor Registration | Spruce",
		SubContext: &struct {
			NextURL string
		}{
			NextURL: nextURL.String(),
		},
	})
}

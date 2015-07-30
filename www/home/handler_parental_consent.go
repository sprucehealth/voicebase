package home

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type parentalConsentHandler struct {
	dataAPI  api.DataAPI
	template *template.Template
}

func newParentalConsentHandler(dataAPI api.DataAPI, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&parentalConsentHandler{
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("home/parental-consent.html", "", nil),
	}, httputil.Get)
}

func (h *parentalConsentHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// account := context.Get(r, www.CKAccount).(*common.Account)
	www.TemplateResponse(w, http.StatusOK, h.template, &struct {
		Account     *common.Account
		Environment string
	}{
		// Account:     account,
		Environment: environment.GetCurrent(),
	})
}

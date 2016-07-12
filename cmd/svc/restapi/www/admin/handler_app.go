package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
)

type appHandler struct {
	template *template.Template
}

func newAppHandler(templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&appHandler{
		template: templateLoader.MustLoadTemplate("admin/app.html", "admin/base.html", nil),
	}, httputil.Get)
}

func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
	perms := www.MustCtxPermissions(r.Context())

	audit.LogAction(account.ID, "Admin", "LoadAdminApp", nil)

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Admin",
		SubContext: &struct {
			Account     *common.Account
			Permissions map[string]bool
			Environment string
		}{
			Account:     account,
			Permissions: perms,
			Environment: environment.GetCurrent(),
		},
	})
}

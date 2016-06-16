package admin

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type appHandler struct {
	template *template.Template
}

func newAppHandler(templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&appHandler{
		template: templateLoader.MustLoadTemplate("admin/app.html", "admin/base.html", nil),
	}, httputil.Get)
}

func (h *appHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	perms := www.MustCtxPermissions(ctx)

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

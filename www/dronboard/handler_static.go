package dronboard

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type staticTemplateHandler struct {
	template *template.Template
	context  interface{}
}

func newStaticTemplateHandler(template *template.Template, context interface{}) httputil.ContextHandler {
	return httputil.SupportedMethods(&staticTemplateHandler{
		template: template,
		context:  context,
	}, httputil.Get)
}

func (h *staticTemplateHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, h.context)
}

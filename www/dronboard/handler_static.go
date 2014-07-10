package dronboard

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/www"
)

type staticTemplateHandler struct {
	template *template.Template
	context  interface{}
}

func NewStaticTemplateHandler(template *template.Template, context interface{}) http.Handler {
	return www.SupportedMethodsHandler(&staticTemplateHandler{
		template: template,
		context:  context,
	}, []string{"GET"})
}

func (h *staticTemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	www.TemplateResponse(w, http.StatusOK, h.template, h.context)
}

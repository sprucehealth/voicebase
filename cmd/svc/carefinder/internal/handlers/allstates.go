package handlers

import (
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
)

type allStatesPageHandler struct {
	refTemplate      *template.Template
	allStatesService service.PageContentBuilder
}

func NewAllStatesPageHandler(templateLoader *www.TemplateLoader, allStatesService service.PageContentBuilder) http.Handler {
	return &allStatesPageHandler{
		refTemplate:      templateLoader.MustLoadTemplate("allstatespage.html", "base.html", nil),
		allStatesService: allStatesService,
	}
}

func (a *allStatesPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sp, err := a.allStatesService.PageContentForID(nil, r)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, a.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML(sp.(*response.AllStatesPage).HTMLTitle),
		SubContext:  sp,
	})
}

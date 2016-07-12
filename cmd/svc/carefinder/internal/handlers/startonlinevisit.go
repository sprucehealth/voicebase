package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type startOnlineVisitHandler struct {
	refTemplate             *template.Template
	startOnlineVisitService service.PageContentBuilder
}

func NewStartOnlineVisitHandler(templateLoader *www.TemplateLoader, startOnlineVisitService service.PageContentBuilder) httputil.ContextHandler {
	return &startOnlineVisitHandler{
		refTemplate:             templateLoader.MustLoadTemplate("startonlinevisit.html", "base.html", nil),
		startOnlineVisitService: startOnlineVisitService,
	}
}

func (d *startOnlineVisitHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	doctorID := "md-" + mux.Vars(ctx)["doctor"]
	sp, err := d.startOnlineVisitService.PageContentForID(&service.StartOnlineVisitPageContext{
		DoctorID: doctorID,
	}, r)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	if sp == nil {
		www.BadRequestError(w, r, fmt.Errorf("No doctor found for %s", doctorID))
		return
	}

	www.TemplateResponse(w, http.StatusOK, d.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML(sp.(*response.StartOnlineVisitPage).HTMLTitle),
		SubContext:  sp,
	})
}

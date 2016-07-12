package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type doctorPageHandler struct {
	refTemplate   *template.Template
	doctorService service.PageContentBuilder
}

func NewDoctorPageHandler(templateLoader *www.TemplateLoader, doctorService service.PageContentBuilder) httputil.ContextHandler {
	return &doctorPageHandler{
		refTemplate:   templateLoader.MustLoadTemplate("doctorpage.html", "base.html", nil),
		doctorService: doctorService,
	}
}

func (d *doctorPageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(ctx)
	doctorID := fmt.Sprintf("md-%s", vars["doctor"])
	cityID := r.FormValue("city_id")

	dp, err := d.doctorService.PageContentForID(&service.DoctorPageContext{
		DoctorID: doctorID,
		CityID:   cityID,
	}, r)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	if dp == nil {
		www.BadRequestError(w, r, fmt.Errorf("Doctor with id %s not found", doctorID))
		return
	}

	www.TemplateResponse(w, http.StatusOK, d.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML(dp.(*response.DoctorPage).HTMLTitle),
		SubContext:  dp,
	})
}

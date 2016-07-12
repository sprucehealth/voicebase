package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/mux"
)

type cityPageHandler struct {
	refTemplate *template.Template
	cityService service.PageContentBuilder
}

func NewCityPageHandler(templateLoader *www.TemplateLoader, cityService service.PageContentBuilder) http.Handler {
	return &cityPageHandler{
		refTemplate: templateLoader.MustLoadTemplate("citypage.html", "base.html", nil),
		cityService: cityService,
	}
}

func (c *cityPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cityID := mux.Vars(r.Context())["city"]
	stateKey := mux.Vars(r.Context())["state"]
	cp, err := c.cityService.PageContentForID(&service.CityPageContext{
		CityID:   cityID,
		StateKey: stateKey,
	}, r)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	if cp == nil {
		www.BadRequestError(w, r, fmt.Errorf("No city page found for %s", cityID))
		return
	}

	www.TemplateResponse(w, http.StatusOK, c.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML(cp.(*response.CityPage).HTMLTitle),
		SubContext:  cp,
	})
}

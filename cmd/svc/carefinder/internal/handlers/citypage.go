package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type cityPageHandler struct {
	refTemplate *template.Template
	cityService service.PageContentBuilder
}

func NewCityPageHandler(templateLoader *www.TemplateLoader, cityService service.PageContentBuilder) httputil.ContextHandler {
	return &cityPageHandler{
		refTemplate: templateLoader.MustLoadTemplate("citypage.html", "base.html", nil),
		cityService: cityService,
	}
}

func (c *cityPageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	cityID := mux.Vars(ctx)["city"]
	cp, err := c.cityService.PageContentForID(&service.CityPageContext{
		CityID: cityID,
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

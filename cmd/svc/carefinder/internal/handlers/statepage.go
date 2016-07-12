package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/mux"
)

type statePageHandler struct {
	refTemplate  *template.Template
	stateService service.PageContentBuilder
	cityDAL      dal.CityDAL
	webURL       string
}

func NewStatePageHandler(templateLoader *www.TemplateLoader, stateService service.PageContentBuilder, cityDAL dal.CityDAL, webURL string) http.Handler {
	return &statePageHandler{
		refTemplate:  templateLoader.MustLoadTemplate("statepage.html", "base.html", nil),
		stateService: stateService,
		cityDAL:      cityDAL,
		webURL:       webURL,
	}
}

func (s *statePageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stateKey := mux.Vars(r.Context())["state"]

	// check if we are dealing with a city page and redirect to city page URL.
	// doing this because the URL structures changed from carefinder/city to carefinder/state/city
	// but google had already indexed at carefinder/city
	// TODO: Remove after some time when we're sure that google has re-indexed at new location.
	city, err := s.cityDAL.ShortListedCity(stateKey)
	if errors.Cause(err) != dal.ErrNoCityFound && err != nil {
		www.InternalServerError(w, r, err)
		return
	} else if city != nil {
		cityURL := response.CityPageURL(city, s.webURL)
		http.Redirect(w, r, cityURL, http.StatusMovedPermanently)
		return
	}

	sp, err := s.stateService.PageContentForID(&service.StatePageContext{
		StateKey: stateKey,
	}, r)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	if sp == nil {
		www.BadRequestError(w, r, fmt.Errorf("No state page found for %s", stateKey))
		return
	}

	www.TemplateResponse(w, http.StatusOK, s.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML(sp.(*response.StatePage).HTMLTitle),
		SubContext:  sp,
	})
}

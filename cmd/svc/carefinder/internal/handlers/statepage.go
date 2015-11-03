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

type statePageHandler struct {
	refTemplate  *template.Template
	stateService service.PageContentBuilder
}

func NewStatePageHandler(templateLoader *www.TemplateLoader, stateService service.PageContentBuilder) httputil.ContextHandler {
	return &statePageHandler{
		refTemplate:  templateLoader.MustLoadTemplate("statepage.html", "base.html", nil),
		stateService: stateService,
	}
}

func (s *statePageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	stateKey := mux.Vars(ctx)["state"]
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

package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type resourceGuideHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

type resourceGuideForm struct {
	SectionID int64
	Ordinal   int
	Title     string
	PhotoURL  string
	Layout    string
}

func NewResourceGuideHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&resourceGuideHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *resourceGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	guideID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var form resourceGuideForm
	var errorMsg string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		if err := schema.NewDecoder().Decode(&form, r.PostForm); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		var layout interface{}
		if err := json.Unmarshal([]byte(form.Layout), &layout); err != nil {
			errorMsg = fmt.Sprintf("Failed to parse JSON: %s", err.Error())
		} else {
			guide := &common.ResourceGuide{
				ID:        guideID,
				SectionID: form.SectionID,
				Ordinal:   form.Ordinal,
				Title:     form.Title,
				PhotoURL:  form.PhotoURL,
				Layout:    layout,
			}
			if err := h.dataAPI.UpdateResourceGuide(guide); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			http.Redirect(w, r, "/admin/resourceguide", http.StatusSeeOther)
			return
		}
	} else {
		guide, err := h.dataAPI.GetResourceGuide(guideID)
		if err == api.NoRowsError {
			http.NotFound(w, r)
			return
		} else if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		form = resourceGuideForm{
			SectionID: guide.SectionID,
			Ordinal:   guide.Ordinal,
			Title:     guide.Title,
			PhotoURL:  guide.PhotoURL,
		}
		b, err := json.MarshalIndent(guide.Layout, "", "  ")
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		form.Layout = string(b)
	}

	www.TemplateResponse(w, http.StatusOK, resourceGuideTemplate, &www.BaseTemplateContext{
		Title: template.HTML(template.HTMLEscapeString(form.Title)),
		SubContext: &resourceGuideTemplateContext{
			Form:  &form,
			Error: errorMsg,
		},
	})
}

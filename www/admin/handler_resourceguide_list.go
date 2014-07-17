package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type resourceGuideListHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewResourceGuideListHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&resourceGuideListHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *resourceGuideListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sections, guides, err := h.dataAPI.ListResourceGuides()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	// doctor, err := h.dataAPI.GetDoctorFromId(doctorID)
	// if err == api.NoRowsError {
	// 	http.NotFound(w, r)
	// } else if err != nil {
	// 	www.InternalServerError(w, r, err)
	// 	return
	// }

	// licenses, err := h.dataAPI.MedicalLicenses(doctorID)
	// if err != nil {
	// 	www.InternalServerError(w, r, err)
	// 	return
	// }

	// attr, err := h.dataAPI.DoctorAttributes(doctorID, nil)
	// if err != nil {
	// 	www.InternalServerError(w, r, err)
	// 	return
	// }

	// attributes := make(map[string]template.HTML, len(attr))
	// for name, value := range attr {
	// 	switch name {
	// 	case api.AttrCVFile, api.AttrDriversLicenseFile, api.AttrClaimsHistoryFile:
	// 		attributes[name] = template.HTML(fmt.Sprintf(`<a href="/admin/doctor/%d/dl/%s">Download</a>`, doctorID, name))
	// 	case api.AttrPreviousLiabilityInsurers:
	// 		parts := strings.Split(value, "\n")
	// 		for i, x := range parts {
	// 			parts[i] = template.HTMLEscapeString(x)
	// 		}
	// 		attributes[name] = template.HTML(strings.Join(parts, "<br>"))
	// 	default:
	// 		attributes[name] = template.HTML(template.HTMLEscapeString(value))
	// 	}
	// }

	www.TemplateResponse(w, http.StatusOK, resourceGuideListTemplate, &www.BaseTemplateContext{
		Title: "Resource Guides",
		SubContext: &resourceGuideListTemplateContext{
			Sections: sections,
			Guides:   guides,
		},
	})
}

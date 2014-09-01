package handlers

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
)

type staticJSONHandler struct {
	staticBaseURL string
	imageTag      string
}

func NewFeaturedDoctorsHandler(staticBaseURL string) http.Handler {
	return &staticJSONHandler{
		staticBaseURL: staticBaseURL,
		imageTag:      "featured_doctors.json",
	}
}

func NewPatientFAQHandler(staticBaseURL string) http.Handler {
	return &staticJSONHandler{
		staticBaseURL: staticBaseURL,
		imageTag:      "faq.json",
	}
}

func (f *staticJSONHandler) NonAuthenticated() bool {
	return true
}

func (f *staticJSONHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (f *staticJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("%s%s", f.staticBaseURL, f.imageTag), http.StatusSeeOther)
}

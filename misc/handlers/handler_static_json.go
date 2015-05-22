package handlers

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type staticJSONHandler struct {
	staticBaseURL string
	imageTag      string
}

func NewFeaturedDoctorsHandler(staticBaseURL string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&staticJSONHandler{
				staticBaseURL: staticBaseURL,
				imageTag:      "featured_doctors.json",
			}), httputil.Get)
}

func NewPatientFAQHandler(staticBaseURL string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&staticJSONHandler{
				staticBaseURL: staticBaseURL,
				imageTag:      "faq.json",
			}), httputil.Get)
}

func NewPricingFAQHandler(staticBaseURL string) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&staticJSONHandler{
				staticBaseURL: staticBaseURL,
				imageTag:      "pricing_faq.json",
			}), httputil.Get)
}

func (f *staticJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("%s%s", f.staticBaseURL, f.imageTag), http.StatusSeeOther)
}

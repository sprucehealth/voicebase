package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, metricsRegistry metrics.Registry) {
	r.Handle("/doctor-register", NewRegisterHandler(r, dataAPI, authAPI)).Name("doctor-register")

	redir := http.RedirectHandler("/doctor-register", http.StatusSeeOther)
	authFilter := www.AuthRequiredFilter(authAPI, []string{api.DOCTOR_ROLE}, redir)

	r.Handle("/doctor-register/credentials", authFilter(NewCredentialsHandler(r, dataAPI))).Name("doctor-register-credentials")
	r.Handle("/doctor-register/upload-cv", authFilter(NewUploadCVHandler(r, dataAPI))).Name("doctor-register-upload-cv")
	r.Handle("/doctor-register/upload-license", authFilter(NewUploadLicenseHandler(r, dataAPI))).Name("doctor-register-upload-license")
	r.Handle("/doctor-register/engagement", authFilter(NewEngagementHandler(r, dataAPI))).Name("doctor-register-engagement")
	r.Handle("/doctor-register/financials", authFilter(NewFinancialsHandler(r, dataAPI))).Name("doctor-register-financials")
}

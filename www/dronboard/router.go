package dronboard

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, metricsRegistry metrics.Registry) {
	r.Handle("/doctor-register", NewSignupHandler(r, dataAPI, authAPI)).Name("doctor-register")
	r.Handle("/doctor-register/credentials", NewCredentialsHandler(r, dataAPI)).Name("doctor-register-credentials")
}

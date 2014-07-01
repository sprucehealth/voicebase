package dronboard

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, metricsRegistry metrics.Registry) {
	signup := NewSignupHandler(dataAPI, authAPI)

	r.Handle("/doctor-register", signup).Name("doctor-register")
}

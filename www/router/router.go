package router

import (
	"carefront/api"
	"carefront/email"
	"carefront/passreset"
	"carefront/www"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

func New(dataAPI api.DataAPI, authAPI api.AuthAPI, twilioCli *twilio.Client, fromNumber string, emailService email.Service, fromEmail, webSubdomain string, metricsRegistry metrics.Registry) http.Handler {
	router := mux.NewRouter()
	// Better a blank page for root than a 404
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		www.TemplateResponse(w, http.StatusOK, www.IndexTemplate, &www.IndexTemplateContext{})
	})
	passreset.RouteResetPassword(router, dataAPI, authAPI, twilioCli, fromNumber, emailService, fromEmail, webSubdomain, metricsRegistry.Scope("reset_password"))
	return router
}

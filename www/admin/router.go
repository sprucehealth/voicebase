package admin

import (
	"log"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, stripeCli *stripe.StripeService, stores map[string]storage.Store, metricsRegistry metrics.Registry) {
	if stores["onboarding"] == nil {
		log.Fatal("onboarding storage not configured")
	}

	adminRoles := []string{api.ADMIN_ROLE}
	authFilter := www.AuthRequiredFilter(authAPI, adminRoles, nil)

	r.Handle("/admin", authFilter(http.RedirectHandler("/admin/doctor", http.StatusSeeOther))).Name("admin")
	r.Handle("/admin/doctor", authFilter(NewDoctorSearchHandler(r, dataAPI))).Name("admin-doctor-search")
	r.Handle("/admin/doctor/{id:[0-9]+}", authFilter(NewDoctorHandler(r, dataAPI))).Name("admin-doctor")
}

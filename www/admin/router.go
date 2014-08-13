package admin

import (
	"log"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, stripeCli *stripe.StripeService, signer *common.Signer, stores map[string]storage.Store, templateLoader *www.TemplateLoader, metricsRegistry metrics.Registry) {
	if stores["onboarding"] == nil {
		log.Fatal("onboarding storage not configured")
	}

	templateLoader.MustLoadTemplate("admin/base.html", "base.html", nil)

	adminRoles := []string{api.ADMIN_ROLE}
	authFilter := www.AuthRequiredFilter(authAPI, adminRoles, nil)
	r.Handle(`/admin/doctors/{id:[0-9]+}/dl/{attr:[A-Za-z0-9_\-]+}`, authFilter(NewDoctorAttrDownloadHandler(r, dataAPI, stores["onboarding"]))).Name("admin-doctor-attr-download")

	apiAuthFailHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		www.JSONResponse(w, r, http.StatusForbidden, &www.APIError{Message: "Access not allowed"})
	})
	apiAuthFilter := www.AuthRequiredFilter(authAPI, adminRoles, apiAuthFailHandler)

	r.Handle(`/admin/api/doctors`, apiAuthFilter(NewDoctorSearchAPIHandler(dataAPI)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}`, apiAuthFilter(NewDoctorAPIHandler(dataAPI)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/attributes`, apiAuthFilter(NewDoctorAttributesAPIHandler(dataAPI)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/licenses`, apiAuthFilter(NewMedicalLicenseAPIHandler(dataAPI)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/profile`, apiAuthFilter(NewDoctorProfileAPIHandler(dataAPI)))
	r.Handle(`/admin/api/dronboarding`, apiAuthFilter(NewDoctorOnboardingURLAPIHandler(r, dataAPI, signer)))
	r.Handle(`/admin/api/guides/resources`, apiAuthFilter(NewResourceGuidesListAPIHandler(dataAPI)))
	r.Handle(`/admin/api/guides/resources/{id:[0-9]+}`, apiAuthFilter(NewResourceGuidesAPIHandler(dataAPI)))
	r.Handle(`/admin/api/guides/rx`, apiAuthFilter(NewRXGuideListAPIHandler(dataAPI)))
	r.Handle(`/admin/api/guides/rx/{ndc:[0-9]+}`, apiAuthFilter(NewRXGuideAPIHandler(dataAPI)))

	appHandler := authFilter(NewAppHandler(templateLoader))
	r.Handle(`/admin`, appHandler)
	r.Handle(`/admin/{page:.*}`, appHandler)
}

package admin

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

const (
	PermAdminAccountsEdit   = "admin_accounts.edit"
	PermAdminAccountsView   = "admin_accounts.view"
	PermAnalyticsReportEdit = "analytics_reports.edit"
	PermAnalyticsReportView = "analytics_reports.view"
)

const (
	maxMemory = 1 << 20
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, analyticsDB *sql.DB, stripeCli *stripe.StripeService, signer *common.Signer, stores storage.StoreMap, templateLoader *www.TemplateLoader, metricsRegistry metrics.Registry, onboardingURLExpires int64) {
	if stores["onboarding"] == nil {
		log.Fatal("onboarding storage not configured")
	}

	templateLoader.MustLoadTemplate("admin/base.html", "base.html", nil)

	noPermsRequired := www.NoPermissionsRequiredFilter(authAPI)

	adminRoles := []string{api.ADMIN_ROLE}
	authFilter := www.AuthRequiredFilter(authAPI, adminRoles, nil)
	r.Handle(`/admin/doctors/{id:[0-9]+}/dl/{attr:[A-Za-z0-9_\-]+}`, authFilter(NewDoctorAttrDownloadHandler(r, dataAPI, stores["onboarding"]))).Name("admin-doctor-attr-download")
	r.Handle(`/admin/analytics/reports/{id:[0-9]+}/presentation/iframe`, authFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET": []string{PermAnalyticsReportView},
			},
			NewAnalyticsPresentationIframeHandler(dataAPI, templateLoader), nil)))

	apiAuthFailHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		www.JSONResponse(w, r, http.StatusForbidden, &www.APIError{Message: "Access not allowed"})
	})
	apiAuthFilter := www.AuthRequiredFilter(authAPI, adminRoles, apiAuthFailHandler)

	r.Handle(`/admin/api/doctors`, apiAuthFilter(noPermsRequired(NewDoctorSearchAPIHandler(dataAPI))))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}`, apiAuthFilter(noPermsRequired(NewDoctorAPIHandler(dataAPI))))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/attributes`, apiAuthFilter(noPermsRequired(NewDoctorAttributesAPIHandler(dataAPI))))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/licenses`, apiAuthFilter(noPermsRequired(NewMedicalLicenseAPIHandler(dataAPI))))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/profile`, apiAuthFilter(noPermsRequired(NewDoctorProfileAPIHandler(dataAPI))))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/savedmessage`, apiAuthFilter(noPermsRequired(NewDoctorSavedMessageHandler(dataAPI))))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/thumbnail/{size:[a-z]+}`, apiAuthFilter(noPermsRequired(NewDoctorThumbnailAPIHandler(dataAPI, stores.MustGet("thumbnails")))))
	r.Handle(`/admin/api/dronboarding`, apiAuthFilter(noPermsRequired(NewDoctorOnboardingURLAPIHandler(r, dataAPI, signer, onboardingURLExpires))))
	r.Handle(`/admin/api/guides/resources`, apiAuthFilter(noPermsRequired(NewResourceGuidesListAPIHandler(dataAPI))))
	r.Handle(`/admin/api/guides/resources/{id:[0-9]+}`, apiAuthFilter(noPermsRequired(NewResourceGuidesAPIHandler(dataAPI))))
	r.Handle(`/admin/api/guides/rx`, apiAuthFilter(noPermsRequired(NewRXGuideListAPIHandler(dataAPI))))
	r.Handle(`/admin/api/guides/rx/{ndc:[0-9]+}`, apiAuthFilter(noPermsRequired(NewRXGuideAPIHandler(dataAPI))))
	r.Handle(`/admin/api/accounts/permissions`, apiAuthFilter(noPermsRequired(NewAccountAvailablePermissionsAPIHandler(authAPI))))
	r.Handle(`/admin/api/accounts/groups`, apiAuthFilter(noPermsRequired(NewAccountAvailableGroupsAPIHandler(authAPI))))
	r.Handle(`/admin/api/admins`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET": []string{PermAdminAccountsView},
			},
			NewAdminsListAPIHandler(authAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET":  []string{PermAdminAccountsView},
				"POST": []string{PermAdminAccountsEdit},
			},
			NewAdminsAPIHandler(authAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/groups`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET":  []string{PermAdminAccountsView},
				"POST": []string{PermAdminAccountsEdit},
			},
			NewAdminsGroupsAPIHandler(authAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/permissions`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET":  []string{PermAdminAccountsView},
				"POST": []string{PermAdminAccountsEdit},
			},
			NewAdminsPermissionsAPIHandler(authAPI), nil)))
	r.Handle(`/admin/api/analytics/query`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"POST": []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsQueryAPIHandler(analyticsDB), nil)))
	r.Handle(`/admin/api/analytics/reports`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET":  []string{PermAnalyticsReportView},
				"POST": []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsReportsListAPIHandler(dataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"GET":  []string{PermAnalyticsReportView},
				"POST": []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsReportsAPIHandler(dataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}/run`, apiAuthFilter(
		www.PermissionsRequiredHandler(authAPI,
			map[string][]string{
				"POST": []string{PermAnalyticsReportView},
			},
			NewAnalyticsReportsRunAPIHandler(dataAPI, analyticsDB), nil)))

	appHandler := authFilter(noPermsRequired(NewAppHandler(templateLoader)))
	r.Handle(`/admin`, appHandler)
	r.Handle(`/admin/{page:.*}`, appHandler)
}

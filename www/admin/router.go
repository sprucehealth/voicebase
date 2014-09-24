package admin

import (
	"database/sql"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

const (
	PermAdminAccountsEdit       = "admin_accounts.edit"
	PermAdminAccountsView       = "admin_accounts.view"
	PermAnalyticsReportEdit     = "analytics_reports.edit"
	PermAnalyticsReportView     = "analytics_reports.view"
	PermDoctorsEdit             = "doctors.edit"
	PermDoctorsView             = "doctors.view"
	PermEmailEdit               = "email.edit"
	PermEmailView               = "email.view"
	PermAppMessageTemplatesEdit = "sched_msgs.edit"
	PermAppMessageTemplatesView = "sched_msgs.view"
)

const (
	maxMemory = 1 << 20
)

type Config struct {
	DataAPI              api.DataAPI
	AuthAPI              api.AuthAPI
	ERxAPI               erx.ERxAPI
	AnalyticsDB          *sql.DB
	Signer               *common.Signer
	Stores               storage.StoreMap
	TemplateLoader       *www.TemplateLoader
	EmailService         email.Service
	OnboardingURLExpires int64
	MetricsRegistry      metrics.Registry
}

func SetupRoutes(r *mux.Router, config *Config) {
	config.TemplateLoader.MustLoadTemplate("admin/base.html", "base.html", nil)

	noPermsRequired := www.NoPermissionsRequiredFilter(config.AuthAPI)

	adminRoles := []string{api.ADMIN_ROLE}
	authFilter := www.AuthRequiredFilter(config.AuthAPI, adminRoles, nil)
	r.Handle(`/admin/doctors/{id:[0-9]+}/dl/{attr:[A-Za-z0-9_\-]+}`, authFilter(
		NewDoctorAttrDownloadHandler(r, config.DataAPI, config.Stores.MustGet("onboarding")))).Name("admin-doctor-attr-download")
	r.Handle(`/admin/analytics/reports/{id:[0-9]+}/presentation/iframe`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermAnalyticsReportView},
			},
			NewAnalyticsPresentationIframeHandler(config.DataAPI, config.TemplateLoader), nil)))

	apiAuthFailHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		www.APIForbidden(w, r)
	})
	apiAuthFilter := www.AuthRequiredFilter(config.AuthAPI, adminRoles, apiAuthFailHandler)

	r.Handle(`/admin/api/drugs`, apiAuthFilter(noPermsRequired(NewDrugSearchAPIHandler(config.DataAPI, config.ERxAPI))))
	r.Handle(`/admin/api/doctors`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
			},
			NewDoctorSearchAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
			},
			NewDoctorAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/attributes`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
			},
			NewDoctorAttributesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/licenses`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
			},
			NewMedicalLicenseAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/profile`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
				"PUT": []string{PermDoctorsEdit},
			},
			NewDoctorProfileAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/savedmessage`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
				"PUT": []string{PermDoctorsEdit},
			},
			NewDoctorSavedMessageHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/doctors/{id:[0-9]+}/thumbnail/{size:[a-z]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView},
				"PUT": []string{PermDoctorsEdit},
			},
			NewDoctorThumbnailAPIHandler(config.DataAPI, config.Stores.MustGet("thumbnails")), nil)))
	r.Handle(`/admin/api/dronboarding`, apiAuthFilter(noPermsRequired(NewDoctorOnboardingURLAPIHandler(r, config.DataAPI, config.Signer, config.OnboardingURLExpires))))
	r.Handle(`/admin/api/guides/resources`, apiAuthFilter(noPermsRequired(NewResourceGuidesListAPIHandler(config.DataAPI))))
	r.Handle(`/admin/api/guides/resources/{id:[0-9]+}`, apiAuthFilter(noPermsRequired(NewResourceGuidesAPIHandler(config.DataAPI))))
	r.Handle(`/admin/api/guides/rx`, apiAuthFilter(noPermsRequired(NewRXGuideListAPIHandler(config.DataAPI))))
	r.Handle(`/admin/api/guides/rx/{ndc:[0-9]+}`, apiAuthFilter(noPermsRequired(NewRXGuideAPIHandler(config.DataAPI))))
	r.Handle(`/admin/api/accounts/permissions`, apiAuthFilter(noPermsRequired(NewAccountAvailablePermissionsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/groups`, apiAuthFilter(noPermsRequired(NewAccountAvailableGroupsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":   []string{PermDoctorsView, PermAdminAccountsView},
				"PATCH": []string{PermDoctorsEdit, PermAdminAccountsEdit},
			},
			NewAccountHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}/phones`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermDoctorsView, PermAdminAccountsView},
			},
			NewAccountPhonesListHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/email/types`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermEmailView},
			},
			NewEmailTypesListHandler(), nil)))
	r.Handle(`/admin/api/email/senders`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermEmailView},
				"POST": []string{PermEmailEdit},
			},
			NewEmailSendersListHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/email/templates`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermEmailView},
				"POST": []string{PermEmailEdit},
			},
			NewEmailTemplatesListHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/email/templates/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermEmailView},
				"PUT": []string{PermEmailEdit},
			},
			NewEmailTemplatesHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/email/templates/{id:[0-9]+}/test`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"POST": []string{PermEmailEdit},
			},
			NewEmailTemplatesTestHandler(config.EmailService, config.DataAPI), nil)))
	r.Handle(`/admin/api/admins`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermAdminAccountsView},
			},
			NewAdminsListAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermAdminAccountsView},
				"POST": []string{PermAdminAccountsEdit},
			},
			NewAdminsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/groups`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermAdminAccountsView},
				"POST": []string{PermAdminAccountsEdit},
			},
			NewAdminsGroupsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/permissions`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermAdminAccountsView},
				"POST": []string{PermAdminAccountsEdit},
			},
			NewAdminsPermissionsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/analytics/query`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"POST": []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsQueryAPIHandler(config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/analytics/reports`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermAnalyticsReportView},
				"POST": []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsReportsListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermAnalyticsReportView},
				"POST": []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsReportsAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}/run`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"POST": []string{PermAnalyticsReportView},
			},
			NewAnalyticsReportsRunAPIHandler(config.DataAPI, config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/schedmsgs/templates`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":  []string{PermAppMessageTemplatesView},
				"POST": []string{PermAppMessageTemplatesEdit},
			},
			NewSchedMessageTemplatesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/schedmsgs/events`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET": []string{PermAppMessageTemplatesView},
			},
			NewSchedMessageEventsListAPIHandler(), nil)))
	r.Handle(`/admin/api/schedmsgs/templates/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				"GET":    []string{PermAppMessageTemplatesView},
				"PUT":    []string{PermAppMessageTemplatesEdit},
				"DELETE": []string{PermAppMessageTemplatesEdit},
			},
			NewSchedMessageTemplatesAPIHandler(config.DataAPI), nil)))
	appHandler := authFilter(noPermsRequired(NewAppHandler(config.TemplateLoader)))
	r.Handle(`/admin`, appHandler)
	r.Handle(`/admin/{page:.*}`, appHandler)
}

package admin

import (
	"database/sql"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/events"
	"github.com/sprucehealth/backend/financial"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/www"
)

const (
	PermAdminAccountsEdit       = "admin_accounts.edit"
	PermAdminAccountsView       = "admin_accounts.view"
	PermAnalyticsReportEdit     = "analytics_reports.edit"
	PermAnalyticsReportView     = "analytics_reports.view"
	PermAppMessageTemplatesEdit = "sched_msgs.edit"
	PermAppMessageTemplatesView = "sched_msgs.view"
	PermCareCoordinatorView     = "care_coordinator.view"
	PermCareCoordinatorEdit     = "care_coordinator.edit"
	PermCaseView                = "case.view"
	PermCaseEdit                = "case.edit"
	PermCFGEdit                 = "cfg.edit"
	PermCFGView                 = "cfg.view"
	PermDoctorsEdit             = "doctors.edit"
	PermDoctorsView             = "doctors.view"
	PermEmailEdit               = "email.edit"
	PermEmailView               = "email.view"
	PermFinancialView           = "financial.view"
	PermFTPEdit                 = "ftp.edit"
	PermFTPView                 = "ftp.view"
	PermLayoutEdit              = "layout.edit"
	PermLayoutView              = "layout.view"
	PermMarketingEdit           = "marketing.edit"
	PermMarketingView           = "marketing.view"
	PermPathwaysEdit            = "pathways.edit"
	PermPathwaysView            = "pathways.view"
	PermResourceGuidesEdit      = "resource_guides.edit"
	PermResourceGuidesView      = "resource_guides.view"
	PermRXGuidesEdit            = "rx_guides.edit"
	PermRXGuidesView            = "rx_guides.view"
	PermSTPEdit                 = "stp.edit"
	PermSTPView                 = "stp.view"
	PermPromotionView           = "promo.view"
	PermPromotionEdit           = "promo.edit"
)

const (
	maxMemory = 1 << 20
)

type Config struct {
	DataAPI         api.DataAPI
	AuthAPI         api.AuthAPI
	ApplicationDB   *sql.DB
	DiagnosisAPI    diagnosis.API
	ERxAPI          erx.ERxAPI
	AnalyticsDB     *sql.DB
	Signer          *sig.Signer
	Stores          storage.StoreMap
	TemplateLoader  *www.TemplateLoader
	EmailService    email.Service
	LibratoClient   *librato.Client
	StripeClient    *stripe.StripeService
	MediaStore      *media.Store
	APIDomain       string
	WebDomain       string
	MetricsRegistry metrics.Registry
	EventsClient    events.Client
	Cfg             cfg.Store
}

func SetupRoutes(r *mux.Router, config *Config) {
	config.TemplateLoader.MustLoadTemplate("admin/base.html", "base.html", nil)

	noPermsRequired := www.NoPermissionsRequiredFilter(config.AuthAPI)
	taggingClient := tagging.NewTaggingClient(config.ApplicationDB)

	adminRoles := []string{api.RoleAdmin}
	authFilter := www.AuthRequiredFilter(config.AuthAPI, adminRoles, nil)
	r.Handle(`/admin/providers/{id:[0-9]+}/dl/{attr:[A-Za-z0-9_\-]+}`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI, map[string][]string{
			httputil.Get: []string{PermDoctorsView},
		}, NewProviderAttrDownloadHandler(r, config.DataAPI, config.Stores.MustGet("onboarding")), nil))).Name("admin-doctor-attr-download")
	r.Handle(`/admin/analytics/reports/{id:[0-9]+}/presentation/iframe`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermAnalyticsReportView},
			},
			NewAnalyticsPresentationIframeHandler(config.DataAPI, config.TemplateLoader), nil)))

	apiAuthFailHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		www.APIForbidden(w, r)
	})
	apiAuthFilter := www.AuthRequiredFilter(config.AuthAPI, adminRoles, apiAuthFailHandler)

	r.Handle(`/admin/api/drugs`, apiAuthFilter(noPermsRequired(NewDrugSearchAPIHandler(config.DataAPI, config.ERxAPI))))
	r.Handle(`/admin/api/diagnosis/code`, apiAuthFilter(noPermsRequired(NewDiagnosisSearchHandler(config.DataAPI, config.DiagnosisAPI))))

	r.Handle(`/admin/api/cfg`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermCFGView},
				httputil.Patch: []string{PermCFGEdit},
			},
			NewCFGHandler(config.Cfg), nil)))
	r.Handle(`/admin/api/providers/state_pathway_mappings`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			NewProviderMappingsHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/state_pathway_mappings/summary`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			NewProviderMappingsSummaryHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermDoctorsView},
				httputil.Post: []string{PermDoctorsEdit},
			},
			NewProviderSearchAPIHandler(config.DataAPI, config.AuthAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			NewProviderAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/attributes`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			NewProviderAttributesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/licenses`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
				httputil.Put: []string{PermDoctorsEdit},
			},
			NewMedicalLicenseAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/profile`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
				httputil.Put: []string{PermDoctorsEdit},
			},
			NewProviderProfileAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/profile_image/{type:[a-z]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
				httputil.Put: []string{PermDoctorsEdit},
			},
			NewProviderProfileImageAPIHandler(config.DataAPI, config.Stores.MustGet("thumbnails")), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/eligibility`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermDoctorsView},
				httputil.Patch: []string{PermDoctorsEdit},
			},
			NewProviderEligibilityListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/treatment_plan/favorite`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermFTPView},
			},
			NewProviderFTPHandler(config.DataAPI, config.MediaStore), nil)))
	r.Handle(`/admin/api/dronboarding`, apiAuthFilter(
		noPermsRequired(NewProviderOnboardingURLAPIHandler(r, config.DataAPI, config.Signer, config.Cfg))))
	r.Handle(`/admin/api/guides/resources`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermResourceGuidesView},
				httputil.Put:  []string{PermResourceGuidesEdit},
				httputil.Post: []string{PermResourceGuidesEdit},
			},
			NewResourceGuidesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/resources/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermResourceGuidesView},
				httputil.Patch: []string{PermResourceGuidesEdit},
			},
			NewResourceGuidesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/rx`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermRXGuidesView},
				httputil.Put: []string{PermRXGuidesEdit},
			},
			NewRXGuideListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/rx/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermRXGuidesView},
			},
			NewRXGuideAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/accounts/permissions`, apiAuthFilter(noPermsRequired(NewAccountAvailablePermissionsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/groups`, apiAuthFilter(noPermsRequired(NewAccountAvailableGroupsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermDoctorsView, PermAdminAccountsView},
				httputil.Patch: []string{PermDoctorsEdit, PermAdminAccountsEdit},
			},
			NewAccountHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}/phones`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView, PermAdminAccountsView},
			},
			NewAccountPhonesListHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermAdminAccountsView},
			},
			NewAdminsListAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAdminAccountsView},
				httputil.Post: []string{PermAdminAccountsEdit},
			},
			NewAdminsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/groups`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAdminAccountsView},
				httputil.Post: []string{PermAdminAccountsEdit},
			},
			NewAdminsGroupsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/permissions`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAdminAccountsView},
				httputil.Post: []string{PermAdminAccountsEdit},
			},
			NewAdminsPermissionsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/analytics/query`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsQueryAPIHandler(config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/analytics/reports`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAnalyticsReportView},
				httputil.Post: []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsReportsListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAnalyticsReportView},
				httputil.Post: []string{PermAnalyticsReportEdit},
			},
			NewAnalyticsReportsAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}/run`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: []string{PermAnalyticsReportView},
			},
			NewAnalyticsReportsRunAPIHandler(config.DataAPI, config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/schedmsgs/templates`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAppMessageTemplatesView},
				httputil.Post: []string{PermAppMessageTemplatesEdit},
			},
			NewSchedMessageTemplatesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/schedmsgs/events`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermAppMessageTemplatesView},
			},
			NewSchedMessageEventsListAPIHandler(), nil)))
	r.Handle(`/admin/api/schedmsgs/templates/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:    []string{PermAppMessageTemplatesView},
				httputil.Put:    []string{PermAppMessageTemplatesEdit},
				httputil.Delete: []string{PermAppMessageTemplatesEdit},
			},
			NewSchedMessageTemplatesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermPathwaysView},
				httputil.Post: []string{PermPathwaysEdit},
			},
			NewPathwaysListHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/menu`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermPathwaysView},
				httputil.Put: []string{PermPathwaysEdit},
			},
			NewPathwayMenuHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermPathwaysView},
				httputil.Patch: []string{PermPathwaysEdit},
			},
			NewPathwayHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/diagnosis_sets`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermPathwaysView},
				httputil.Patch: []string{PermPathwaysEdit},
			},
			NewDiagnosisSetsHandler(config.DataAPI, config.DiagnosisAPI), nil)))
	r.Handle(`/admin/api/email/test`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: []string{PermEmailEdit},
			},
			NewEmailTestSendHandler(config.EmailService, config.Signer, config.WebDomain), nil)))

	// Layout CMS APIS
	r.Handle(`/admin/api/layouts/versioned_question`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermLayoutView},
			httputil.Post: []string{PermLayoutEdit},
		}, NewVersionedQuestionHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layouts/version`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermLayoutView},
		}, NewLayoutVersionHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layouts/template`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermLayoutView},
		}, NewLayoutTemplateHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layout`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermLayoutView},
			httputil.Post: []string{PermLayoutEdit},
		}, NewLayoutUploadHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layout/diagnosis`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermLayoutView},
			httputil.Post: []string{PermLayoutEdit},
		}, NewDiagnosisDetailsIntakeUploadHandler(config.DataAPI, config.DiagnosisAPI), nil)))
	r.Handle(`/admin/api/layout/saml`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Post: []string{PermLayoutEdit},
		}, NewSAMLAPIHandler(), nil)))

	// STP Interaction
	r.Handle(`/admin/api/sample_treatment_plan`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermSTPView},
			httputil.Put: []string{PermSTPEdit},
		}, NewSampleTreatmentPlanHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/treatment_plan/csv`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: []string{PermSTPEdit},
		}, NewTreatmentPlanCSVHandler(config.DataAPI, config.ERxAPI), nil)))

	// FTP Interaction
	r.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:    []string{PermFTPView},
			httputil.Post:   []string{PermFTPEdit},
			httputil.Delete: []string{PermFTPEdit},
		}, NewFTPMembershipHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFTPView},
		}, NewFTPHandler(config.DataAPI, config.MediaStore), nil)))
	r.Handle(`/admin/api/treatment_plan/favorite/global`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFTPView},
		}, NewGlobalFTPHandler(config.DataAPI, config.MediaStore), nil)))

	// Financial APIs
	financialAccess := financial.NewDataAccess(config.ApplicationDB)

	r.Handle("/admin/api/financial/incoming", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFinancialView},
		}, NewIncomingFinancialItemsHandler(financialAccess), nil)))
	r.Handle("/admin/api/financial/outgoing", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFinancialView},
		}, NewOutgoingFinancialItemsHandler(financialAccess), nil)))

	r.Handle("/admin/api/financial/skus/visit", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFinancialView, PermLayoutView},
		}, NewVisitSKUListHandler(config.DataAPI), nil)))

	// Case/Visit Interations
	r.Handle("/admin/api/case/{caseID:[0-9]+}/visit/{visitID:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermCaseView},
		}, NewCaseVisitHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/case/visit", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermCaseView},
		}, NewCaseVisitsHandler(config.DataAPI), nil)))

	// Event interaction
	r.Handle("/admin/api/event/server", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermCaseView},
		}, NewServerEventsHandler(config.EventsClient), nil)))

	// Tagging interaction
	r.Handle("/admin/api/tag", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Delete: []string{PermCareCoordinatorEdit},
			httputil.Get:    []string{PermCareCoordinatorView},
			httputil.Post:   []string{PermCareCoordinatorEdit},
			httputil.Put:    []string{PermCareCoordinatorEdit},
		}, NewTagHandler(taggingClient), nil)))
	r.Handle("/admin/api/tag/saved_search/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Delete: []string{PermCareCoordinatorEdit},
		}, NewTagSavedSearchHandler(taggingClient), nil)))
	r.Handle("/admin/api/tag/saved_search", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermCareCoordinatorView},
			httputil.Post: []string{PermCareCoordinatorEdit},
		}, NewTagSavedSearchesHandler(taggingClient), nil)))

	// Promotion Interaction
	r.Handle("/admin/api/promotion", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermMarketingView},
			httputil.Post: []string{PermMarketingEdit},
		}, NewPromotionHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_route", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermMarketingView},
			httputil.Post: []string{PermMarketingEdit},
		}, NewPromotionReferralRoutesHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_route/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: []string{PermMarketingEdit},
		}, NewPromotionReferralRouteHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_template", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermMarketingView},
			httputil.Post: []string{PermMarketingEdit},
		}, NewReferralProgramTemplateHandler(config.DataAPI), nil)))

	// Used for dashboard
	r.Handle(`/admin/api/librato/composite`, apiAuthFilter(noPermsRequired(NewLibratoCompositeAPIHandler(config.LibratoClient))))
	r.Handle(`/admin/api/stripe/charges`, apiAuthFilter(noPermsRequired(NewStripeChargesAPIHAndler(config.StripeClient))))

	if !environment.IsProd() {
		r.Handle(`/admin/medrecord`, apiAuthFilter(noPermsRequired(
			NewMedicalRecordHandler(
				config.DataAPI,
				config.DiagnosisAPI,
				config.MediaStore,
				config.APIDomain,
				config.WebDomain,
				config.Signer))))
	}

	r.Handle(`/admin/_dashboard/{id:[0-9]+}`, authFilter(noPermsRequired(newDashboardHandler(config.DataAPI, config.TemplateLoader))))
	appHandler := authFilter(noPermsRequired(NewAppHandler(config.TemplateLoader)))
	r.Handle(`/admin`, appHandler)
	r.Handle(`/admin/{page:.*}`, appHandler)
}

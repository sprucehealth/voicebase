package admin

import (
	"database/sql"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/events"
	"github.com/sprucehealth/backend/financial"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient/identification"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/www"
)

// Available permissions. If adding a new one then there must be a matching
// migration to actually add it to the database.
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
	PermAccountView             = "account.view"
	PermAccountEdit             = "account.edit"
)

const (
	maxMemory = 1 << 20
)

// Config is all the options and dependent services for the admin handlers.
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
	StripeClient    *stripe.Client
	MediaStore      *media.Store
	APIDomain       string
	WebDomain       string
	MetricsRegistry metrics.Registry
	EventsClient    events.Client
	Cfg             cfg.Store
	AnalyticsLogger analytics.Logger
}

// SetupRoutes configures all the admin handler routes using the provided router and config.
func SetupRoutes(r *mux.Router, config *Config) {
	config.TemplateLoader.MustLoadTemplate("admin/base.html", "base.html", nil)

	noPermsRequired := www.NoPermissionsRequiredFilter(config.AuthAPI)
	taggingClient := tagging.NewTaggingClient(config.ApplicationDB)

	// Initialize business logic services
	identificationService := identification.NewPatientIdentificationService(config.DataAPI, config.AuthAPI, config.AnalyticsLogger)

	authFilter := func(h httputil.ContextHandler) httputil.ContextHandler {
		return www.AuthRequiredHandler(www.RoleRequiredHandler(h, nil, api.RoleAdmin), nil, config.AuthAPI)
	}
	r.Handle(`/admin/providers/{id:[0-9]+}/dl/{attr:[A-Za-z0-9_\-]+}`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI, map[string][]string{
			httputil.Get: []string{PermDoctorsView},
		}, newProviderAttrDownloadHandler(r, config.DataAPI, config.Stores.MustGet("onboarding")), nil))).Name("admin-doctor-attr-download")
	r.Handle(`/admin/analytics/reports/{id:[0-9]+}/presentation/iframe`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermAnalyticsReportView},
			},
			newAnalyticsPresentationIframeHandler(config.DataAPI, config.TemplateLoader), nil)))

	apiAuthFilter := func(h httputil.ContextHandler) httputil.ContextHandler {
		return www.APIAuthRequiredHandler(www.APIRoleRequiredHandler(h, api.RoleAdmin), config.AuthAPI)
	}

	r.Handle(`/admin/api/drugs`, apiAuthFilter(noPermsRequired(newDrugSearchAPIHandler(config.DataAPI, config.ERxAPI))))
	r.Handle(`/admin/api/diagnosis/code`, apiAuthFilter(noPermsRequired(newDiagnosisSearchHandler(config.DataAPI, config.DiagnosisAPI))))

	r.Handle(`/admin/api/cfg`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermCFGView},
				httputil.Patch: []string{PermCFGEdit},
			},
			newCFGHandler(config.Cfg), nil)))
	r.Handle(`/admin/api/providers/state_pathway_mappings`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			newProviderMappingsHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/state_pathway_mappings/summary`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			newProviderMappingsSummaryHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermDoctorsView},
				httputil.Post: []string{PermDoctorsEdit},
			},
			newProviderSearchAPIHandler(config.DataAPI, config.AuthAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			newProviderAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/attributes`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
			},
			newProviderAttributesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/licenses`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
				httputil.Put: []string{PermDoctorsEdit},
			},
			newMedicalLicenseAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/profile`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
				httputil.Put: []string{PermDoctorsEdit},
			},
			newProviderProfileAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/profile_image/{type:[a-z]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView},
				httputil.Put: []string{PermDoctorsEdit},
			},
			newProviderProfileImageAPIHandler(config.DataAPI, config.Stores.MustGet("thumbnails")), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/eligibility`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermDoctorsView},
				httputil.Patch: []string{PermDoctorsEdit},
			},
			newProviderEligibilityListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/treatment_plan/favorite`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermFTPView},
			},
			newProviderFTPHandler(config.DataAPI, config.MediaStore), nil)))
	r.Handle(`/admin/api/dronboarding`, apiAuthFilter(
		noPermsRequired(newProviderOnboardingURLAPIHandler(r, config.DataAPI, config.Signer, config.Cfg))))
	r.Handle(`/admin/api/guides/resources`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermResourceGuidesView},
				httputil.Put:  []string{PermResourceGuidesEdit},
				httputil.Post: []string{PermResourceGuidesEdit},
			},
			newResourceGuidesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/resources/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermResourceGuidesView},
				httputil.Patch: []string{PermResourceGuidesEdit},
			},
			newResourceGuidesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/rx`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermRXGuidesView},
				httputil.Put: []string{PermRXGuidesEdit},
			},
			newRXGuideListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/rx/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermRXGuidesView},
			},
			newRXGuideAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/accounts/permissions`, apiAuthFilter(noPermsRequired(newAccountAvailablePermissionsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/groups`, apiAuthFilter(noPermsRequired(newAccountAvailableGroupsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermDoctorsView, PermAdminAccountsView},
				httputil.Patch: []string{PermDoctorsEdit, PermAdminAccountsEdit},
			},
			newAccountHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}/phones`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermDoctorsView, PermAdminAccountsView},
			},
			newAccountPhonesListHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermAdminAccountsView},
			},
			newAdminsListAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAdminAccountsView},
				httputil.Post: []string{PermAdminAccountsEdit},
			},
			newAdminsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/groups`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAdminAccountsView},
				httputil.Post: []string{PermAdminAccountsEdit},
			},
			newAdminsGroupsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/permissions`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAdminAccountsView},
				httputil.Post: []string{PermAdminAccountsEdit},
			},
			newAdminsPermissionsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/analytics/query`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: []string{PermAnalyticsReportEdit},
			},
			newAnalyticsQueryAPIHandler(config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/analytics/reports`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAnalyticsReportView},
				httputil.Post: []string{PermAnalyticsReportEdit},
			},
			newAnalyticsReportsListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAnalyticsReportView},
				httputil.Post: []string{PermAnalyticsReportEdit},
			},
			newAnalyticsReportsAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}/run`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: []string{PermAnalyticsReportView},
			},
			newAnalyticsReportsRunAPIHandler(config.DataAPI, config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/schedmsgs/templates`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermAppMessageTemplatesView},
				httputil.Post: []string{PermAppMessageTemplatesEdit},
			},
			newSchedMessageTemplatesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/schedmsgs/events`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermAppMessageTemplatesView},
			},
			newSchedMessageEventsListAPIHandler(), nil)))
	r.Handle(`/admin/api/schedmsgs/templates/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:    []string{PermAppMessageTemplatesView},
				httputil.Put:    []string{PermAppMessageTemplatesEdit},
				httputil.Delete: []string{PermAppMessageTemplatesEdit},
			},
			newSchedMessageTemplatesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  []string{PermPathwaysView},
				httputil.Post: []string{PermPathwaysEdit},
			},
			newPathwaysListHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/menu`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: []string{PermPathwaysView},
				httputil.Put: []string{PermPathwaysEdit},
			},
			newPathwayMenuHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermPathwaysView},
				httputil.Patch: []string{PermPathwaysEdit},
			},
			newPathwayHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/diagnosis_sets`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   []string{PermPathwaysView},
				httputil.Patch: []string{PermPathwaysEdit},
			},
			newDiagnosisSetsHandler(config.DataAPI, config.DiagnosisAPI), nil)))
	r.Handle(`/admin/api/email/test`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: []string{PermEmailEdit},
			},
			newEmailTestSendHandler(config.EmailService, config.Signer, config.WebDomain), nil)))

	// Layout CMS APIS
	r.Handle(`/admin/api/layouts/versioned_question`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermLayoutView},
			httputil.Post: []string{PermLayoutEdit},
		}, newVersionedQuestionHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layouts/version`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermLayoutView},
		}, newLayoutVersionHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layouts/template`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermLayoutView},
		}, newLayoutTemplateHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layout`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermLayoutView},
			httputil.Post: []string{PermLayoutEdit},
		}, newLayoutUploadHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layout/diagnosis`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermLayoutView},
			httputil.Post: []string{PermLayoutEdit},
		}, newDiagnosisDetailsIntakeUploadHandler(config.DataAPI, config.DiagnosisAPI), nil)))
	r.Handle(`/admin/api/layout/saml`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Post: []string{PermLayoutEdit},
		}, newSAMLAPIHandler(), nil)))

	// STP Interaction
	r.Handle(`/admin/api/sample_treatment_plan`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermSTPView},
			httputil.Put: []string{PermSTPEdit},
		}, newSampleTreatmentPlanHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/treatment_plan/csv`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: []string{PermSTPEdit},
		}, newTreatmentPlanCSVHandler(config.DataAPI, config.ERxAPI), nil)))

	// FTP Interaction
	r.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:    []string{PermFTPView},
			httputil.Post:   []string{PermFTPEdit},
			httputil.Delete: []string{PermFTPEdit},
		}, newFTPMembershipHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFTPView},
		}, newFTPHandler(config.DataAPI, config.MediaStore), nil)))
	r.Handle(`/admin/api/treatment_plan/favorite/global`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFTPView},
		}, newGlobalFTPHandler(config.DataAPI, config.MediaStore), nil)))

	// Financial APIs
	financialAccess := financial.NewDataAccess(config.ApplicationDB)

	r.Handle("/admin/api/financial/incoming", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFinancialView},
		}, newIncomingFinancialItemsHandler(financialAccess), nil)))
	r.Handle("/admin/api/financial/outgoing", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFinancialView},
		}, newOutgoingFinancialItemsHandler(financialAccess), nil)))

	r.Handle("/admin/api/financial/skus/visit", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermFinancialView, PermLayoutView},
		}, newVisitSKUListHandler(config.DataAPI), nil)))

	// Case/Visit Interations
	r.Handle("/admin/api/case/{caseID:[0-9]+}/visit/{visitID:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermCaseView},
		}, newCaseVisitHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/case/visit", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermCaseView},
		}, newCaseVisitsHandler(config.DataAPI), nil)))

	// Event interaction
	r.Handle("/admin/api/event/server", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermCaseView},
		}, newServerEventsHandler(config.EventsClient), nil)))

	// Tagging interaction
	r.Handle("/admin/api/tag", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Delete: []string{PermCareCoordinatorEdit},
			httputil.Get:    []string{PermCareCoordinatorView},
			httputil.Post:   []string{PermCareCoordinatorEdit},
			httputil.Put:    []string{PermCareCoordinatorEdit},
		}, newTagHandler(taggingClient), nil)))
	r.Handle("/admin/api/tag/saved_search/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Delete: []string{PermCareCoordinatorEdit},
		}, newTagSavedSearchHandler(taggingClient), nil)))
	r.Handle("/admin/api/tag/saved_search", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermCareCoordinatorView},
			httputil.Post: []string{PermCareCoordinatorEdit},
		}, newTagSavedSearchesHandler(taggingClient), nil)))

	// Promotion Interaction
	r.Handle("/admin/api/promotion", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermMarketingView},
			httputil.Post: []string{PermMarketingEdit},
		}, newPromotionsHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: []string{PermMarketingEdit},
		}, newPromotionHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/group", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: []string{PermMarketingView},
		}, newPromotionGroupsHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_route", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermMarketingView},
			httputil.Post: []string{PermMarketingEdit},
		}, newPromotionReferralRoutesHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_route/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: []string{PermMarketingEdit},
		}, newPromotionReferralRouteHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_template", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  []string{PermMarketingView},
			httputil.Post: []string{PermMarketingEdit},
			httputil.Put:  []string{PermMarketingEdit},
		}, newReferralProgramTemplateHandler(config.DataAPI), nil)))

	// Patient Interaction
	r.Handle("/admin/api/patient/{id:[0-9]+}/account/needs_id_verification", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Post: []string{PermAccountEdit},
		}, newPatientAccountNeedsVerifyIDHandler(identificationService), nil)))

	if !environment.IsProd() {
		r.Handle(`/admin/medrecord`, apiAuthFilter(noPermsRequired(
			newMedicalRecordHandler(
				config.DataAPI,
				config.DiagnosisAPI,
				config.MediaStore,
				config.APIDomain,
				config.WebDomain,
				config.Signer))))
	}

	appHandler := authFilter(noPermsRequired(newAppHandler(config.TemplateLoader)))
	r.Handle(`/admin`, appHandler)
	r.Handle(`/admin/{page:.*}`, appHandler)
}

package admin

import (
	"database/sql"
	"net/http"

	"github.com/samuel/go-librato/librato"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/analytics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/diagnosis"
	"github.com/sprucehealth/backend/cmd/svc/restapi/email"
	"github.com/sprucehealth/backend/cmd/svc/restapi/erx"
	"github.com/sprucehealth/backend/cmd/svc/restapi/events"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/cmd/svc/restapi/financial"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/mediastore"
	"github.com/sprucehealth/backend/cmd/svc/restapi/patient/identification"
	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
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
	MediaStore      *mediastore.Store
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

	authFilter := func(h http.Handler) http.Handler {
		return www.AuthRequiredHandler(www.RoleRequiredHandler(h, nil, api.RoleAdmin), nil, config.AuthAPI)
	}
	r.Handle(`/admin/providers/{id:[0-9]+}/dl/{attr:[A-Za-z0-9_\-]+}`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI, map[string][]string{
			httputil.Get: {PermDoctorsView},
		}, newProviderAttrDownloadHandler(r, config.DataAPI, config.Stores.MustGet("onboarding")), nil))).Name("admin-doctor-attr-download")
	r.Handle(`/admin/analytics/reports/{id:[0-9]+}/presentation/iframe`, authFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermAnalyticsReportView},
			},
			newAnalyticsPresentationIframeHandler(config.DataAPI, config.TemplateLoader), nil)))

	apiAuthFilter := func(h http.Handler) http.Handler {
		return www.APIAuthRequiredHandler(www.APIRoleRequiredHandler(h, api.RoleAdmin), config.AuthAPI)
	}

	r.Handle(`/admin/api/drugs`, apiAuthFilter(noPermsRequired(newDrugSearchAPIHandler(config.DataAPI, config.ERxAPI))))
	r.Handle(`/admin/api/diagnosis/code`, apiAuthFilter(noPermsRequired(newDiagnosisSearchHandler(config.DataAPI, config.DiagnosisAPI))))

	r.Handle(`/admin/api/cfg`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermCFGView},
				httputil.Patch: {PermCFGEdit},
			},
			newCFGHandler(config.Cfg), nil)))
	r.Handle(`/admin/api/providers/state_pathway_mappings`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
			},
			newProviderMappingsHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/state_pathway_mappings/summary`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
			},
			newProviderMappingsSummaryHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermDoctorsView},
				httputil.Post: {PermDoctorsEdit},
			},
			newProviderSearchAPIHandler(config.DataAPI, config.AuthAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermDoctorsView},
				httputil.Patch: {PermDoctorsEdit},
			},
			newProviderAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/attributes`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
			},
			newProviderAttributesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/licenses`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
				httputil.Put: {PermDoctorsEdit},
			},
			newMedicalLicenseAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/practice_model`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
				httputil.Put: {PermDoctorsEdit},
			},
			newPracticeModelHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/profile`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
				httputil.Put: {PermDoctorsEdit},
			},
			newProviderProfileAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/profile_image/{type:[a-z]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView},
				httputil.Put: {PermDoctorsEdit},
			},
			newProviderProfileImageAPIHandler(config.DataAPI, config.Stores.MustGet("thumbnails"), config.APIDomain), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/eligibility`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermDoctorsView},
				httputil.Patch: {PermDoctorsEdit},
			},
			newProviderEligibilityListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/treatment_plan/favorite`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermFTPView},
			},
			newProviderFTPHandler(config.DataAPI, config.MediaStore), nil)))
	r.Handle(`/admin/api/providers/{id:[0-9]+}/treatment_plan/sync_sftps`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: {PermDoctorsEdit},
			},
			newSyncGlobalFTPHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/dronboarding`, apiAuthFilter(
		noPermsRequired(newProviderOnboardingURLAPIHandler(r, config.DataAPI, config.Signer, config.Cfg))))
	r.Handle(`/admin/api/guides/resources`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermResourceGuidesView},
				httputil.Put:  {PermResourceGuidesEdit},
				httputil.Post: {PermResourceGuidesEdit},
			},
			newResourceGuidesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/resources/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermResourceGuidesView},
				httputil.Patch: {PermResourceGuidesEdit},
			},
			newResourceGuidesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/rx`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermRXGuidesView},
				httputil.Put: {PermRXGuidesEdit},
			},
			newRXGuideListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/guides/rx/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermRXGuidesView},
			},
			newRXGuideAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/accounts/permissions`, apiAuthFilter(noPermsRequired(newAccountAvailablePermissionsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/groups`, apiAuthFilter(noPermsRequired(newAccountAvailableGroupsAPIHandler(config.AuthAPI))))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermDoctorsView, PermAdminAccountsView},
				httputil.Patch: {PermDoctorsEdit, PermAdminAccountsEdit},
			},
			newAccountHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/accounts/{id:[0-9]+}/phones`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermDoctorsView, PermAdminAccountsView},
			},
			newAccountPhonesListHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermAdminAccountsView},
			},
			newAdminsListAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermAdminAccountsView},
				httputil.Post: {PermAdminAccountsEdit},
			},
			newAdminsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/groups`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermAdminAccountsView},
				httputil.Post: {PermAdminAccountsEdit},
			},
			newAdminsGroupsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/admins/{id:[0-9]+}/permissions`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermAdminAccountsView},
				httputil.Post: {PermAdminAccountsEdit},
			},
			newAdminsPermissionsAPIHandler(config.AuthAPI), nil)))
	r.Handle(`/admin/api/analytics/query`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: {PermAnalyticsReportEdit},
			},
			newAnalyticsQueryAPIHandler(config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/analytics/reports`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermAnalyticsReportView},
				httputil.Post: {PermAnalyticsReportEdit},
			},
			newAnalyticsReportsListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermAnalyticsReportView},
				httputil.Post: {PermAnalyticsReportEdit},
			},
			newAnalyticsReportsAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/analytics/reports/{id:[0-9]+}/run`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: {PermAnalyticsReportView},
			},
			newAnalyticsReportsRunAPIHandler(config.DataAPI, config.AnalyticsDB), nil)))
	r.Handle(`/admin/api/schedmsgs/templates`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermAppMessageTemplatesView},
				httputil.Post: {PermAppMessageTemplatesEdit},
			},
			newSchedMessageTemplatesListAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/schedmsgs/events`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermAppMessageTemplatesView},
			},
			newSchedMessageEventsListAPIHandler(), nil)))
	r.Handle(`/admin/api/schedmsgs/templates/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:    {PermAppMessageTemplatesView},
				httputil.Put:    {PermAppMessageTemplatesEdit},
				httputil.Delete: {PermAppMessageTemplatesEdit},
			},
			newSchedMessageTemplatesAPIHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:  {PermPathwaysView},
				httputil.Post: {PermPathwaysEdit},
			},
			newPathwaysListHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/menu`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get: {PermPathwaysView},
				httputil.Put: {PermPathwaysEdit},
			},
			newPathwayMenuHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/{id:[0-9]+}`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermPathwaysView},
				httputil.Patch: {PermPathwaysEdit},
			},
			newPathwayHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/pathways/diagnosis_sets`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Get:   {PermPathwaysView},
				httputil.Patch: {PermPathwaysEdit},
			},
			newDiagnosisSetsHandler(config.DataAPI, config.DiagnosisAPI), nil)))
	r.Handle(`/admin/api/email/test`, apiAuthFilter(
		www.PermissionsRequiredHandler(config.AuthAPI,
			map[string][]string{
				httputil.Post: {PermEmailEdit},
			},
			newEmailTestSendHandler(config.EmailService, config.Signer, config.WebDomain), nil)))

	// Layout CMS APIS
	r.Handle(`/admin/api/layouts/versioned_question`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermLayoutView},
			httputil.Post: {PermLayoutEdit},
		}, newVersionedQuestionHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layouts/version`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermLayoutView},
		}, newLayoutVersionHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layouts/template`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermLayoutView},
		}, newLayoutTemplateHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layout`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermLayoutView},
			httputil.Post: {PermLayoutEdit},
		}, newLayoutUploadHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/layout/diagnosis`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermLayoutView},
			httputil.Post: {PermLayoutEdit},
		}, newDiagnosisDetailsIntakeUploadHandler(config.DataAPI, config.DiagnosisAPI), nil)))
	r.Handle(`/admin/api/layout/saml`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Post: {PermLayoutEdit},
		}, newSAMLAPIHandler(), nil)))

	// STP Interaction
	r.Handle(`/admin/api/sample_treatment_plan`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermSTPView},
			httputil.Put: {PermSTPEdit},
		}, newSampleTreatmentPlanHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/treatment_plan/csv`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: {PermSTPEdit},
		}, newTreatmentPlanCSVHandler(config.DataAPI, config.ERxAPI), nil)))

	// FTP Interaction
	r.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:    {PermFTPView},
			httputil.Post:   {PermFTPEdit},
			httputil.Delete: {PermFTPEdit},
		}, newFTPMembershipHandler(config.DataAPI), nil)))
	r.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermFTPView},
		}, newFTPHandler(config.DataAPI, config.MediaStore), nil)))
	r.Handle(`/admin/api/treatment_plan/favorite/global`, apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermFTPView},
		}, newGlobalFTPHandler(config.DataAPI, config.MediaStore), nil)))

	// Financial APIs
	financialAccess := financial.NewDataAccess(config.ApplicationDB)

	r.Handle("/admin/api/financial/incoming", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermFinancialView},
		}, newIncomingFinancialItemsHandler(financialAccess), nil)))
	r.Handle("/admin/api/financial/outgoing", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermFinancialView},
		}, newOutgoingFinancialItemsHandler(financialAccess), nil)))

	r.Handle("/admin/api/financial/skus/visit", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermFinancialView, PermLayoutView},
		}, newVisitSKUListHandler(config.DataAPI), nil)))

	// Case/Visit Interations
	r.Handle("/admin/api/case/{caseID:[0-9]+}/visit/{visitID:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermCaseView},
		}, newCaseVisitHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/case/visit", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermCaseView},
		}, newCaseVisitsHandler(config.DataAPI), nil)))

	// Event interaction
	r.Handle("/admin/api/event/server", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermCaseView},
		}, newServerEventsHandler(config.EventsClient), nil)))

	// Tagging interaction
	r.Handle("/admin/api/tag", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Delete: {PermCareCoordinatorEdit},
			httputil.Get:    {PermCareCoordinatorView},
			httputil.Post:   {PermCareCoordinatorEdit},
			httputil.Put:    {PermCareCoordinatorEdit},
		}, newTagHandler(taggingClient), nil)))
	r.Handle("/admin/api/tag/saved_search/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Delete: {PermCareCoordinatorEdit},
		}, newTagSavedSearchHandler(taggingClient), nil)))
	r.Handle("/admin/api/tag/saved_search", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermCareCoordinatorView},
			httputil.Post: {PermCareCoordinatorEdit},
		}, newTagSavedSearchesHandler(taggingClient), nil)))

	// Promotion Interaction
	r.Handle("/admin/api/promotion", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermMarketingView},
			httputil.Post: {PermMarketingEdit},
		}, newPromotionsHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: {PermMarketingEdit},
		}, newPromotionHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/group", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermMarketingView},
		}, newPromotionGroupsHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_route", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermMarketingView},
			httputil.Post: {PermMarketingEdit},
		}, newPromotionReferralRoutesHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_route/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: {PermMarketingEdit},
		}, newPromotionReferralRouteHandler(config.DataAPI), nil)))
	r.Handle("/admin/api/promotion/referral_template", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get:  {PermMarketingView},
			httputil.Post: {PermMarketingEdit},
			httputil.Put:  {PermMarketingEdit},
		}, newReferralProgramTemplateHandler(config.DataAPI), nil)))

	// Feedback templates
	r.Handle("/admin/api/feedback/template_types", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermMarketingView},
		}, newFeedbackTemplateTypesHandler(), nil)))

	feedbackClient := feedback.NewDAL(config.ApplicationDB)
	r.Handle("/admin/api/feedback/template/{id:[0-9]+}", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermMarketingView},
		}, newFeedbackTemplateHandler(feedbackClient), nil)))
	r.Handle("/admin/api/feedback/template", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Put: {PermMarketingEdit},
		}, newFeedbackTemplateHandler(feedbackClient), nil)))
	r.Handle("/admin/api/feedback/template/list", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermMarketingView},
		}, newFeedbackTemplateListHandler(feedbackClient), nil)))
	r.Handle("/admin/api/feedback/rating_config", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Get: {PermMarketingView},
			httputil.Put: {PermMarketingEdit},
		}, newRatingLevelFeedbackConfigHandler(feedbackClient), nil)))

	// Patient Interaction
	r.Handle("/admin/api/patient/{id:[0-9]+}/account/needs_id_verification", apiAuthFilter(www.PermissionsRequiredHandler(config.AuthAPI,
		map[string][]string{
			httputil.Post: {PermAccountEdit},
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

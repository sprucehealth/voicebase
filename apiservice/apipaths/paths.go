package apipaths

const (
	AnalyticsURLPath                     = "/v1/event/client"
	AppEventURLPath                      = "/v1/app_event"
	AuthCheckEmailURLPath                = "/v1/auth/check_email"
	AutocompleteURLPath                  = "/v1/autocomplete"
	CareProviderProfileURLPath           = "/v1/care_provider_profile"
	CaseMessagesListURLPath              = "/v1/case/messages/list"
	CaseMessagesURLPath                  = "/v1/case/messages"
	NotifyMeURLPath                      = "/v1/notifyme"
	CheckEligibilityURLPath              = "/v1/check_eligibility"
	CareProviderSelectionURLPath         = "/v1/care_provider_selection"
	ContentURLPath                       = "/v1/content"
	DoctorAssignCaseURLPath              = "/v1/doctor/case/assign"
	DoctorAuthenticateTwoFactorURLPath   = "/v1/doctor/authenticate/two_factor"
	DoctorAuthenticateURLPath            = "/v1/doctor/authenticate"
	DoctorCaseCareTeamURLPath            = "/v1/doctor/case/care_team"
	DoctorCaseClaimURLPath               = "/v1/doctor/patient/case/claim"
	DoctorCaseHistoryURLPath             = "/v1/doctor/history/cases"
	DoctorFTPURLPath                     = "/v1/doctor/favorite_treatment_plans"
	DoctorIsAuthenticatedURLPath         = "/v1/doctor/isauthenticated"
	DoctorManageFTPURLPath               = "/v1/doctor/favorite_treatment_plans/manage"
	DoctorMedicationDispenseUnitsURLPath = "/v1/doctor/visit/treatment/medication_dispense_units"
	DoctorMedicationSearchURLPath        = "/v1/doctor/visit/treatment/medication_suggestions"
	DoctorMedicationStrengthsURLPath     = "/v1/doctor/visit/treatment/medication_strengths"
	DoctorPatientInfoURLPath             = "/v1/doctor/patient"
	DoctorPatientAppInfoURLPath          = "/v1/doctor/patient/app_info"
	DoctorPatientCasesListURLPath        = "/v1/doctor/patient/cases/list"
	DoctorPatientFollowupURLPath         = "/v1/doctor/patient/case/followup"
	DoctorPatientPharmacyURLPath         = "/v1/doctor/patient/pharmacy"
	DoctorPatientTreatmentsURLPath       = "/v1/doctor/patient/treatments"
	DoctorPatientVisitsURLPath           = "/v1/doctor/patient/visits"
	DoctorPharmacySearchURLPath          = "/v1/doctor/pharmacy"
	DoctorQueueURLPath                   = "/v1/doctor/queue"
	DoctorQueueInboxURLPath              = "/v1/doctor/queue/inbox"
	DoctorQueueUnassignedURLPath         = "/v1/doctor/queue/unassigned"
	DoctorQueueHistoryURLPath            = "/v1/doctor/queue/history"
	DoctorRefillRxDenialReasonsURLPath   = "/v1/doctor/rx/refill/denial_reasons"
	DoctorRefillRxURLPath                = "/v1/doctor/rx/refill/request"
	DoctorRegimenURLPath                 = "/v1/doctor/visit/regimen"
	DoctorRXErrorResolveURLPath          = "/v1/doctor/rx/error/resolve"
	DoctorRXErrorURLPath                 = "/v1/doctor/rx/error"
	DoctorSavedNoteURLPath               = "/v1/doctor/treatment_plans/note"
	DoctorSelectMedicationURLPath        = "/v1/doctor/visit/treatment/new"
	DoctorSignupURLPath                  = "/v1/doctor/signup"
	DoctorTPScheduledMessageURLPath      = "/v1/doctor/treatment_plans/schedmsg"
	DoctorTreatmentPlansListURLPath      = "/v1/doctor/treatment_plans/list"
	DoctorTreatmentPlansURLPath          = "/v1/doctor/treatment_plans"
	DoctorTreatmentTemplatesURLPath      = "/v1/doctor/treatment/templates"
	DoctorDiagnosisURLPath               = "/v1/doctor/diagnosis"
	DoctorDiagnosisSearchURLPath         = "/v1/doctor/diagnosis/search"
	DoctorVisitDiagnosisURLPath          = "/v1/doctor/visit/diagnosis"
	DoctorVisitDiagnosisListURLPath      = "/v1/doctor/visit/diagnosis/list"
	DoctorVisitReviewURLPath             = "/v1/doctor/visit/review"
	DoctorVisitTreatmentsURLPath         = "/v1/doctor/visit/treatment/treatments"
	LogoutURLPath                        = "/v1/logout"
	MediaURLPath                         = "/v1/media"
	NotificationPromptStatusURLPath      = "/v1/notification/prompt_status"
	NotificationTokenURLPath             = "/v1/notification/token"
	PatientAddressURLPath                = "/v1/patient/address/billing"
	PatientAlertsURLPath                 = "/v1/patient/alerts"
	PatientAuthenticateURLPath           = "/v1/authenticate"
	PatientCardURLPath                   = "/v1/credit_card"
	PatientCareTeamsURLPath              = "/v1/patient/care_teams"
	PatientCareTeamURLPath               = "/v1/patient/care_team"
	PatientCaseNotificationsURLPath      = "/v1/patient/case/notifications"
	PatientCasesListURLPath              = "/v1/cases/list"
	PatientCasesURLPath                  = "/v1/cases"
	PatientCostURLPath                   = "/v1/patient/cost"
	PatientCreditsURLPath                = "/v1/patient/credits"
	PatientDefaultCardURLPath            = "/v1/credit_card/default"
	PatientEmergencyContactsURLPath      = "/v1/patient/emergency_contacts"
	PatientFeaturedDoctorsURLPath        = "/v1/patient/featured_doctors"
	PatientHomeURLPath                   = "/v1/patient/home"
	PatientHowFAQURLPath                 = "/v1/patient/faq/general"
	PatientIsAuthenticatedURLPath        = "/v1/patient/isauthenticated"
	PatientMeURLPath                     = "/v1/patient/me"
	PatientPathwaysURLPath               = "/v1/patient/pathways"
	PatientPathwayDetailsURLPath         = "/v1/patient/pathway_details"
	PatientPCPURLPath                    = "/v1/patient/pcp"
	PatientPharmacyURLPath               = "/v1/patient/pharmacy"
	PatientPricingFAQURLPath             = "/v1/patient/faq/pricing"
	PatientRequestMedicalRecordURLPath   = "/v1/patient/request_medical_record"
	PatientSignupURLPath                 = "/v1/patient"
	PatientTreatmentsURLPath             = "/v1/patient/treatments"
	PatientUpdateURLPath                 = "/v1/patient/update"
	PatientVisitIntakeURLPath            = "/v1/patient/visit/answer"
	PatientVisitMessageURLPath           = "/v1/patient/visit/message"
	PatientVisitPhotoAnswerURLPath       = "/v1/patient/visit/photo_answer"
	PatientVisitsListURLPath             = "/v1/patient/visits/list"
	PatientVisitURLPath                  = "/v1/patient/visit"
	PharmacySearchURLPath                = "/v1/pharmacy_search"
	PhotoURLPath                         = "/v1/photo"
	PingURLPath                          = "/v1/ping"
	PromotionsURLPath                    = "/v1/promotions"
	ReferralProgramsTemplateURLPath      = "/v1/referrals/templates"
	ReferralsURLPath                     = "/v1/referrals"
	ResetPasswordURLPath                 = "/v1/reset_password"
	ResourceGuidesListURLPath            = "/v1/resourceguide/list"
	ResourceGuideURLPath                 = "/v1/resourceguide"
	SettingsURLPath                      = "/v1/settings"
	ProfileImageURLPath                  = "/v1/profile_image"
	TrainingCasesURLPath                 = "/v1/doctor/demo/patient_visit"
	TreatmentGuideURLPath                = "/v1/treatment_guide"
	TreatmentPlanURLPath                 = "/v1/treatment_plan"
	TPResourceGuideURLPath               = "/v1/treatment_plans/resourceguides"
)

// FIXME: these paths are in support of older apps. remove once not needed
const (
	DeprecatedDoctorSavedMessagesURLPath = "/v1/doctor/saved_messages" // pre Buzz Lightyear
)

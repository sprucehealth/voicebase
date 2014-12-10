package api

import (
	"errors"
	"net/http"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/sku"
)

const (
	EN_LANGUAGE_ID                 = 1
	ADMIN_ROLE                     = "ADMIN"
	DOCTOR_ROLE                    = "DOCTOR"
	PRIMARY_DOCTOR_STATUS          = "PRIMARY"
	PATIENT_ROLE                   = "PATIENT"
	MA_ROLE                        = "MA"
	FOLLOW_UP_WEEK                 = "week"
	FOLLOW_UP_DAY                  = "day"
	FOLLOW_UP_MONTH                = "month"
	REFILL_REQUEST_STATUS_PENDING  = "PENDING"
	REFILL_REQUEST_STATUS_SENDING  = "SENDING"
	REFILL_REQUEST_STATUS_APPROVED = "APPROVED"
	REFILL_REQUEST_STATUS_DENIED   = "DENIED"
	PATIENT_UNLINKED               = "UNLINKED"
	PATIENT_REGISTERED             = "REGISTERED"
	DOCTOR_REGISTERED              = "REGISTERED"
	HIPAA_AUTH                     = "hipaa"
	CONSENT_AUTH                   = "consent"
	PHONE_HOME                     = "Home"
	PHONE_WORK                     = "Work"
	PHONE_CELL                     = "Cell"
	HEALTH_CONDITION_ACNE_ID       = 1

	MinimumPasswordLength  = 6
	ReviewPurpose          = "REVIEW"
	ConditionIntakePurpose = "CONDITION_INTAKE"
	DiagnosePurpose        = "DIAGNOSE"
)

var (
	NoRowsError                 = errors.New("No rows exist")
	NoElligibileProviderInState = errors.New("There are no providers eligible in the state the patient resides")
	NoDiagnosisResponseErr      = errors.New("No diagnosis response exists to the question queried tag queried with")
)

type PatientAPI interface {
	Patient(id int64, basicInfoOnly bool) (*common.Patient, error)
	GetPatientFromID(patientID int64) (patient *common.Patient, err error)
	GetPatientFromAccountID(accountID int64) (patient *common.Patient, err error)
	GetPatientFromErxPatientID(erxPatientID int64) (*common.Patient, error)
	GetPatientFromRefillRequestID(refillRequestID int64) (*common.Patient, error)
	GetPatientFromTreatmentID(treatmentID int64) (*common.Patient, error)
	GetPatientFromCaseID(patientCaseID int64) (*common.Patient, error)
	GetPatientFromUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId int64) (*common.Patient, error)
	GetPatientVisitsForPatient(patientID int64) ([]*common.PatientVisit, error)
	PatientState(patientID int64) (string, error)
	AnyVisitSubmitted(patientID int64) (bool, error)
	RegisterPatient(patient *common.Patient) error
	UpdateTopLevelPatientInformation(patient *common.Patient) error
	UpdatePatientInformation(patient *common.Patient, updateFromDoctor bool) error
	CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, healthConditionID int64) error
	UpdatePatientWithERxPatientID(patientID, erxPatientID int64) error
	GetPatientIDFromAccountID(accountID int64) (int64, error)
	AddDoctorToCareTeamForPatient(patientID, healthConditionID, doctorID int64) error
	CreateCareTeamForPatientWithPrimaryDoctor(patientID, healthConditionID, doctorID int64) (careTeam *common.PatientCareTeam, err error)
	GetCareTeamForPatient(patientID int64) (careTeam *common.PatientCareTeam, err error)
	IsEligibleToServePatientsInState(state string, healthConditionID int64) (bool, error)
	UpdatePatientAddress(patientID int64, addressLine1, addressLine2, city, state, zipCode, addressType string) error
	UpdatePatientPharmacy(patientID int64, pharmacyDetails *pharmacy.PharmacyData) error
	TrackPatientAgreements(patientID int64, agreements map[string]bool) error
	PatientAgreements(patientID int64) (map[string]time.Time, error)
	GetPatientFromPatientVisitID(patientVisitID int64) (patient *common.Patient, err error)
	GetPatientFromTreatmentPlanID(treatmentPlanID int64) (patient *common.Patient, err error)
	GetPatientsForIDs(patientIDs []int64) ([]*common.Patient, error)
	GetPharmacySelectionForPatients(patientIDs []int64) ([]*pharmacy.PharmacyData, error)
	GetPharmacyBasedOnReferenceIdAndSource(pharmacyid int64, pharmacySource string) (*pharmacy.PharmacyData, error)
	GetPharmacyFromID(pharmacyLocalId int64) (*pharmacy.PharmacyData, error)
	AddPharmacy(pharmacyDetails *pharmacy.PharmacyData) error
	UpdatePatientWithPaymentCustomerId(patientID int64, paymentCustomerID string) error
	CreatePendingTask(workType, status string, itemID int64) (int64, error)
	DeletePendingTask(pendingTaskID int64) error
	AddCardForPatient(patientID int64, card *common.Card) error
	MarkCardInactiveForPatient(patientID int64, card *common.Card) error
	DeleteCardForPatient(patientID int64, card *common.Card) error
	MakeLatestCardDefaultForPatient(patientID int64) (*common.Card, error)
	MakeCardDefaultForPatient(patientID int64, card *common.Card) error
	GetCardsForPatient(patientID int64) ([]*common.Card, error)
	GetDefaultCardForPatient(patientID int64) (*common.Card, error)
	GetCardFromID(cardID int64) (*common.Card, error)
	GetCardFromThirdPartyID(thirdPartyID string) (*common.Card, error)
	UpdateDefaultAddressForPatient(patientID int64, address *common.Address) error
	DeleteAddress(addressID int64) error
	AddAlertsForPatient(patientID int64, source string, alerts []*common.Alert) error
	GetAlertsForPatient(patientID int64) ([]*common.Alert, error)
	UpdatePatientPCP(pcp *common.PCP) error
	DeletePatientPCP(patientID int64) error
	UpdatePatientEmergencyContacts(patientID int64, emergencyContacts []*common.EmergencyContact) error
	GetPatientPCP(patientID int64) (*common.PCP, error)
	GetPatientEmergencyContacts(patientID int64) ([]*common.EmergencyContact, error)
	GetActiveMembersOfCareTeamForPatient(patientID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
}

type MedicalRecordUpdate struct {
	Status     *common.MedicalRecordStatus
	Error      *string
	StorageURL *string
	Completed  *time.Time
}

type MedicalRecordAPI interface {
	MedicalRecordsForPatient(patientID int64) ([]*common.MedicalRecord, error)
	MedicalRecord(id int64) (*common.MedicalRecord, error)
	CreateMedicalRecord(patientID int64) (int64, error)
	UpdateMedicalRecord(id int64, update *MedicalRecordUpdate) error
}

type PatientCaseAPI interface {
	GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error)
	GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error)
	GetActiveMembersOfCareTeamForCase(patientCaseID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
	AssignDoctorToPatientFileAndCase(doctorID int64, patientCase *common.PatientCase) error
	GetPatientCaseFromPatientVisitID(patientVisitID int64) (*common.PatientCase, error)
	GetPatientCaseFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientCase, error)
	GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error)
	DoesActiveTreatmentPlanForCaseExist(patientCaseID int64) (bool, error)
	GetActiveTreatmentPlanForCase(patientCaseID int64) (*common.TreatmentPlan, error)
	GetTreatmentPlansForCase(patientCaseID int64) ([]*common.TreatmentPlan, error)
	DeleteDraftTreatmentPlanByDoctorForCase(doctorID, patientCaseID int64) error
	GetCasesForPatient(patientID int64) ([]*common.PatientCase, error)
	GetVisitsForCase(patientCaseID int64, statuses []string) ([]*common.PatientVisit, error)
	GetNotificationsForCase(patientCaseID int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error)
	GetNotificationCountForCase(patientCaseID int64) (int64, error)
	InsertCaseNotification(caseNotificationItem *common.CaseNotification) error
	DeleteCaseNotification(uid string, patientCaseID int64) error
}

type DoctorNotify struct {
	DoctorID     int64
	LastNotified *time.Time
}

type CaseRouteAPI interface {
	TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorID int64, patientCase *common.PatientCase, duration time.Duration) error
	TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(doctorID int64, patientCase *common.PatientCase) error
	PermanentlyAssignDoctorToCaseAndRouteToQueue(doctorID int64, patientCase *common.PatientCase, queueItem *DoctorQueueItem) error
	ExtendClaimForDoctor(doctorID, patientID, patientCaseID int64, duration time.Duration) error
	GetClaimedItemsInQueue() ([]*DoctorQueueItem, error)
	GetTempClaimedCaseInQueue(patientCaseID, doctorID int64) (*DoctorQueueItem, error)
	GetElligibleItemsInUnclaimedQueue(doctorID int64) ([]*DoctorQueueItem, error)
	GetAllItemsInUnclaimedQueue() ([]*DoctorQueueItem, error)
	OldestUnclaimedItems(maxItems int) ([]*ItemAge, error)
	InsertUnclaimedItemIntoQueue(doctorQueueItem *DoctorQueueItem) error
	RevokeDoctorAccessToCase(patientCaseID, patientID, doctorID int64) error
	CareProvidingStatesWithUnclaimedCases() ([]int64, error)
	DoctorsToNotifyInCareProvidingState(careProvidingStateID int64, avoidDoctorsRegisteredInStates []int64, timeThreshold time.Time) ([]*DoctorNotify, error)
	RecordDoctorNotifiedOfUnclaimedCases(doctorID int64) error
	RecordCareProvidingStateNotified(careProvidingStateID int64) error
	LastNotifiedTimeForCareProvidingState(careProvidingStateID int64) (time.Time, error)
}

type PatientVisitUpdate struct {
	Status          *string
	LayoutVersionID *int64
	ClosedDate      *time.Time
}

type ItemAge struct {
	ID  int64
	Age time.Duration
}

type PatientVisitAPI interface {
	GetLastCreatedPatientVisit(patientID int64) (*common.PatientVisit, error)
	GetPatientIDFromPatientVisitID(patientVisitID int64) (int64, error)
	GetLatestSubmittedPatientVisit() (*common.PatientVisit, error)
	GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error)
	GetPatientVisitFromID(patientVisitID int64) (*common.PatientVisit, error)
	GetPatientVisitForSKU(patientID int64, skuType sku.SKU) (*common.PatientVisit, error)
	GetPatientVisitFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientVisit, error)
	GetPatientCaseIDFromPatientVisitID(patientVisitID int64) (int64, error)
	PendingFollowupVisitForCase(caseID int64) (*common.PatientVisit, error)
	CreatePatientVisit(visit *common.PatientVisit) (int64, error)
	SetMessageForPatientVisit(patientVisitID int64, message string) error
	GetMessageForPatientVisit(patientVisitID int64) (string, error)
	StartNewTreatmentPlan(patientID, patientVisitID, doctorID int64, parent *common.TreatmentPlanParent, contentSource *common.TreatmentPlanContentSource) (int64, error)
	GetAbridgedTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error)
	UpdateTreatmentPlanStatus(treatmentPlanID int64, status common.TreatmentPlanStatus) error
	GetTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error)
	GetAbridgedTreatmentPlanList(doctorID, patientID int64, statuses []common.TreatmentPlanStatus) ([]*common.TreatmentPlan, error)
	GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, patientID int64) ([]*common.TreatmentPlan, error)
	DeleteTreatmentPlan(treatmentPlanID int64) error
	GetPatientIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error)
	UpdatePatientVisit(id int64, update *PatientVisitUpdate) error
	UpdatePatientVisits(ids []int64, update *PatientVisitUpdate) error
	ClosePatientVisit(patientVisitID int64, event string) error
	ActivateTreatmentPlan(treatmentPlanID, doctorID int64) error
	SubmitPatientVisitWithID(patientVisitID int64) error
	CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan) error
	GetRegimenPlanForTreatmentPlan(treatmentPlanID int64) (regimenPlan *common.RegimenPlan, err error)
	AddTreatmentsForTreatmentPlan(treatments []*common.Treatment, doctorID, treatmentPlanID, patientID int64) error
	GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID int64) ([]*common.Treatment, error)
	GetTreatmentBasedOnPrescriptionID(erxID int64) (*common.Treatment, error)
	GetTreatmentsForPatient(patientID int64) ([]*common.Treatment, error)
	GetTreatmentFromID(treatmentID int64) (*common.Treatment, error)
	GetActiveTreatmentPlansForPatient(patientID int64) ([]*common.TreatmentPlan, error)
	GetTreatmentPlanForPatient(patientID, treatmentPlanID int64) (*common.TreatmentPlan, error)
	IsRevisedTreatmentPlan(treatmentPlanID int64) (bool, error)
	StartRXRoutingForTreatmentsAndTreatmentPlan(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, treatmentPlanID, doctorID int64) error
	UpdateTreatmentWithPharmacyAndErxID(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error
	AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error
	GetPrescriptionStatusEventsForPatient(erxPatientID int64) ([]common.StatusEvent, error)
	GetPrescriptionStatusEventsForTreatment(treatmentID int64) ([]common.StatusEvent, error)
	MarkTPDeviatedFromContentSource(treatmentPlanID int64) error
	GetOldestVisitsInStatuses(max int, statuses []string) ([]*ItemAge, error)
	UpdateDiagnosisForVisit(id, doctorID int64, diagnosis string) error
	DiagnosisForVisit(visitID int64) (string, error)
}

type RefillRequestDenialReason struct {
	ID           int64  `json:"id,string"`
	DenialCode   string `json:"denial_code"`
	DenialReason string `json:"denial_reason"`
}

type PrescriptionsAPI interface {
	GetPendingRefillRequestStatusEventsForClinic() ([]common.StatusEvent, error)
	GetApprovedOrDeniedRefillRequestsForPatient(patientID int64) ([]common.StatusEvent, error)
	GetRefillStatusEventsForRefillRequest(refillRequestID int64) ([]common.StatusEvent, error)
	CreateRefillRequest(*common.RefillRequestItem) error
	AddRefillRequestStatusEvent(refillRequestStatus common.StatusEvent) error
	GetRefillRequestFromID(refillRequestID int64) (*common.RefillRequestItem, error)
	GetRefillRequestFromPrescriptionID(prescriptionID int64) (*common.RefillRequestItem, error)
	GetRefillRequestsForPatient(patientID int64) ([]*common.RefillRequestItem, error)
	GetRefillRequestDenialReasons() ([]*RefillRequestDenialReason, error)
	MarkRefillRequestAsApproved(prescriptionID, approvedRefillCount, rxRefillRequestID int64, comments string) error
	MarkRefillRequestAsDenied(prescriptionID, denialReasonID, rxRefillRequestID int64, comments string) error
	LinkRequestedPrescriptionToOriginalTreatment(requestedTreatment *common.Treatment, patient *common.Patient) error
	AddUnlinkedTreatmentInEventOfDNTF(treatment *common.Treatment, refillRequestID int64) error
	GetUnlinkedDNTFTreatment(treatmentID int64) (*common.Treatment, error)
	GetUnlinkedDNTFTreatmentsForPatient(patientID int64) ([]*common.Treatment, error)
	GetUnlinkedDNTFTreatmentFromPrescriptionID(prescriptionID int64) (*common.Treatment, error)
	AddTreatmentToTreatmentPlanInEventOfDNTF(treatment *common.Treatment, refillRequestID int64) error
	UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxID(treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error
	AddErxStatusEventForDNTFTreatment(statusEvent common.StatusEvent) error
	GetErxStatusEventsForDNTFTreatment(treatmentID int64) ([]common.StatusEvent, error)
	GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(patientID int64) ([]common.StatusEvent, error)
}

type DrugAPI interface {
	DoesDrugDetailsExist(ndc string) (bool, error)
	ExistingDrugDetails(ndcs []string) ([]string, error)
	DrugDetails(ndc string) (*common.DrugDetails, error)
	ListDrugDetails() ([]*common.DrugDetails, error)
	SetDrugDetails(ndcToDrugDetails map[string]*common.DrugDetails) error
}

type Provider struct {
	ProviderID   int64
	ProviderRole string
}

type DoctorManagementAPI interface {
	GetCareProvidingStateID(stateAbbreviation string, healthConditionID int64) (int64, error)
	AddCareProvidingState(stateAbbreviation, fullStateName string, healthConditionID int64) (int64, error)
	MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID int64) error
	GetDoctorWithEmail(email string) (*common.Doctor, error)
}

type TreatmentPlanAge struct {
	ID  int64
	Age time.Duration
}

type DoctorAPI interface {
	RegisterDoctor(doctor *common.Doctor) (int64, error)
	GetAccountIDFromDoctorID(doctorID int64) (int64, error)
	UpdateDoctor(doctorID int64, req *DoctorUpdate) error
	GetDoctorFromID(doctorID int64) (doctor *common.Doctor, err error)
	Doctor(id int64, basicInfoOnly bool) (doctor *common.Doctor, err error)
	GetDoctorFromAccountID(accountID int64) (doctor *common.Doctor, err error)
	GetDoctorFromDoseSpotClinicianID(clincianID int64) (doctor *common.Doctor, err error)
	GetDoctorIDFromAccountID(accountID int64) (int64, error)
	GetMAInClinic() (*common.Doctor, error)
	GetRegimenStepsForDoctor(doctorID int64) ([]*common.DoctorInstructionItem, error)
	GetRegimenStepForDoctor(regimenStepID, doctorID int64) (*common.DoctorInstructionItem, error)
	AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorID int64) error
	UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorID int64) error
	MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorID int64) error
	MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorID int64) error
	MarkPatientVisitAsOngoingInDoctorQueue(doctorID, patientVisitID int64) error
	GetPendingItemsInDoctorQueue(doctorID int64) (doctorQueue []*DoctorQueueItem, err error)
	GetCompletedItemsInDoctorQueue(doctorID int64) (doctorQueue []*DoctorQueueItem, err error)
	GetPendingItemsForClinic() ([]*DoctorQueueItem, error)
	GetCompletedItemsForClinic() ([]*DoctorQueueItem, error)
	GetPendingItemCountForDoctorQueue(doctorID int64) (int64, error)
	GetMedicationDispenseUnits(languageID int64) (dispenseUnitIDs []int64, dispenseUnits []string, err error)
	AddTreatmentTemplates(treatments []*common.DoctorTreatmentTemplate, doctorID, treatmentPlanID int64) error
	GetTreatmentTemplates(doctorID int64) ([]*common.DoctorTreatmentTemplate, error)
	DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorID int64) error
	InsertItemIntoDoctorQueue(doctorQueueItem DoctorQueueItem) error
	ReplaceItemInDoctorQueue(doctorQueueItem DoctorQueueItem, currentState string) error
	DeleteItemFromDoctorQueue(doctorQueueItem DoctorQueueItem) error
	CompleteVisitOnTreatmentPlanGeneration(doctorID, patientVisitID, treatmentPlanID int64, currentState, updatedState string) error
	SetTreatmentPlanNote(doctorID, treatmentPlanID int64, note string) error
	GetTreatmentPlanNote(treatmentPlanID int64) (string, error)
	DoctorAttributes(doctorID int64, names []string) (map[string]string, error)
	UpdateDoctorAttributes(doctorID int64, attributes map[string]string) error
	AddMedicalLicenses([]*common.MedicalLicense) error
	MedicalLicenses(doctorID int64) ([]*common.MedicalLicense, error)
	CareProviderProfile(accountID int64) (*common.CareProviderProfile, error)
	UpdateCareProviderProfile(accountID int64, profile *common.CareProviderProfile) error
	GetFirstDoctorWithAClinicianID() (*common.Doctor, error)
	GetOldestTreatmentPlanInStatuses(max int, statuses []common.TreatmentPlanStatus) ([]*TreatmentPlanAge, error)
	DoctorEligibleToTreatInState(state string, doctorID, healthConditionID int64) (bool, error)
	PatientCaseFeed() ([]*common.PatientCaseFeedItem, error)
	PatientCaseFeedForDoctor(doctorID int64) ([]*common.PatientCaseFeedItem, error)
	UpdatePatientCaseFeedItem(item *common.PatientCaseFeedItem) error
	// DEPRECATED: Remove after Buzz Lightyear release
	GetSavedDoctorNote(doctorID int64) (string, error)
}

type ClinicAPI interface {
	GetAllDoctorsInClinic() ([]*common.Doctor, error)
}

type FavoriteTreatmentPlanAPI interface {
	CreateOrUpdateFavoriteTreatmentPlan(favoriteTreatmentPlan *common.FavoriteTreatmentPlan, treatmentPlanID int64) error
	GetFavoriteTreatmentPlansForDoctor(doctorID int64) ([]*common.FavoriteTreatmentPlan, error)
	GetFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) (*common.FavoriteTreatmentPlan, error)
	DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanID, doctorID int64) error
	GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) ([]*common.Treatment, error)
	GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) (*common.RegimenPlan, error)
}

type ColumnValue struct {
	Column string
	Value  interface{}
}

type IntakeInfo interface {
	TableName() string
	Role() *ColumnValue
	Context() *ColumnValue
	SessionID() string
	SessionCounter() uint
	LayoutVersionID() int64
	Answers() map[int64][]*common.AnswerIntake
}

type IntakeAPI interface {
	PatientPhotoSectionsForQuestionIDs(questionIDs []int64, patientID, patientVisitID int64) (map[int64][]common.Answer, error)
	PreviousPatientAnswersForQuestions(questionIDs []int64, patientID int64, beforeTime time.Time) (map[int64][]common.Answer, error)
	AnswersForQuestions(questionIDs []int64, info IntakeInfo) (map[int64][]common.Answer, error)
	StoreAnswersForQuestion(info IntakeInfo) error
	StorePhotoSectionsForQuestion(questionID, patientID, patientVisitID int64, sessionID string, sessionCounter uint, photoSections []*common.PhotoIntakeSection) error
}

type VersionInfo struct {
	Major *int
	Minor *int
	Patch *int
}

type LayoutTemplateVersion struct {
	ID                int64
	Layout            []byte
	Version           common.Version
	Role              string
	Purpose           string
	HealthConditionID int64
	SKUID             *int64
	Status            string
}

type LayoutVersion struct {
	ID                      int64
	Layout                  []byte
	Version                 common.Version
	LayoutTemplateVersionID int64
	Purpose                 string
	HealthConditionID       int64
	SKUID                   *int64
	LanguageID              int64
	Status                  string
}

type IntakeLayoutAPI interface {
	CreateLayoutTemplateVersion(layout *LayoutTemplateVersion) error
	CreateLayoutVersion(layout *LayoutVersion) error
	CreateLayoutMapping(intakeMajor, intakeMinor, reviewMajor, reviewMinor int, healthConditionID int64, skuType sku.SKU) error
	IntakeLayoutForReviewLayoutVersion(reviewMajor, reviewMinor int, healthConditionID int64, skuType sku.SKU) ([]byte, int64, error)
	ReviewLayoutForIntakeLayoutVersionID(layoutVersionID int64, healthConditionID int64, skuType sku.SKU) ([]byte, int64, error)
	ReviewLayoutForIntakeLayoutVersion(intakeMajor, intakeMinor int, healthConditionID int64, skuType sku.SKU) ([]byte, int64, error)
	IntakeLayoutForAppVersion(appVersion *common.Version, platform common.Platform, healthConditionID, languageID int64, skuType sku.SKU) ([]byte, int64, error)
	IntakeLayoutVersionIDForAppVersion(appVersion *common.Version, platform common.Platform, healthConditionID, languageID int64, skuType sku.SKU) (int64, error)
	CreateAppVersionMapping(appVersion *common.Version, platform common.Platform, layoutMajor int, role, purpose string, healthConditionID int64, skuType sku.SKU) error
	UpdateActiveLayouts(purpose string, version *common.Version, layoutTemplateID int64, layoutIDs []int64, healthConditionID int64, skuID *int64) error
	LatestAppVersionSupported(healthConditionID int64, skuID *int64, platform common.Platform, role, purpose string) (*common.Version, error)
	LayoutTemplateVersionBeyondVersion(versionInfo *VersionInfo, role, purpose string, healthConditionID int64, skuID *int64) (*LayoutTemplateVersion, error)
	GetActiveDoctorDiagnosisLayout(healthConditionID int64) (*LayoutVersion, error)
	GetPatientLayout(layoutVersionID, languageID int64) (*LayoutVersion, error)
	GetLayoutVersionIDOfActiveDiagnosisLayout(healthConditionID int64) (int64, error)
	GetSectionIDsForHealthCondition(healthConditionID int64) ([]int64, error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageID int64) (id int64, title string, err error)
	GetQuestionType(questionID int64) (questionType string, err error)
	GetQuestionInfo(questionTag string, languageID int64) (*info_intake.Question, error)
	GetQuestionInfoForTags(questionTags []string, languageID int64) ([]*info_intake.Question, error)
	GetAnswerInfo(questionID int64, languageID int64) (answerInfos []*info_intake.PotentialAnswer, err error)
	GetAnswerInfoForTags(answerTags []string, languageID int64) ([]*info_intake.PotentialAnswer, error)
	GetTipSectionInfo(tipSectionTag string, languageID int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageID int64) (id int64, tip string, err error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
	GetPhotoSlots(questionID, languageID int64) ([]*info_intake.PhotoSlot, error)
}

type ObjectStorageDBAPI interface {
	CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error)
	UpdateCloudObjectRecordToSayCompleted(id int64) error
}

type PeopleAPI interface {
	GetPeople(ids []int64) (map[int64]*common.Person, error)
	GetPersonIDByRole(roleType string, roleID int64) (int64, error)
}

type CaseMessageAPI interface {
	CreateCaseMessage(msg *common.CaseMessage) (int64, error)
	ListCaseMessages(caseID int64, role string) ([]*common.CaseMessage, error)
	CaseMessageParticipants(caseID int64, withRoleObjects bool) (map[int64]*common.CaseMessageParticipant, error)
	MarkCaseMessagesAsRead(caseID, personID int64) error
	GetCaseIDFromMessageID(messageID int64) (int64, error)
	CaseMessageForAttachment(itemType string, itemID, senderPersonID, patientCaseID int64) (*common.CaseMessage, error)
}

type NotificationAPI interface {
	GetPushConfigData(deviceToken string) (*common.PushConfigData, error)
	DeletePushCommunicationPreferenceForAccount(accountID int64) error
	GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error)
	SetOrReplacePushConfigData(pConfigData *common.PushConfigData) error
	GetCommunicationPreferencesForAccount(accountID int64) ([]*common.CommunicationPreference, error)
	SetPushPromptStatus(patientID int64, pStatus common.PushPromptStatus) error
	SnoozeConfigsForAccount(accountID int64) ([]*common.SnoozeConfig, error)
}

type MediaAPI interface {
	AddMedia(uploaderID int64, url, mimetype string) (int64, error)
	GetMedia(mediaID int64) (*common.Media, error)
	ClaimMedia(mediaID int64, claimerType string, claimerID int64) error
}

type ResourceLibraryAPI interface {
	ListResourceGuideSections() ([]*common.ResourceGuideSection, error)
	GetResourceGuide(id int64) (*common.ResourceGuide, error)
	ListResourceGuides(withLayouts bool) ([]*common.ResourceGuideSection, map[int64][]*common.ResourceGuide, error)
	ReplaceResourceGuides(sections []*common.ResourceGuideSection, guides map[int64][]*common.ResourceGuide) error
	CreateResourceGuideSection(*common.ResourceGuideSection) (int64, error)
	UpdateResourceGuideSection(*common.ResourceGuideSection) error
	CreateResourceGuide(*common.ResourceGuide) (int64, error)
	UpdateResourceGuide(*common.ResourceGuide) error
}

type GeoAPI interface {
	GetFullNameForState(state string) (string, error)
	ListStates() ([]*common.State, error)
}

type BankingAPI interface {
	AddBankAccount(bankAccount *common.BankAccount) (int64, error)
	DeleteBankAccount(id int64) error
	ListBankAccounts(userAccountID int64) ([]*common.BankAccount, error)
	UpdateBankAccountVerficiation(id int64, amount1, amount2 int, transfer1ID, transfer2ID string, expires time.Time, verified bool) error
}

type PatientReceiptUpdate struct {
	Status         *common.PatientReceiptStatus
	StripeChargeID *string
	CreditCardID   *int64
}

type CostAPI interface {
	GetActiveItemCost(itemType sku.SKU) (*common.ItemCost, error)
	GetItemCost(id int64) (*common.ItemCost, error)
	SKUID(skuType sku.SKU) (int64, error)
	CreatePatientReceipt(receipt *common.PatientReceipt) error
	GetPatientReceipt(patientID, itemID int64, itemType sku.SKU, includeLineItems bool) (*common.PatientReceipt, error)
	UpdatePatientReceipt(id int64, update *PatientReceiptUpdate) error
	CreateDoctorTransaction(*common.DoctorTransaction) error
	TransactionsForDoctor(doctorID int64) ([]*common.DoctorTransaction, error)
	TransactionForItem(itemID, doctorID int64, skuType sku.SKU) (*common.DoctorTransaction, error)
}

type SearchAPI interface {
	SearchDoctors(query string) ([]*common.DoctorSearchResult, error)
}

type AnalyticsAPI interface {
	AnalyticsReport(id int64) (*common.AnalyticsReport, error)
	ListAnalyticsReports() ([]*common.AnalyticsReport, error)
	CreateAnalyticsReport(ownerAccountID int64, name, query, presentation string) (int64, error)
	UpdateAnalyticsReport(id int64, name, query, presentation *string) error
}

type TrainingCasesAPI interface {
	TrainingCaseSetCount(status string) (int, error)
	CreateTrainingCaseSet(status string) (int64, error)
	ClaimTrainingSet(doctorID, healthConditionID int64) error
	QueueTrainingCase(*common.TrainingCase) error
	UpdateTrainingCaseSetStatus(id int64, status string) error
}

type EmailAPI interface {
	ListEmailSenders() ([]*common.EmailSender, error)
	ListEmailTemplates(typeKey string) ([]*common.EmailTemplate, error)
	CreateEmailSender(name, address string) (int64, error)
	CreateEmailTemplate(tmpl *common.EmailTemplate) (int64, error)
	GetEmailSender(id int64) (*common.EmailSender, error)
	GetEmailTemplate(id int64) (*common.EmailTemplate, error)
	GetActiveEmailTemplateForType(typeKey string) (*common.EmailTemplate, error)
	UpdateEmailTemplate(id int64, update *EmailTemplateUpdate) error
}

type ScheduledMessageAPI interface {
	CreateScheduledMessage(*common.ScheduledMessage) error
	CreateScheduledMessageTemplate(*common.ScheduledMessageTemplate) error
	DeleteScheduledMessageTemplate(id int64) error
	ListScheduledMessageTemplates() ([]*common.ScheduledMessageTemplate, error)
	RandomlyPickAndStartProcessingScheduledMessage(messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error)
	ScheduledMessage(id int64, messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error)
	ScheduledMessageTemplate(id int64) (*common.ScheduledMessageTemplate, error)
	ScheduledMessageTemplates(eventType string) ([]*common.ScheduledMessageTemplate, error)
	UpdateScheduledMessage(id int64, status common.ScheduledMessageStatus) error
	UpdateScheduledMessageTemplate(*common.ScheduledMessageTemplate) error
}

type AccountPromotionUpdate struct {
	Status        *common.PromotionStatus
	PromotionData common.Typed
}

type PromotionsAPI interface {
	LookupPromoCode(code string) (*common.PromoCode, error)
	PromoCodeForAccountExists(accountID, codeID int64) (bool, error)
	PromotionCountInGroupForAccount(accountID int64, group string) (int, error)
	PromoCodePrefixes() ([]string, error)
	PendingPromotionsForAccount(accountID int64, types map[string]reflect.Type) ([]*common.AccountPromotion, error)
	CreatePromoCodePrefix(prefix string) error
	CreatePromotionGroup(promotionGroup *common.PromotionGroup) (int64, error)
	PromotionGroup(name string) (*common.PromotionGroup, error)
	Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error)
	CreateReferralProgramTemplate(template *common.ReferralProgramTemplate) (int64, error)
	ActiveReferralProgramTemplate(role string, types map[string]reflect.Type) (*common.ReferralProgramTemplate, error)
	ReferralProgram(codeID int64, types map[string]reflect.Type) (*common.ReferralProgram, error)
	ActiveReferralProgramForAccount(accountID int64, types map[string]reflect.Type) (*common.ReferralProgram, error)
	CreatePromotion(promotion *common.Promotion) error
	CreateReferralProgram(referralProgram *common.ReferralProgram) error
	UpdateReferralProgram(accountID int64, codeID int64, data common.Typed) error
	CreateAccountPromotion(accountPromotion *common.AccountPromotion) error
	UpdateAccountPromotion(accountID, promoCodeID int64, update *AccountPromotionUpdate) error
	UpdateCredit(accountID int64, credit int, currency string) error
	AccountCredit(accountID int64) (*common.AccountCredit, error)
	PendingReferralTrackingForAccount(accountID int64) (*common.ReferralTrackingEntry, error)
	TrackAccountReferral(referralTracking *common.ReferralTrackingEntry) error
	UpdateAccountReferral(accountID int64, status common.ReferralTrackingStatus) error
	CreateParkedAccount(parkedAccount *common.ParkedAccount) (int64, error)
	ParkedAccount(email string) (*common.ParkedAccount, error)
	MarkParkedAccountAsAccountCreated(id int64) error
}

type DataAPI interface {
	AdminAPI
	AnalyticsAPI
	BankingAPI
	CaseMessageAPI
	CaseRouteAPI
	ClinicAPI
	CostAPI
	DoctorAPI
	DoctorManagementAPI
	DrugAPI
	EmailAPI
	FavoriteTreatmentPlanAPI
	FormAPI
	GeoAPI
	IntakeAPI
	IntakeLayoutAPI
	MediaAPI
	MedicalRecordAPI
	NotificationAPI
	ObjectStorageDBAPI
	PatientAPI
	PatientCaseAPI
	PatientVisitAPI
	PeopleAPI
	PrescriptionsAPI
	PromotionsAPI
	ResourceLibraryAPI
	ScheduledMessageAPI
	SearchAPI
	TrainingCasesAPI
}

type CloudStorageAPI interface {
	GetObjectAtLocation(bucket, key, region string) (rawData []byte, responseHeader http.Header, err error)
	GetSignedURLForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error)
	DeleteObjectAtLocation(bucket, key, region string) error
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataAPI DataAPI) (int64, string, error)
}

const (
	LostPassword       = "LostPassword"
	LostPasswordCode   = "LostPasswordCode"
	PasswordReset      = "PasswordReset"
	TwoFactorAuthToken = "TwoFactorAuthToken"
	TwoFactorAuthCode  = "TwoFactorAuthCode"
)

type Platform string

const (
	Mobile      Platform = "mobile"
	Web         Platform = "web"
	RegularAuth bool     = false
)

type AppInfo struct {
	Version         *common.Version
	Build           string
	Platform        common.Platform
	PlatformVersion string
	Device          string
	DeviceModel     string
	LastSeen        time.Time
}

type AuthAPI interface {
	Authenticate(email, password string) (*common.Account, error)
	CreateAccount(email, password, roleType string) (int64, error)
	CreateToken(accountID int64, platform Platform, extended bool) (string, error)
	DeleteToken(token string) error
	GetAccount(id int64) (*common.Account, error)
	GetAccountForEmail(email string) (*common.Account, error)
	GetPhoneNumbersForAccount(id int64) ([]*common.PhoneNumber, error)
	GetToken(accountID int64) (string, error)
	ReplacePhoneNumbersForAccount(accountID int64, numbers []*common.PhoneNumber) error
	SetPassword(accountID int64, password string) error
	UpdateAccount(accountID int64, email *string, twoFactorEnabled *bool) error
	UpdateLastOpenedDate(accountID int64) error
	ValidateToken(token string, platform Platform) (*common.Account, error)
	TimezoneForAccount(id int64) (string, error)
	// Devices
	GetAccountDevice(accountID int64, deviceID string) (*common.AccountDevice, error)
	UpdateAccountDeviceVerification(accountID int64, deviceID string, verified bool) error
	// Temporary auth tokens
	CreateTempToken(accountID int64, expireSec int, purpose, token string) (string, error)
	ValidateTempToken(purpose, token string) (*common.Account, error)
	DeleteTempToken(purpose, token string) error
	DeleteTempTokensForAccount(accountID int64) error
	// Permissions
	AvailableAccountPermissions() ([]string, error)
	AvailableAccountGroups(withPermissions bool) ([]*common.AccountGroup, error)
	PermissionsForAccount(accountID int64) ([]string, error)
	GroupsForAccount(accountID int64) ([]*common.AccountGroup, error)
	UpdateGroupsForAccount(accountID int64, groups map[int64]bool) error
	UpdateAppDevice(accountID int64, appVersion *common.Version, p common.Platform, platformVersion, device, deviceModel, build string) error
	LatestAppInfo(accountID int64) (*AppInfo, error)
}

type SMSAPI interface {
	Send(fromNumber, toNumber, text string) error
}

type Form interface {
	TableColumnValues() (string, []string, []interface{})
}

type FormAPI interface {
	RecordForm(form Form, source string, requestID int64) error
}

type LockAPI interface {
	Locked() bool
	Wait() bool
	Release()
}

type AdminAPI interface {
	Dashboard(id int64) (*common.AdminDashboard, error)
}

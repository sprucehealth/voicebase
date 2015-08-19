package api

import (
	"fmt"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/patient_case/model"
	"github.com/sprucehealth/backend/pharmacy"
)

// Account role types
const (
	RoleAdmin   = "ADMIN"
	RoleDoctor  = "DOCTOR"
	RoleCC      = "MA"
	RolePatient = "PATIENT"
)

const (
	LanguageIDEnglish = 1
	PatientUnlinked   = "UNLINKED"
	PatientRegistered = "REGISTERED"
	DoctorRegistered  = "REGISTERED"

	// TODO: This is temporary until we remove all hardcoded cases of pathways
	AcnePathwayTag = "health_condition_acne"

	MinimumPasswordLength  = 6
	ReviewPurpose          = "REVIEW"
	ConditionIntakePurpose = "CONDITION_INTAKE"
	DiagnosePurpose        = "DIAGNOSE"
)

// ErrNotFound is used to signal that an object is not found. The
// string value will be the name of the type of object.
type ErrNotFound string

// Error implements the error interface.
func (e ErrNotFound) Error() string {
	return fmt.Sprintf("object of type '%s' not found", string(e))
}

// IsErrNotFound returns true iff the provided error is api.ErrNotFound.
func IsErrNotFound(err error) bool {
	_, ok := errors.Cause(err).(ErrNotFound)
	return ok
}

type PatientAPI interface {
	Patient(id common.PatientID, basicInfoOnly bool) (*common.Patient, error)
	Patients(ids []common.PatientID) (map[common.PatientID]*common.Patient, error)
	GetPatientFromID(patientID common.PatientID) (patient *common.Patient, err error)
	GetPatientFromAccountID(accountID int64) (patient *common.Patient, err error)
	GetPatientFromErxPatientID(erxPatientID int64) (*common.Patient, error)
	PatientLocation(patientID common.PatientID) (zipcode string, state string, err error)
	AnyVisitSubmitted(patientID common.PatientID) (bool, error)
	RegisterPatient(patient *common.Patient) error
	UpdatePatient(id common.PatientID, update *PatientUpdate, updateFromDoctor bool) error
	CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, pathwayTag string) error
	GetPatientIDFromAccountID(accountID int64) (common.PatientID, error)
	AddDoctorToCareTeamForPatient(patientID common.PatientID, doctorID int64, pathwayTag string) error
	UpdatePatientPharmacy(patientID common.PatientID, pharmacyDetails *pharmacy.PharmacyData) error
	TrackPatientAgreements(patientID common.PatientID, agreements map[string]bool) error
	PatientAgreements(patientID common.PatientID) (map[string]time.Time, error)
	GetPatientFromPatientVisitID(patientVisitID int64) (patient *common.Patient, err error)
	GetPatientFromTreatmentPlanID(treatmentPlanID int64) (patient *common.Patient, err error)
	GetPatientsForIDs(patientIDs []int64) ([]*common.Patient, error)
	GetPharmacyBasedOnReferenceIDAndSource(pharmacyid int64, pharmacySource string) (*pharmacy.PharmacyData, error)
	GetPharmacyFromID(pharmacyLocalID int64) (*pharmacy.PharmacyData, error)
	AddPharmacy(pharmacyDetails *pharmacy.PharmacyData) error
	CreatePendingTask(workType, status string, itemID int64) (int64, error)
	DeletePendingTask(pendingTaskID int64) error
	AddCardForPatient(patientID common.PatientID, card *common.Card) error
	MarkCardInactiveForPatient(patientID common.PatientID, card *common.Card) error
	DeleteCardForPatient(patientID common.PatientID, card *common.Card) error
	MakeLatestCardDefaultForPatient(patientID common.PatientID) (*common.Card, error)
	MakeCardDefaultForPatient(patientID common.PatientID, card *common.Card) error
	GetCardsForPatient(patientID common.PatientID) ([]*common.Card, error)
	GetDefaultCardForPatient(patientID common.PatientID) (*common.Card, error)
	GetCardFromID(cardID int64) (*common.Card, error)
	GetCardFromThirdPartyID(thirdPartyID string) (*common.Card, error)
	UpdateDefaultAddressForPatient(patientID common.PatientID, address *common.Address) error
	DeleteAddress(addressID int64) error
	UpdatePatientPCP(pcp *common.PCP) error
	DeletePatientPCP(patientID common.PatientID) error
	UpdatePatientEmergencyContacts(patientID common.PatientID, emergencyContacts []*common.EmergencyContact) error
	GetPatientPCP(patientID common.PatientID) (*common.PCP, error)
	GetPatientEmergencyContacts(patientID common.PatientID) ([]*common.EmergencyContact, error)
	GetActiveMembersOfCareTeamForPatient(patientID common.PatientID, fillInDetails bool) ([]*common.CareProviderAssignment, error)
}

type PathwayOption int

const (
	POActiveOnly PathwayOption = 1 << iota
	POWithDetails
	PONone PathwayOption = 0
)

type PathwayUpdate struct {
	Name    *string                `json:"name,omitempty"`
	Details *common.PathwayDetails `json:"details,omitempty"`
}

type Pathways interface {
	CreatePathway(pathway *common.Pathway) error
	ListPathways(opts PathwayOption) ([]*common.Pathway, error)
	Pathway(id int64, opts PathwayOption) (*common.Pathway, error)
	PathwayForTag(tag string, opts PathwayOption) (*common.Pathway, error)
	PathwaysForTags(tags []string, opts PathwayOption) (map[string]*common.Pathway, error)
	PathwayMenu() (*common.PathwayMenu, error)
	UpdatePathway(id int64, update *PathwayUpdate) error
	UpdatePathwayMenu(menu *common.PathwayMenu) error
	CreatePathwaySTP(pathwayTag string, stp []byte) error
	PathwaySTP(pathwayTag string) ([]byte, error)
}

type MedicalRecordUpdate struct {
	Status     *common.MedicalRecordStatus
	Error      *string
	StorageURL *string
	Completed  *time.Time
}

type MedicalRecordAPI interface {
	MedicalRecordsForPatient(patientID common.PatientID) ([]*common.MedicalRecord, error)
	MedicalRecord(id int64) (*common.MedicalRecord, error)
	CreateMedicalRecord(patientID common.PatientID) (int64, error)
	UpdateMedicalRecord(id int64, update *MedicalRecordUpdate) error
}

type PatientCaseUpdate struct {
	Status      *common.CaseStatus
	ClosedDate  *time.Time
	TimeoutDate NullableTime
}

type PatientCaseAPI interface {
	CasesForPathway(patientID common.PatientID, pathwayTag string, states []string) ([]*common.PatientCase, error)
	TimedOutCases() ([]*common.PatientCase, error)
	GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error)
	GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error)
	GetActiveMembersOfCareTeamForCase(patientCaseID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
	GetPatientCaseFromPatientVisitID(patientVisitID int64) (*common.PatientCase, error)
	GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error)
	AddDoctorToPatientCase(doctorID, caseID int64) error
	DoesActiveTreatmentPlanForCaseExist(patientCaseID int64) (bool, error)
	GetActiveTreatmentPlanForCase(patientCaseID int64) (*common.TreatmentPlan, error)
	GetTreatmentPlansForCase(patientCaseID int64) ([]*common.TreatmentPlan, error)
	DeleteDraftTreatmentPlanByDoctorForCase(doctorID, patientCaseID int64) error
	GetCasesForPatient(patientID common.PatientID, states []string) ([]*common.PatientCase, error)
	CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error)
	GetVisitsForCase(patientCaseID int64, statuses []string) ([]*common.PatientVisit, error)
	GetNotificationsForCase(patientCaseID int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error)
	NotificationsForCases(patientID common.PatientID, notificationTypeRegistry map[string]reflect.Type) (map[int64][]*common.CaseNotification, error)
	GetNotificationCountForCase(patientCaseID int64) (int64, error)
	InsertCaseNotification(caseNotificationItem *common.CaseNotification) error
	DeleteCaseNotification(uid string, patientCaseID int64) error
	UpdatePatientCase(id int64, update *PatientCaseUpdate) error
	InsertPatientCaseNote(n *model.PatientCaseNote) (int64, error)
	UpdatePatientCaseNote(nu *model.PatientCaseNoteUpdate) (int64, error)
	PatientCaseNote(id int64) (*model.PatientCaseNote, error)
	PatientCaseNotes(caseIDs []int64) (map[int64][]*model.PatientCaseNote, error)
	DeletePatientCaseNote(id int64) (int64, error)
}

type DoctorNotify struct {
	DoctorID     int64
	LastNotified *time.Time
}

type CaseRouteAPI interface {
	CareProvidingStatesWithUnclaimedCases() ([]int64, error)
	DoctorsToNotifyInCareProvidingState(careProvidingStateID int64, avoidDoctorsRegisteredInStates []int64, timeThreshold time.Time) ([]*DoctorNotify, error)
	ExtendClaimForDoctor(doctorID, patientID, patientCaseID int64, duration time.Duration) error
	GetAllItemsInUnclaimedQueue() ([]*DoctorQueueItem, error)
	GetClaimedItemsInQueue() ([]*DoctorQueueItem, error)
	GetElligibleItemsInUnclaimedQueue(doctorID int64) ([]*DoctorQueueItem, error)
	GetTempClaimedCaseInQueue(patientCaseID int64) (*DoctorQueueItem, error)
	InsertUnclaimedItemIntoQueue(doctorQueueItem *DoctorQueueItem) error
	LastNotifiedTimeForCareProvidingState(careProvidingStateID int64) (time.Time, error)
	OldestUnclaimedItems(maxItems int) ([]*ItemAge, error)
	PermanentlyAssignDoctorToCaseAndRouteToQueue(doctorID int64, patientCase *common.PatientCase, queueItem *DoctorQueueItem) error
	RecordCareProvidingStateNotified(careProvidingStateID int64) error
	RecordDoctorNotifiedOfUnclaimedCases(doctorID int64) error
	RevokeDoctorAccessToCase(patientCaseID, patientID, doctorID int64) error
	TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorID int64, patientCase *common.PatientCase, duration time.Duration) error
	TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(doctorID int64, patientCase *common.PatientCase) error
}

type PatientUpdate struct {
	FirstName          *string
	MiddleName         *string
	LastName           *string
	Prefix             *string
	Suffix             *string
	DOB                *encoding.Date
	Gender             *string
	PhoneNumbers       []*common.PhoneNumber
	Address            *common.Address
	ERxID              *int64
	StripeCustomerID   *string
	HasParentalConsent *bool
}

// PatientVisitUpdate is the set of values that can be updated for a visit
type PatientVisitUpdate struct {
	Status          *string
	LayoutVersionID *int64
	ClosedDate      *time.Time
	SubmittedDate   *time.Time

	// RequiredStatus if not-nil provides the required value for the status for the update to succeed
	RequiredStatus *string
}

type ItemAge struct {
	ID  int64
	Age time.Duration
}

type TreatmentPlanUpdate struct {
	Status        *common.TreatmentPlanStatus
	PatientViewed *bool
}

type PatientVisitAPI interface {
	GetPatientIDFromPatientVisitID(patientVisitID int64) (common.PatientID, error)
	GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error)
	GetPatientVisitFromID(patientVisitID int64) (*common.PatientVisit, error)
	GetPatientVisitForSKU(patientID common.PatientID, skuType string) (*common.PatientVisit, error)
	VisitsSubmittedForPatientSince(patientID common.PatientID, since time.Time) ([]*common.PatientVisit, error)
	GetPatientCaseIDFromPatientVisitID(patientVisitID int64) (int64, error)
	PendingFollowupVisitForCase(caseID int64) (*common.PatientVisit, error)
	CreatePatientVisit(visit *common.PatientVisit, requestedDoctorID *int64) (int64, error)
	SetMessageForPatientVisit(patientVisitID int64, message string) error
	GetMessageForPatientVisit(patientVisitID int64) (string, error)
	GetPatientIDFromTreatmentPlanID(treatmentPlanID int64) (common.PatientID, error)
	// CaseIDForTreatmentPlan returns the case ID for a treatment plan
	CaseIDForTreatmentPlan(treatmentPlanID int64) (int64, error)
	UpdatePatientVisit(id int64, update *PatientVisitUpdate) (int, error)
	UpdatePatientVisits(ids []int64, update *PatientVisitUpdate) error
	AddTreatmentsForTreatmentPlan(treatments []*common.Treatment, doctorID, treatmentPlanID int64, patientID common.PatientID) error
	GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID int64) ([]*common.Treatment, error)
	GetTreatmentBasedOnPrescriptionID(erxID int64) (*common.Treatment, error)
	GetTreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error)
	GetTreatmentFromID(treatmentID int64) (*common.Treatment, error)
	UpdateTreatmentWithPharmacyAndErxID(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error
	AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error
	GetPrescriptionStatusEventsForPatient(erxPatientID int64) ([]common.StatusEvent, error)
	GetPrescriptionStatusEventsForTreatment(treatmentID int64) ([]common.StatusEvent, error)

	GetOldestVisitsInStatuses(max int, statuses []string) ([]*ItemAge, error)
	UpdateDiagnosisForVisit(id, doctorID int64, diagnosis string) error
	DiagnosisForVisit(visitID int64) (string, error)
	DoesCaseExistForPatient(patientID common.PatientID, patientCaseID int64) (bool, error)

	AddAlertsForVisit(visitID int64, alerts []*common.Alert) error
	AlertsForVisit(visitID int64) ([]*common.Alert, error)

	// treatment plan
	UpdateTreatmentPlan(treatmentPlanID int64, update *TreatmentPlanUpdate) error
	MarkTPDeviatedFromContentSource(treatmentPlanID int64) error
	GetActiveTreatmentPlansForPatient(patientID common.PatientID) ([]*common.TreatmentPlan, error)
	GetTreatmentPlanForPatient(patientID common.PatientID, treatmentPlanID int64) (*common.TreatmentPlan, error)
	IsRevisedTreatmentPlan(treatmentPlanID int64) (bool, error)
	StartRXRoutingForTreatmentsAndTreatmentPlan(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, treatmentPlanID, doctorID int64) error
	CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan) error
	GetRegimenPlanForTreatmentPlan(treatmentPlanID int64) (regimenPlan *common.RegimenPlan, err error)
	ActivateTreatmentPlan(treatmentPlanID, doctorID int64) error
	DeleteTreatmentPlan(treatmentPlanID int64) error
	StartNewTreatmentPlan(patientVisitID int64, tp *common.TreatmentPlan) (int64, error)
	GetAbridgedTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error)
	GetTreatmentPlan(treatmentPlanID, doctorID int64) (*common.TreatmentPlan, error)
	GetAbridgedTreatmentPlanList(doctorID, patientCaseID int64, statuses []common.TreatmentPlanStatus) ([]*common.TreatmentPlan, error)
	GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, patientCaseID int64) ([]*common.TreatmentPlan, error)
	VisitSummaries(visitStatuses []string, createdFrom, createdTo time.Time) ([]*common.VisitSummary, error)
	VisitSummary(visitID int64) (*common.VisitSummary, error)

	// diagnosis set related apis
	CreateDiagnosisSet(set *common.VisitDiagnosisSet) error
	ActiveDiagnosisSet(visitID int64) (*common.VisitDiagnosisSet, error)
}

type RefillRequestDenialReason struct {
	ID           int64  `json:"id,string"`
	DenialCode   string `json:"denial_code"`
	DenialReason string `json:"denial_reason"`
}

type PrescriptionsAPI interface {
	GetPendingRefillRequestStatusEventsForClinic() ([]common.StatusEvent, error)
	GetApprovedOrDeniedRefillRequestsForPatient(patientID common.PatientID) ([]common.StatusEvent, error)
	GetRefillStatusEventsForRefillRequest(refillRequestID int64) ([]common.StatusEvent, error)
	CreateRefillRequest(*common.RefillRequestItem) error
	AddRefillRequestStatusEvent(refillRequestStatus common.StatusEvent) error
	FilterOutRefillRequestsThatExist(queueItemIDs []int64) ([]int64, error)
	GetRefillRequestFromID(refillRequestID int64) (*common.RefillRequestItem, error)
	GetRefillRequestFromPrescriptionID(prescriptionID int64) (*common.RefillRequestItem, error)
	GetRefillRequestsForPatient(patientID common.PatientID) ([]*common.RefillRequestItem, error)
	GetRefillRequestDenialReasons() ([]*RefillRequestDenialReason, error)
	MarkRefillRequestAsApproved(prescriptionID, approvedRefillCount, rxRefillRequestID int64, comments string) error
	MarkRefillRequestAsDenied(prescriptionID, denialReasonID, rxRefillRequestID int64, comments string) error
	LinkRequestedPrescriptionToOriginalTreatment(requestedTreatment *common.Treatment, patient *common.Patient) error
	AddUnlinkedTreatmentInEventOfDNTF(treatment *common.Treatment, refillRequestID int64) error
	GetUnlinkedDNTFTreatment(treatmentID int64) (*common.Treatment, error)
	GetUnlinkedDNTFTreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error)
	GetUnlinkedDNTFTreatmentFromPrescriptionID(prescriptionID int64) (*common.Treatment, error)
	AddTreatmentToTreatmentPlanInEventOfDNTF(treatment *common.Treatment, refillRequestID int64) error
	UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxID(treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error
	AddErxStatusEventForDNTFTreatment(statusEvent common.StatusEvent) error
	GetErxStatusEventsForDNTFTreatment(treatmentID int64) ([]common.StatusEvent, error)
	GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(patientID common.PatientID) ([]common.StatusEvent, error)
}

type DrugDetailsQuery struct {
	NDC         string
	GenericName string
	Route       string
	Form        string
}

// DrugDescription contains information about a drug that uniquely identifies a particular drug.
type DrugDescription struct {
	InternalName    string            `json:"drug_internal_name"`
	DosageStrength  string            `json:"dosage_strength"`
	DrugDBIDs       map[string]string `json:"drug_db_ids"`
	OTC             bool              `json:"otc"`
	Schedule        int               `json:"schedule"`
	DrugName        string            `json:"drug_name"`
	DrugForm        string            `json:"drug_form"`
	DrugRoute       string            `json:"drug_route"`
	GenericDrugName string            `json:"generic_drug_name"`
}

type DrugDescriptionQuery struct {
	InternalName   string
	DosageStrength string
}

type DrugAPI interface {
	SetDrugDescription(description *DrugDescription) error
	DrugDescriptions(queries []*DrugDescriptionQuery) ([]*DrugDescription, error)
	QueryDrugDetails(query *DrugDetailsQuery) (*common.DrugDetails, error)
	MultiQueryDrugDetailIDs(queries []*DrugDetailsQuery) ([]int64, error)
	DrugDetails(id int64) (*common.DrugDetails, error)
	ListDrugDetails() ([]*common.DrugDetails, error)
	SetDrugDetails([]*common.DrugDetails) error
}

type DiagnosisSetPatch struct {
	Title  *string
	Delete []string
	Create []string
}

type DiagnosisAPI interface {
	DiagnosesThatHaveDetails(codeIDs []string) (map[string]bool, error)
	LayoutVersionIDsForDiagnosisCodes(codes map[string]*common.Version) (map[string]int64, error)
	SetDiagnosisDetailsIntake(template, info *common.DiagnosisDetailsIntake) error
	ActiveDiagnosisDetailsIntakeVersion(codeID string) (*common.Version, error)
	ActiveDiagnosisDetailsIntake(codeID string, types map[string]reflect.Type) (*common.DiagnosisDetailsIntake, error)
	DetailsIntakeVersionForDiagnoses(codeIDs []string) (map[string]*common.Version, error)
	DiagnosisDetailsIntake(ids []int64, types map[string]reflect.Type) (map[int64]*common.DiagnosisDetailsIntake, error)
	CommonDiagnosisSet(pathwayTag string) (string, []string, error)
	PatchCommonDiagnosisSet(pathwayTag string, patch *DiagnosisSetPatch) error
}

type Provider struct {
	ID   int64  `json:"id,string"`
	Role string `json:"role"`
}

type CareProviderStatePathway struct {
	ID               int64    `json:"id,string"`
	StateCode        string   `json:"state_code"`
	PathwayTag       string   `json:"pathway_tag"`
	Provider         Provider `json:"provider"`
	ShortDisplayName string   `json:"short_display_name"`
	ThumbnailID      string   `json:"thumbnail_id"`
	Notify           bool     `json:"notify"`
	Unavailable      bool     `json:"unavailable"`
}

type CareProviderStatePathwayMappingUpdate struct {
	ID          int64 `json:"id"`
	Notify      *bool `json:"notify,omitempty"`
	Unavailable *bool `json:"unavailable,omitempty"`
}

type CareProviderStatePathwayMappingPatch struct {
	Delete []int64                                  `json:"delete,omitempty"`
	Create []*CareProviderStatePathway              `json:"create,omitempty"`
	Update []*CareProviderStatePathwayMappingUpdate `json:"update,omitempty"`
}

// CareProviderStatePathwayMappingQuery provides filters when querying the list
// of state pathway mappings. All values are optional in which case all mappings
// are returned.
type CareProviderStatePathwayMappingQuery struct {
	State      string
	PathwayTag string
	Provider   Provider
}

type CareProviderStatePathwayMappingSummary struct {
	StateCode   string `json:"state_code"`
	PathwayTag  string `json:"pathway_tag"`
	DoctorCount int    `json:"doctor_count"`
}

type DoctorManagementAPI interface {
	// AvailableStates returns a list of states with elligible doctors.
	AvailableStates() ([]*common.State, error)
	SpruceAvailableInState(state string) (bool, error)
	GetCareProvidingStateID(stateAbbreviation, pathwayTag string) (int64, error)
	AddCareProvidingState(stateAbbreviation, fullStateName, pathwayTag string) (int64, error)
	MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID int64) error
	// CareProviderEligible returns true if the provided care provider is eligible
	// to see patients in the provided pathway,state combination.
	CareProviderEligible(careProviderID int64, role, state, pathwayTag string) (bool, error)
	GetDoctorWithEmail(email string) (*common.Doctor, error)
	DoctorIDsInCareProvidingState(careProvidingStateID int64) ([]int64, error)
	EligibleDoctorIDs(doctorIDs []int64, careProvidingStateID int64) ([]int64, error)
	AvailableDoctorIDs(n int) ([]int64, error)
	// CareProviderStatePathwayMappings returns a list of care provider / state / pathway mappings.
	CareProviderStatePathwayMappings(query *CareProviderStatePathwayMappingQuery) ([]*CareProviderStatePathway, error)
	CareProviderStatePathwayMappingSummary() ([]*CareProviderStatePathwayMappingSummary, error)
	UpdateCareProviderStatePathwayMapping(patch *CareProviderStatePathwayMappingPatch) error
}

type TreatmentPlanAge struct {
	ID  int64
	Age time.Duration
}

type ListCareProvidersOption int

const (
	LCPOptDoctorsOnly ListCareProvidersOption = 1 << iota
	LCPOptCCOnly
	LCPOptPrimaryCCOnly
)

func (o ListCareProvidersOption) Has(opt ListCareProvidersOption) bool {
	return o&opt == opt
}

type DoctorAPI interface {
	ListCareProviders(opt ListCareProvidersOption) ([]*common.Doctor, error)
	RegisterProvider(provider *common.Doctor, role string) (int64, error)
	GetAccountIDFromDoctorID(doctorID int64) (int64, error)
	UpdateDoctor(doctorID int64, req *DoctorUpdate) error
	GetDoctorFromID(doctorID int64) (doctor *common.Doctor, err error)
	Doctor(id int64, basicInfoOnly bool) (doctor *common.Doctor, err error)
	Doctors(id []int64) ([]*common.Doctor, error)
	GetDoctorFromAccountID(accountID int64) (doctor *common.Doctor, err error)
	GetDoctorFromDoseSpotClinicianID(clincianID int64) (doctor *common.Doctor, err error)
	GetDoctorIDFromAccountID(accountID int64) (int64, error)
	GetRegimenStepsForDoctor(doctorID int64) ([]*common.DoctorInstructionItem, error)
	GetRegimenStepForDoctor(regimenStepID, doctorID int64) (*common.DoctorInstructionItem, error)
	AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorID int64) error
	UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorID int64) error
	MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorID int64) error
	// GetPendingItemsInCCQueues returns all items in all CC inboxes.
	GetPendingItemsInCCQueues() ([]*DoctorQueueItem, error)
	GetPendingItemsInDoctorQueue(doctorID int64) (doctorQueue []*DoctorQueueItem, err error)
	GetCompletedItemsInDoctorQueue(doctorID int64) (doctorQueue []*DoctorQueueItem, err error)
	GetPendingItemsForClinic() ([]*DoctorQueueItem, error)
	GetCompletedItemsForClinic() ([]*DoctorQueueItem, error)
	GetMedicationDispenseUnits(languageID int64) (dispenseUnitIDs []int64, dispenseUnits []string, err error)
	MarkPatientVisitAsOngoingInDoctorQueue(doctorID, patientVisitID int64) error
	AddTreatmentTemplates(treatments []*common.DoctorTreatmentTemplate, doctorID, treatmentPlanID int64) error
	GetTreatmentTemplates(doctorID int64) ([]*common.DoctorTreatmentTemplate, error)
	DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorID int64) error

	UpdateDoctorQueue(updates []*DoctorQueueUpdate) error
	CompleteVisitOnTreatmentPlanGeneration(doctorID, patientVisitID, treatmentPlanID int64,
		updates []*DoctorQueueUpdate) error

	DoctorAttributes(doctorID int64, names []string) (map[string]string, error)
	UpdateDoctorAttributes(doctorID int64, attributes map[string]string) error
	AddMedicalLicenses([]*common.MedicalLicense) error
	UpdateMedicalLicenses(doctorID int64, licenses []*common.MedicalLicense) error
	MedicalLicenses(doctorID int64) ([]*common.MedicalLicense, error)
	CareProviderProfile(accountID int64) (*common.CareProviderProfile, error)
	UpdateCareProviderProfile(accountID int64, profile *common.CareProviderProfile) error
	GetFirstDoctorWithAClinicianID() (*common.Doctor, error)
	GetOldestTreatmentPlanInStatuses(max int, statuses []common.TreatmentPlanStatus) ([]*TreatmentPlanAge, error)
	PatientCaseFeed(caseIDs []int64, start, end *time.Time) ([]*common.PatientCaseFeedItem, error)
	PatientCaseFeedForDoctor(doctorID int64) ([]*common.PatientCaseFeedItem, error)

	// Treatment plan notes
	SetTreatmentPlanNote(doctorID, treatmentPlanID int64, note string) error
	GetTreatmentPlanNote(treatmentPlanID int64) (string, error)

	// Treatment plan scheduled messages
	TreatmentPlanScheduledMessage(id int64) (*common.TreatmentPlanScheduledMessage, error)
	CreateTreatmentPlanScheduledMessage(msg *common.TreatmentPlanScheduledMessage) (int64, error)
	ListTreatmentPlanScheduledMessages(treatmentPlanID int64) ([]*common.TreatmentPlanScheduledMessage, error)
	DeleteTreatmentPlanScheduledMessage(treatmentPlanID, messageID int64) error
	ReplaceTreatmentPlanScheduledMessage(id int64, msg *common.TreatmentPlanScheduledMessage) error
	UpdateTreatmentPlanScheduledMessage(id int64, smID *int64) error

	// Favorite treatment plan scheduled messages
	SetFavoriteTreatmentPlanScheduledMessages(ftpID int64, msgs []*common.TreatmentPlanScheduledMessage) error
	DeleteFavoriteTreatmentPlanScheduledMessages(ftpID int64) error

	// Treatment plan resource guides
	ListTreatmentPlanResourceGuides(tpID int64) ([]*common.ResourceGuide, error)
	AddResourceGuidesToTreatmentPlan(tpID int64, guideIDs []int64) error
	RemoveResourceGuidesFromTreatmentPlan(tpID int64, guideIDs []int64) error
}

type FavoriteTreatmentPlanAPI interface {
	InsertFavoriteTreatmentPlan(ftp *common.FavoriteTreatmentPlan, pathwayTag string, treatmentPlanID int64) (int64, error)
	FavoriteTreatmentPlansForDoctor(doctorID int64, pathwayTag string) (map[string][]*common.FavoriteTreatmentPlan, error)
	FavoriteTreatmentPlan(favoriteTreatmentPlanID int64) (*common.FavoriteTreatmentPlan, error)
	GlobalFavoriteTreatmentPlans(lifecycles []string) ([]*common.FavoriteTreatmentPlan, error)
	DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanID, doctorID int64, pathwayTag string) error
	GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) ([]*common.Treatment, error)
	GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanID int64) (*common.RegimenPlan, error)
	CreateFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error)
	CreateFTPMemberships(memberships []*common.FTPMembership) error
	DeleteFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error)
	FTPMemberships(ftpID int64) ([]*common.FTPMembership, error)
	FTPMembershipsForDoctor(doctorID int64) ([]*common.FTPMembership, error)
	InsertGlobalFTPsAndUpdateMemberships(ftpsByPathwayID map[int64][]*common.FavoriteTreatmentPlan) error
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
	PatientPhotoSectionsForQuestionIDs(questionIDs []int64, patientID common.PatientID, patientVisitID int64) (map[int64][]common.Answer, error)
	PreviousPatientAnswersForQuestions(questionTags []string, patientID common.PatientID, beforeTime time.Time) (map[string][]common.Answer, error)
	AnswersForQuestions(questionIDs []int64, info IntakeInfo) (map[int64][]common.Answer, error)
	StoreAnswersForIntakes(intakes []IntakeInfo) error
	StorePhotoSectionsForQuestion(questionID, patientID, patientVisitID int64, sessionID string, sessionCounter uint, photoSections []*common.PhotoIntakeSection) error
}

type VersionInfo struct {
	Major *int
	Minor *int
	Patch *int
}

type LayoutTemplateVersion struct {
	ID        int64
	Layout    []byte
	Version   common.Version
	Role      string
	Purpose   string
	PathwayID int64
	SKUID     *int64
	Status    string
}

type LayoutVersion struct {
	ID                      int64
	Layout                  []byte
	Version                 common.Version
	LayoutTemplateVersionID int64
	Purpose                 string
	PathwayID               int64
	SKUID                   *int64
	LanguageID              int64
	Status                  string
}

type IntakeLayoutAPI interface {
	CreateLayoutTemplateVersion(layout *LayoutTemplateVersion) error
	CreateLayoutVersion(layout *LayoutVersion) error
	CreateLayoutMapping(intakeMajor, intakeMinor, reviewMajor, reviewMinor int, pathwayID int64, skuType string) error
	IntakeLayoutForReviewLayoutVersion(reviewMajor, reviewMinor int, pathwayID int64, skuType string) ([]byte, int64, error)
	ReviewLayoutForIntakeLayoutVersionID(layoutVersionID int64, pathwayID int64, skuType string) ([]byte, int64, error)
	ReviewLayoutForIntakeLayoutVersion(intakeMajor, intakeMinor int, pathwayID int64, skuType string) ([]byte, int64, error)
	IntakeLayoutForAppVersion(appVersion *common.Version, platform common.Platform, pathwayID, languageID int64, skuType string) ([]byte, int64, error)
	IntakeLayoutVersionIDForAppVersion(appVersion *common.Version, platform common.Platform, pathwayID, languageID int64, skuType string) (int64, error)
	CreateAppVersionMapping(appVersion *common.Version, platform common.Platform, layoutMajor int, role, purpose string, pathwayID int64, skuType string) error
	UpdateActiveLayouts(purpose string, version *common.Version, layoutTemplateID int64, layoutIDs []int64, pathwayID int64, skuID *int64) error
	LatestAppVersionSupported(pathwayID int64, skuID *int64, platform common.Platform, role, purpose string) (*common.Version, error)
	LayoutTemplateVersionBeyondVersion(versionInfo *VersionInfo, role, purpose string, pathwayID int64, skuID *int64) (*LayoutTemplateVersion, error)
	GetActiveDoctorDiagnosisLayout(pathwayID int64) (*LayoutVersion, error)
	GetPatientLayout(layoutVersionID, languageID int64) (*LayoutVersion, error)
	GetLayoutVersionIDOfActiveDiagnosisLayout(pathwayID int64) (int64, error)
	GetSectionIDsForPathway(pathwayID int64) ([]int64, error)
	GetSectionInfo(sectionTag string, languageID int64) (id int64, title string, err error)
	MaxQuestionVersion(questionTag string, languageID int64) (int64, error)
	VersionedQuestionFromID(ID int64) (*common.VersionedQuestion, error)
	VersionedQuestions(questionQueryParams []*QuestionQueryParams) ([]*common.VersionedQuestion, error)
	InsertVersionedQuestion(*common.VersionedQuestion, []*common.VersionedAnswer, []*common.VersionedPhotoSlot, *common.VersionedAdditionalQuestionField) (int64, error)
	VersionedAnswerFromID(int64) (*common.VersionedAnswer, error)
	VersionedAnswers([]*AnswerQueryParams) ([]*common.VersionedAnswer, error)
	VersionedAdditionalQuestionFields(questionID, languageID int64) ([]*common.VersionedAdditionalQuestionField, error)
	VersionedPhotoSlots(questionID, languageID int64) ([]*common.VersionedPhotoSlot, error)
	InsertVersionedPhotoSlot(*common.VersionedPhotoSlot) (int64, error)
	GetQuestionType(questionID int64) (questionType string, err error)
	GetQuestionInfo(questionTag string, languageID, version int64) (*info_intake.Question, error)
	GetQuestionInfoForTags(questionTags []string, languageID int64) ([]*info_intake.Question, error)
	GetAnswerInfo(questionID int64, languageID int64) (answerInfos []*info_intake.PotentialAnswer, err error)
	GetAnswerInfoForTags(answerTags []string, languageID int64) ([]*info_intake.PotentialAnswer, error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
	GetPhotoSlotsInfo(questionID, languageID int64) ([]*info_intake.PhotoSlot, error)
	LayoutVersions() ([]*LayoutVersionInfo, error)
	LayoutTemplate(pathwayTag, sku, purpose string, version *common.Version) ([]byte, error)
}

type PeopleAPI interface {
	GetPeople(ids []int64) (map[int64]*common.Person, error)
	GetPersonIDByRole(roleType string, roleID int64) (int64, error)
}

// ListCaseMessagesOption is the type for options passed to the ListCaseMessages function
type ListCaseMessagesOption int

const (
	// LCMOIncludePrivate returns private messages (between doctor and cc)
	LCMOIncludePrivate ListCaseMessagesOption = 1 << iota
	// LCMOIncludeReadReceipts returns read receipts with the messages
	LCMOIncludeReadReceipts
)

func (o ListCaseMessagesOption) has(opt ListCaseMessagesOption) bool {
	return (o & opt) != 0
}

// CaseMessagesReadOption are options passed to AllCaseMessageRead
type CaseMessagesReadOption int

const (
	// CMROIncludePrivate marks even private messages as read (otherwise only non-private messages are counted)
	CMROIncludePrivate CaseMessagesReadOption = 1 << iota
)

func (o CaseMessagesReadOption) has(opt CaseMessagesReadOption) bool {
	return (o & opt) != 0
}

type CaseMessageAPI interface {
	CaseMessageForAttachment(itemType string, itemID, senderPersonID, patientCaseID int64) (*common.CaseMessage, error)
	CaseMessageParticipants(caseID int64, withRoleObjects bool) (map[int64]*common.CaseMessageParticipant, error)
	// CaseMessagesRead records that a person read each message. If they have previously
	// read a message then the old timestamp is maintained (earliest read timestamp is used).
	CaseMessagesRead(messageIDs []int64, personID int64) error
	// AllCaseMessagesRead records all messages in a case as read. If they have previously
	// read a message then the old timestamp is maintained (earliest read timestamp is used).
	AllCaseMessagesRead(caseID, personID int64, opts CaseMessagesReadOption) error
	CreateCaseMessage(msg *common.CaseMessage) (int64, error)
	GetCaseIDFromMessageID(messageID int64) (int64, error)
	ListCaseMessages(caseID int64, opts ListCaseMessagesOption) ([]*common.CaseMessage, error)
	// UnreadMessageCount returns a person's number of unread case messages.
	UnreadMessageCount(caseID, personID int64) (int, error)
}

type NotificationAPI interface {
	GetPushConfigData(deviceToken string) (*common.PushConfigData, error)
	DeletePushCommunicationPreferenceForAccount(accountID int64) error
	GetPushConfigDataForAccount(accountID int64) ([]*common.PushConfigData, error)
	SetOrReplacePushConfigData(pConfigData *common.PushConfigData) error
	GetCommunicationPreferencesForAccount(accountID int64) ([]*common.CommunicationPreference, error)
	SetPushPromptStatus(accountID int64, pStatus common.PushPromptStatus) error
	SnoozeConfigsForAccount(accountID int64) ([]*common.SnoozeConfig, error)
}

type MediaAPI interface {
	AddMedia(ownerPersonID int64, url, mimetype string) (int64, error)
	GetMedia(mediaID int64) (*common.Media, error)
	ClaimMedia(mediaID int64, claimerType string, claimerID int64) error
	MediaHasClaim(mediaID int64, claimerType string, claimerID int64) (bool, error)
}

type ResourceGuideListOption int

const (
	RGActiveOnly ResourceGuideListOption = 1 << iota
	RGWithLayouts
	RGNone ResourceGuideListOption = 0
)

type ResourceGuideUpdate struct {
	SectionID *int64      `json:"section_id,string"`
	Ordinal   *int        `json:"ordinal"`
	Title     *string     `json:"title"`
	PhotoURL  *string     `json:"photo_url"`
	Layout    interface{} `json:"layout"`
	Active    *bool       `json:"active"`
}

type ResourceLibraryAPI interface {
	ListResourceGuideSections() ([]*common.ResourceGuideSection, error)
	GetResourceGuide(id int64) (*common.ResourceGuide, error)
	GetResourceGuideFromTag(tag string) (*common.ResourceGuide, error)
	ListResourceGuides(opt ResourceGuideListOption) ([]*common.ResourceGuideSection, map[int64][]*common.ResourceGuide, error)
	ReplaceResourceGuides(sections []*common.ResourceGuideSection, guides map[int64][]*common.ResourceGuide) error
	CreateResourceGuideSection(*common.ResourceGuideSection) (int64, error)
	UpdateResourceGuideSection(*common.ResourceGuideSection) error
	CreateResourceGuide(*common.ResourceGuide) (int64, error)
	UpdateResourceGuide(id int64, update *ResourceGuideUpdate) error
}

type GeoAPI interface {
	State(state string) (full string, short string, err error)
	ListStates() ([]*common.State, error)
}

type BankAccountUpdate struct {
	StripeRecipientID *string
	Default           *bool
	Verified          *bool
	VerifyAmount1     *int
	VerifyAmount2     *int
	VerifyTransfer1ID *string
	VerifyTransfer2ID *string
	VerifyExpires     *time.Time
}

type BankingAPI interface {
	AddBankAccount(bankAccount *common.BankAccount) (int64, error)
	DeleteBankAccount(id int64) error
	ListBankAccounts(userAccountID int64) ([]*common.BankAccount, error)
	UpdateBankAccount(id int64, update *BankAccountUpdate) (int, error)
}

type PatientReceiptUpdate struct {
	Status         *common.PatientReceiptStatus
	StripeChargeID *string
}

type CostAPI interface {
	GetActiveItemCost(skuType string) (*common.ItemCost, error)
	GetItemCost(id int64) (*common.ItemCost, error)
	CreatePatientReceipt(receipt *common.PatientReceipt) error
	GetPatientReceipt(patientID common.PatientID, itemID int64, skuType string, includeLineItems bool) (*common.PatientReceipt, error)
	UpdatePatientReceipt(id int64, update *PatientReceiptUpdate) error
	CreateDoctorTransaction(*common.DoctorTransaction) error
	TransactionsForDoctor(doctorID int64) ([]*common.DoctorTransaction, error)
	TransactionForItem(itemID, doctorID int64, skuType string) (*common.DoctorTransaction, error)
	VisitSKUs(activeOnly bool) ([]string, error)
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
	ClaimTrainingSet(doctorID int64, pathwayTag string) error
	QueueTrainingCase(*common.TrainingCase) error
	UpdateTrainingCaseSetStatus(id int64, status string) error
}

type Recipient struct {
	AccountID int64
	Name      string
	Email     string
}

type EmailAPI interface {
	EmailUpdateOptOut(accountID int64, emailType string, optout bool) error
	EmailRecipients(accountIDs []int64) ([]*Recipient, error)
	EmailRecipientsWithOptOut(accountIDs []int64, emailType string, onlyOnce bool) ([]*Recipient, error)
	EmailRecordSend(accountIDs []int64, emailType string) error
	// EmailCampaignState returns the state data for an email campaign. If there's no
	// existing data then it will return nil data and a nil error.
	EmailCampaignState(campaignKey string) ([]byte, error)
	// UpdateEmailCampaignState updates the state data for an email campaign.
	UpdateEmailCampaignState(campaignKey string, state []byte) error
}

type ScheduledMessageAPI interface {
	CreateScheduledMessage(*common.ScheduledMessage) (int64, error)
	CreateScheduledMessageTemplate(*common.ScheduledMessageTemplate) error
	DeleteScheduledMessageTemplate(id int64) error
	ListScheduledMessageTemplates() ([]*common.ScheduledMessageTemplate, error)
	RandomlyPickAndStartProcessingScheduledMessage(messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error)
	ScheduledMessage(id int64, messageTypes map[string]reflect.Type) (*common.ScheduledMessage, error)
	ScheduledMessageTemplate(id int64) (*common.ScheduledMessageTemplate, error)
	ScheduledMessageTemplates(eventType string) ([]*common.ScheduledMessageTemplate, error)
	UpdateScheduledMessage(id int64, status common.ScheduledMessageStatus) error
	UpdateScheduledMessageTemplate(*common.ScheduledMessageTemplate) error
	DeactivateScheduledMessagesForPatient(patientID common.PatientID) (int64, error)
}

type AccountPromotionUpdate struct {
	Status        *common.PromotionStatus
	PromotionData common.Typed
}

type PromotionsAPI interface {
	AccountCode(accountID int64) (*uint64, error)
	AccountCredit(accountID int64) (*common.AccountCredit, error)
	AccountForAccountCode(accountCode uint64) (*common.Account, error)
	ActiveReferralProgramForAccount(accountID int64, types map[string]reflect.Type) (*common.ReferralProgram, error)
	AssociateRandomAccountCode(accountID int64) (uint64, error)
	CreateAccountPromotion(accountPromotion *common.AccountPromotion) error
	CreateParkedAccount(parkedAccount *common.ParkedAccount) (int64, error)
	CreatePromoCodePrefix(prefix string) error
	CreatePromotion(promotion *common.Promotion) (int64, error)
	CreatePromotionGroup(promotionGroup *common.PromotionGroup) (int64, error)
	CreateReferralProgram(referralProgram *common.ReferralProgram) error
	CreateReferralProgramTemplate(template *common.ReferralProgramTemplate) (int64, error)
	DefaultReferralProgramTemplate(types map[string]reflect.Type) (*common.ReferralProgramTemplate, error)
	DeleteAccountPromotion(accountID, promotionCodeID int64) (int64, error)
	InactivateReferralProgramTemplate(id int64) error
	InsertPromotionReferralRoute(route *common.PromotionReferralRoute) (int64, error)
	LookupPromoCode(code string) (*common.PromoCode, error)
	MarkParkedAccountAsAccountCreated(id int64) error
	ParkedAccount(email string) (*common.ParkedAccount, error)
	PendingPromotionsForAccount(accountID int64, types map[string]reflect.Type) ([]*common.AccountPromotion, error)
	PendingReferralTrackingForAccount(accountID int64) (*common.ReferralTrackingEntry, error)
	PromoCodeForAccountExists(accountID, codeID int64) (bool, error)
	PromoCodePrefixes() ([]string, error)
	Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error)
	PromotionCountInGroupForAccount(accountID int64, group string) (int, error)
	PromotionGroup(name string) (*common.PromotionGroup, error)
	PromotionGroups() ([]*common.PromotionGroup, error)
	PromotionReferralRoutes(lifecycles []string) ([]*common.PromotionReferralRoute, error)
	Promotions(codeIDs []int64, promoTypes []string, types map[string]reflect.Type) ([]*common.Promotion, error)
	ReferralProgram(codeID int64, types map[string]reflect.Type) (*common.ReferralProgram, error)
	ReferralProgramTemplate(id int64, types map[string]reflect.Type) (*common.ReferralProgramTemplate, error)
	ReferralProgramTemplateRouteQuery(params *RouteQueryParams) (*int64, *common.ReferralProgramTemplate, error)
	ReferralProgramTemplates(statuses common.ReferralProgramStatusList, types map[string]reflect.Type) ([]*common.ReferralProgramTemplate, error)
	RouteQueryParamsForAccount(accountID int64) (*RouteQueryParams, error)
	SetDefaultReferralProgramTemplate(id int64) error
	TrackAccountReferral(referralTracking *common.ReferralTrackingEntry) error
	UpdateAccountPromotion(accountID, promoCodeID int64, update *AccountPromotionUpdate, apo AccountPromotionOption) error
	UpdateAccountReferral(accountID int64, status common.ReferralTrackingStatus) error
	UpdateCredit(accountID int64, credit int, currency string) error
	UpdatePromotion(pu *common.PromotionUpdate) (int64, error)
	UpdatePromotionReferralRoute(routeUpdate *common.PromotionReferralRouteUpdate) (int64, error)
	UpdateReferralProgram(accountID, codeID int64, data common.Typed) error
	UpdateReferralProgramStatusesForRoute(routeID int64, newStatus common.ReferralProgramStatus) (int64, error)
	UpdateReferralProgramTemplate(rpt *common.ReferralProgramTemplateUpdate) (int64, error)
}

type TextAPI interface {
	LocalizedText(langID int64, tags []string) (map[string]string, error)
	UpdateLocalizedText(langID int64, tagText map[string]string) error
}

type PatientFeedbackAPI interface {
	PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error)
	RecordPatientFeedback(patientID common.PatientID, feedbackFor string, rating int, comment *string) error
	PatientFeedback(feedbackFor string) ([]*common.PatientFeedback, error)
}

// ParentalConsentProof represents the photoIDs for
// ID verification. If anything is added to this struct/table then
// the IsComplete method must be updated.
type ParentalConsentProof struct {
	SelfiePhotoID       *int64
	GovernmentIDPhotoID *int64
}

// IsComplete returns true iff all pieces of proof have been completed
func (pcp *ParentalConsentProof) IsComplete() bool {
	return pcp.SelfiePhotoID != nil && pcp.GovernmentIDPhotoID != nil
}

// ParentalConsent are DAL functions for dealing with parent/child treatment consent
type ParentalConsent interface {
	// GrantParentChildConsent creates a relationship between the patient accounts and consents to treatment.
	// However, this doesn't update the patient because we can't allow the patient to do a visit until
	// we've also collected the parent's identification photos. It returns true iff consent had not preivously
	// been granted.
	GrantParentChildConsent(parentPatientID, childPatientID common.PatientID, relationship string) (bool, error)
	// ParentalConsentCompletedForPatient updates the patient record and visits to reflect consent has been granted
	// and all necessary information has been recorded (identification photos).  It returns true iff consent had not
	// previously been completed for the patient.
	ParentalConsentCompletedForPatient(childPatientID common.PatientID) (bool, error)
	// ParentalConsent returns the consent status for a given child
	ParentalConsent(childPatientID common.PatientID) ([]*common.ParentalConsent, error)
	// AllParentalConsent returns the full set of parent/child consent relationships which
	// is a mapping from child's patient ID to the ParentalConsent model.
	AllParentalConsent(parentPatientID common.PatientID) (map[common.PatientID]*common.ParentalConsent, error)
	// UpsertParentConsentProof updates the proof of parental consent based on the
	// data present in the object if it already exists, and inserts otherwise.
	UpsertParentConsentProof(parentPatientID common.PatientID, proof *ParentalConsentProof) (int64, error)
	// ParentConsentProof returns an object that corresponds to the proof of ID verification.
	ParentConsentProof(parentPatientID common.PatientID) (*ParentalConsentProof, error)
}

// Tokens is a set of DAL functions that let you create temporary tokens to use for validation.
// The functionality is similar (and mostly equivalent) to using signatures, but it allows for
// shorter (read nicer) tokens and to invalidate tokens outside of just an expiration.
//
// NOTE: this functional overlaps with the temp tokens in AuthAPI but this is more generic.
// They may merge at some point, but for now it's simplest to just have them separate.
type Tokens interface {
	// CreateToken creates a new token for the specific purpose and key. The key can be used
	// by the caller to store what the token is protecting. For instance, it could be an account ID.
	// If the token is not provided (empty) then one will be generated.
	CreateToken(purpose, key, token string, expire time.Duration) (string, error)
	// ValidateToken returns the key for the token if it's valid. Otherwise it returns ErrTokenDoesNotExist
	ValidateToken(purpose, token string) (string, error)
	// DeleteToken deletes a specific token and returns number of rows deleted.
	// It's useful for invalidation / revoking access.
	DeleteToken(purpose, token string) (int, error)
}

type DataAPI interface {
	AnalyticsAPI
	BankingAPI
	CaseMessageAPI
	CaseRouteAPI
	CostAPI
	DiagnosisAPI
	DoctorAPI
	DoctorManagementAPI
	DrugAPI
	EmailAPI
	FavoriteTreatmentPlanAPI
	PatientFeedbackAPI
	FormAPI
	GeoAPI
	IntakeAPI
	IntakeLayoutAPI
	MediaAPI
	MedicalRecordAPI
	NotificationAPI
	ParentalConsent
	Pathways
	PatientAPI
	PatientCaseAPI
	PatientVisitAPI
	PeopleAPI
	PrescriptionsAPI
	PromotionsAPI
	ResourceLibraryAPI
	ScheduledMessageAPI
	SearchAPI
	SKUs
	TextAPI
	Tokens
	TrainingCasesAPI
}

type AuthTokenPurpose string

func (p AuthTokenPurpose) String() string {
	return string(p)
}

const (
	LostPassword       AuthTokenPurpose = "LostPassword"
	LostPasswordCode   AuthTokenPurpose = "LostPasswordCode"
	PasswordReset      AuthTokenPurpose = "PasswordReset"
	TwoFactorAuthToken AuthTokenPurpose = "TwoFactorAuthToken"
	TwoFactorAuthCode  AuthTokenPurpose = "TwoFactorAuthCode"
)

type Platform string

const (
	Mobile Platform = "mobile"
	Web    Platform = "web"
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

type SKUs interface {
	SKUForPathway(pathwayTag string, category common.SKUCategoryType) (*common.SKU, error)
	SKU(skuType string) (*common.SKU, error)
	CategoryForSKU(skuType string) (*common.SKUCategoryType, error)
	CreateSKU(sku *common.SKU) (int64, error)
}

type AuthAPI interface {
	Authenticate(email, password string) (*common.Account, error)
	CreateAccount(email, password, roleType string) (int64, error)
	CreateToken(accountID int64, platform Platform, options CreateTokenOption) (string, error)
	DeleteToken(token string) error
	GetAccount(id int64) (*common.Account, error)
	AccountForEmail(email string) (*common.Account, error)
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
	CreateTempToken(accountID int64, expireSec int, purpose AuthTokenPurpose, token string) (string, error)
	ValidateTempToken(purpose AuthTokenPurpose, token string) (*common.Account, error)
	DeleteTempToken(purpose AuthTokenPurpose, token string) error
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
	RecordForm(form Form, source string, requestID uint64) error
	FormEntryExists(tableName, uniqueKey string) (bool, error)
}

type LockAPI interface {
	Locked() bool
	Wait() bool
	Release()
}

package api

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/pharmacy"
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

	// TODO: This is temporary until we remove all hardcoded cases of pathways
	AcnePathwayTag = "health_condition_acne"

	MinimumPasswordLength  = 6
	ReviewPurpose          = "REVIEW"
	ConditionIntakePurpose = "CONDITION_INTAKE"
	DiagnosePurpose        = "DIAGNOSE"
)

var (
	NoElligibileProviderInState = errors.New("There are no providers eligible in the state the patient resides")
	NoDiagnosisResponseErr      = errors.New("No diagnosis response exists to the question queried tag queried with")
)

type ErrNotFound string

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("object of type '%s' not found", string(e))
}

func IsErrNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}

type PatientAPI interface {
	Patient(id int64, basicInfoOnly bool) (*common.Patient, error)
	Patients(ids []int64) (map[int64]*common.Patient, error)
	GetPatientFromID(patientID int64) (patient *common.Patient, err error)
	GetPatientFromAccountID(accountID int64) (patient *common.Patient, err error)
	GetPatientFromErxPatientID(erxPatientID int64) (*common.Patient, error)
	GetPatientFromRefillRequestID(refillRequestID int64) (*common.Patient, error)
	GetPatientFromTreatmentID(treatmentID int64) (*common.Patient, error)
	GetPatientFromCaseID(patientCaseID int64) (*common.Patient, error)
	GetPatientFromUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId int64) (*common.Patient, error)
	GetPatientVisitsForPatient(patientID int64) ([]*common.PatientVisit, error)
	PatientLocation(patientID int64) (zipcode string, state string, err error)
	AnyVisitSubmitted(patientID int64) (bool, error)
	RegisterPatient(patient *common.Patient) error
	UpdatePatient(id int64, update *PatientUpdate, updateFromDoctor bool) error
	CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, pathwayTag string) error
	UpdatePatientWithERxPatientID(patientID, erxPatientID int64) error
	GetPatientIDFromAccountID(accountID int64) (int64, error)
	AddDoctorToCareTeamForPatient(patientID, doctorID int64, pathwayTag string) error
	GetCareTeamsForPatientByCase(patientID int64) (map[int64]*common.PatientCareTeam, error)
	GetCareTeamForPatient(patientID int64) (*common.PatientCareTeam, error)
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
	MedicalRecordsForPatient(patientID int64) ([]*common.MedicalRecord, error)
	MedicalRecord(id int64) (*common.MedicalRecord, error)
	CreateMedicalRecord(patientID int64) (int64, error)
	UpdateMedicalRecord(id int64, update *MedicalRecordUpdate) error
}

type PatientCaseUpdate struct {
	Status      *common.CaseStatus
	ClosedDate  *time.Time
	TimeoutDate NullableTime
}

type PatientCaseAPI interface {
	CasesForPathway(patientID int64, pathwayTag string, states []string) ([]*common.PatientCase, error)
	TimedOutCases() ([]*common.PatientCase, error)
	GetDoctorsAssignedToPatientCase(patientCaseID int64) ([]*common.CareProviderAssignment, error)
	GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error)
	GetActiveMembersOfCareTeamForCase(patientCaseID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
	GetPatientCaseFromPatientVisitID(patientVisitID int64) (*common.PatientCase, error)
	GetPatientCaseFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientCase, error)
	GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error)
	AddDoctorToPatientCase(doctorID, caseID int64) error
	DoesActiveTreatmentPlanForCaseExist(patientCaseID int64) (bool, error)
	GetActiveTreatmentPlanForCase(patientCaseID int64) (*common.TreatmentPlan, error)
	GetTreatmentPlansForCase(patientCaseID int64) ([]*common.TreatmentPlan, error)
	DeleteDraftTreatmentPlanByDoctorForCase(doctorID, patientCaseID int64) error
	GetCasesForPatient(patientID int64, states []string) ([]*common.PatientCase, error)
	GetVisitsForCase(patientCaseID int64, statuses []string) ([]*common.PatientVisit, error)
	GetNotificationsForCase(patientCaseID int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error)
	NotificationsForCases(patientID int64, notificationTypeRegistry map[string]reflect.Type) (map[int64][]*common.CaseNotification, error)
	GetNotificationCountForCase(patientCaseID int64) (int64, error)
	InsertCaseNotification(caseNotificationItem *common.CaseNotification) error
	DeleteCaseNotification(uid string, patientCaseID int64) error
	UpdatePatientCase(id int64, update *PatientCaseUpdate) error
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
	GetTempClaimedCaseInQueue(patientCaseID int64) (*DoctorQueueItem, error)
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

type PatientUpdate struct {
	FirstName    *string
	MiddleName   *string
	LastName     *string
	Prefix       *string
	Suffix       *string
	DOB          *encoding.Date
	Gender       *string
	PhoneNumbers []*common.PhoneNumber
	Address      *common.Address
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

type TreatmentPlanUpdate struct {
	Status        *common.TreatmentPlanStatus
	PatientViewed *bool
}

type PatientVisitAPI interface {
	GetPatientIDFromPatientVisitID(patientVisitID int64) (int64, error)
	GetLatestSubmittedPatientVisit() (*common.PatientVisit, error)
	GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error)
	GetPatientVisitFromID(patientVisitID int64) (*common.PatientVisit, error)
	GetPatientVisitForSKU(patientID int64, skuType string) (*common.PatientVisit, error)
	GetPatientVisitFromTreatmentPlanID(treatmentPlanID int64) (*common.PatientVisit, error)
	GetPatientCaseIDFromPatientVisitID(patientVisitID int64) (int64, error)
	PendingFollowupVisitForCase(caseID int64) (*common.PatientVisit, error)
	CreatePatientVisit(visit *common.PatientVisit) (int64, error)
	SetMessageForPatientVisit(patientVisitID int64, message string) error
	GetMessageForPatientVisit(patientVisitID int64) (string, error)
	GetPatientIDFromTreatmentPlanID(treatmentPlanID int64) (int64, error)
	UpdatePatientVisit(id int64, update *PatientVisitUpdate) error
	UpdatePatientVisits(ids []int64, update *PatientVisitUpdate) error
	ClosePatientVisit(patientVisitID int64, event string) error
	SubmitPatientVisitWithID(patientVisitID int64) error
	AddTreatmentsForTreatmentPlan(treatments []*common.Treatment, doctorID, treatmentPlanID, patientID int64) error
	GetTreatmentsBasedOnTreatmentPlanID(treatmentPlanID int64) ([]*common.Treatment, error)
	GetTreatmentBasedOnPrescriptionID(erxID int64) (*common.Treatment, error)
	GetTreatmentsForPatient(patientID int64) ([]*common.Treatment, error)
	GetTreatmentFromID(treatmentID int64) (*common.Treatment, error)
	UpdateTreatmentWithPharmacyAndErxID(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error
	AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error
	GetPrescriptionStatusEventsForPatient(erxPatientID int64) ([]common.StatusEvent, error)
	GetPrescriptionStatusEventsForTreatment(treatmentID int64) ([]common.StatusEvent, error)

	GetOldestVisitsInStatuses(max int, statuses []string) ([]*ItemAge, error)
	UpdateDiagnosisForVisit(id, doctorID int64, diagnosis string) error
	DiagnosisForVisit(visitID int64) (string, error)
	DoesCaseExistForPatient(patientID, patientCaseID int64) (bool, error)

	// treatment plan
	UpdateTreatmentPlan(treatmentPlanID int64, update *TreatmentPlanUpdate) error
	MarkTPDeviatedFromContentSource(treatmentPlanID int64) error
	GetActiveTreatmentPlansForPatient(patientID int64) ([]*common.TreatmentPlan, error)
	GetTreatmentPlanForPatient(patientID, treatmentPlanID int64) (*common.TreatmentPlan, error)
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
	GetApprovedOrDeniedRefillRequestsForPatient(patientID int64) ([]common.StatusEvent, error)
	GetRefillStatusEventsForRefillRequest(refillRequestID int64) ([]common.StatusEvent, error)
	CreateRefillRequest(*common.RefillRequestItem) error
	AddRefillRequestStatusEvent(refillRequestStatus common.StatusEvent) error
	FilterOutRefillRequestsThatExist(queueItemIDs []int64) ([]int64, error)
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
	SpruceAvailableInState(state string) (bool, error)
	GetCareProvidingStateID(stateAbbreviation, pathwayTag string) (int64, error)
	AddCareProvidingState(stateAbbreviation, fullStateName, pathwayTag string) (int64, error)
	MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorID int64) error
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

type DoctorAPI interface {
	RegisterDoctor(doctor *common.Doctor) (int64, error)
	GetAccountIDFromDoctorID(doctorID int64) (int64, error)
	UpdateDoctor(doctorID int64, req *DoctorUpdate) error
	GetDoctorFromID(doctorID int64) (doctor *common.Doctor, err error)
	Doctor(id int64, basicInfoOnly bool) (doctor *common.Doctor, err error)
	Doctors(id []int64) ([]*common.Doctor, error)
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
	GetPendingItemsInDoctorQueue(doctorID int64) (doctorQueue []*DoctorQueueItem, err error)
	GetCompletedItemsInDoctorQueue(doctorID int64) (doctorQueue []*DoctorQueueItem, err error)
	GetPendingItemsForClinic() ([]*DoctorQueueItem, error)
	GetCompletedItemsForClinic() ([]*DoctorQueueItem, error)
	GetPendingItemCountForDoctorQueue(doctorID int64) (int64, error)
	GetMedicationDispenseUnits(languageID int64) (dispenseUnitIDs []int64, dispenseUnits []string, err error)
	MarkPatientVisitAsOngoingInDoctorQueue(doctorID, patientVisitID int64) error
	AddTreatmentTemplates(treatments []*common.DoctorTreatmentTemplate, doctorID, treatmentPlanID int64) error
	GetTreatmentTemplates(doctorID int64) ([]*common.DoctorTreatmentTemplate, error)
	DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorID int64) error

	InsertItemIntoDoctorQueue(doctorQueueItem DoctorQueueItem) error
	ReplaceItemInDoctorQueue(doctorQueueItem DoctorQueueItem, currentState string) error
	DeleteItemFromDoctorQueue(doctorQueueItem DoctorQueueItem) error
	CompleteVisitOnTreatmentPlanGeneration(doctorID, patientVisitID, treatmentPlanID int64,
		currentState string,
		queueItem *DoctorQueueItem) error

	DoctorAttributes(doctorID int64, names []string) (map[string]string, error)
	UpdateDoctorAttributes(doctorID int64, attributes map[string]string) error
	AddMedicalLicenses([]*common.MedicalLicense) error
	UpdateMedicalLicenses(doctorID int64, licenses []*common.MedicalLicense) error
	MedicalLicenses(doctorID int64) ([]*common.MedicalLicense, error)
	CareProviderProfile(accountID int64) (*common.CareProviderProfile, error)
	UpdateCareProviderProfile(accountID int64, profile *common.CareProviderProfile) error
	GetFirstDoctorWithAClinicianID() (*common.Doctor, error)
	GetOldestTreatmentPlanInStatuses(max int, statuses []common.TreatmentPlanStatus) ([]*TreatmentPlanAge, error)
	DoctorEligibleToTreatInState(state string, doctorID int64, pathwayTag string) (bool, error)
	PatientCaseFeed() ([]*common.PatientCaseFeedItem, error)
	PatientCaseFeedForDoctor(doctorID int64) ([]*common.PatientCaseFeedItem, error)
	UpdatePatientCaseFeedItem(item *common.PatientCaseFeedItem) error

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

	// DEPRECATED: Remove after Buzz Lightyear release
	GetSavedDoctorNote(doctorID int64) (string, error)
	// DEPRECATED: Remove after doctor queue migration
	GetNDQItemsWithoutDescription(n int) ([]*DoctorQueueItem, error)
	GetTotalNumberOfDoctorQueueItemsWithoutDescription() (int, error)
	UpdateDoctorQueueItems(dqItems []*DoctorQueueItem) error
}

type ClinicAPI interface {
	GetAllDoctorsInClinic() ([]*common.Doctor, error)
}

type FavoriteTreatmentPlanAPI interface {
	InsertFavoriteTreatmentPlan(ftp *common.FavoriteTreatmentPlan, pathwayTag string, treatmentPlanID int64) (int64, error)
	FavoriteTreatmentPlansForDoctor(doctorID int64, pathwayTag string) ([]*common.FavoriteTreatmentPlan, error)
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
	PatientPhotoSectionsForQuestionIDs(questionIDs []int64, patientID, patientVisitID int64) (map[int64][]common.Answer, error)
	PreviousPatientAnswersForQuestions(questionIDs []int64, patientID int64, beforeTime time.Time) (map[int64][]common.Answer, error)
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
	LayoutVersionMapping() (PathwayPurposeVersionMapping, error)
	LayoutTemplate(pathwayTag, purpose string, version *common.Version) ([]byte, error)
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
	GetActiveItemCost(skuType string) (*common.ItemCost, error)
	GetItemCost(id int64) (*common.ItemCost, error)
	CreatePatientReceipt(receipt *common.PatientReceipt) error
	GetPatientReceipt(patientID, itemID int64, skuType string, includeLineItems bool) (*common.PatientReceipt, error)
	UpdatePatientReceipt(id int64, update *PatientReceiptUpdate) error
	CreateDoctorTransaction(*common.DoctorTransaction) error
	TransactionsForDoctor(doctorID int64) ([]*common.DoctorTransaction, error)
	TransactionForItem(itemID, doctorID int64, skuType string) (*common.DoctorTransaction, error)
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
	UpdateReferralProgram(accountID, codeID int64, data common.Typed) error
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
	DiagnosisAPI
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
	TrainingCasesAPI
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

type SKUs interface {
	SKUForPathway(pathwayTag string, category common.SKUCategoryType) (*common.SKU, error)
	SKU(skuType string) (*common.SKU, error)
	CategoryForSKU(skuType string) (*common.SKUCategoryType, error)
	CreateSKU(sku *common.SKU) (int64, error)
}

type AuthAPI interface {
	Authenticate(email, password string) (*common.Account, error)
	CreateAccount(email, password, roleType string) (int64, error)
	CreateToken(accountID int64, platform Platform, extended bool) (string, error)
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
	FormEntryExists(tableName, uniqueKey string) (bool, error)
}

type LockAPI interface {
	Locked() bool
	Wait() bool
	Release()
}

type AdminAPI interface {
	Dashboard(id int64) (*common.AdminDashboard, error)
}

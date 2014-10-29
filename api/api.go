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
	GetPatientFromId(patientId int64) (patient *common.Patient, err error)
	GetPatientFromAccountId(accountId int64) (patient *common.Patient, err error)
	GetPatientFromErxPatientId(erxPatientId int64) (*common.Patient, error)
	GetPatientFromRefillRequestId(refillRequestId int64) (*common.Patient, error)
	GetPatientFromTreatmentId(treatmentId int64) (*common.Patient, error)
	GetPatientFromCaseId(patientCaseId int64) (*common.Patient, error)
	GetPatientFromUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId int64) (*common.Patient, error)
	GetPatientVisitsForPatient(patientId int64) ([]*common.PatientVisit, error)
	PatientState(patientID int64) (string, error)
	AnyVisitSubmitted(patientID int64) (bool, error)
	RegisterPatient(patient *common.Patient) error
	UpdateTopLevelPatientInformation(patient *common.Patient) error
	UpdatePatientInformation(patient *common.Patient, updateFromDoctor bool) error
	CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, healthConditionId int64) error
	UpdatePatientWithERxPatientId(patientId, erxPatientId int64) error
	GetPatientIdFromAccountId(accountId int64) (int64, error)
	AddDoctorToCareTeamForPatient(patientId, healthConditionId, doctorId int64) error
	CreateCareTeamForPatientWithPrimaryDoctor(patientId, healthConditionId, doctorId int64) (careTeam *common.PatientCareTeam, err error)
	GetCareTeamForPatient(patientId int64) (careTeam *common.PatientCareTeam, err error)
	IsEligibleToServePatientsInState(state string, healthConditionId int64) (bool, error)
	UpdatePatientAddress(patientId int64, addressLine1, addressLine2, city, state, zipCode, addressType string) error
	UpdatePatientPharmacy(patientId int64, pharmacyDetails *pharmacy.PharmacyData) error
	TrackPatientAgreements(patientId int64, agreements map[string]bool) error
	PatientAgreements(patientID int64) (map[string]time.Time, error)
	GetPatientFromPatientVisitId(patientVisitId int64) (patient *common.Patient, err error)
	GetPatientFromTreatmentPlanId(treatmentPlanId int64) (patient *common.Patient, err error)
	GetPatientsForIds(patientIds []int64) ([]*common.Patient, error)
	GetPharmacySelectionForPatients(patientIds []int64) ([]*pharmacy.PharmacyData, error)
	GetPharmacyBasedOnReferenceIdAndSource(pharmacyid int64, pharmacySource string) (*pharmacy.PharmacyData, error)
	GetPharmacyFromId(pharmacyLocalId int64) (*pharmacy.PharmacyData, error)
	AddPharmacy(pharmacyDetails *pharmacy.PharmacyData) error
	UpdatePatientWithPaymentCustomerId(patientId int64, paymentCustomerId string) error
	CreatePendingTask(workType, status string, itemId int64) (int64, error)
	DeletePendingTask(pendingTaskId int64) error
	AddCardForPatient(patientId int64, card *common.Card) error
	MarkCardInactiveForPatient(patientId int64, card *common.Card) error
	DeleteCardForPatient(patientId int64, card *common.Card) error
	MakeLatestCardDefaultForPatient(patientId int64) (*common.Card, error)
	MakeCardDefaultForPatient(patientId int64, card *common.Card) error
	GetCardsForPatient(patientId int64) ([]*common.Card, error)
	GetDefaultCardForPatient(patientId int64) (*common.Card, error)
	GetCardFromId(cardId int64) (*common.Card, error)
	GetCardFromThirdPartyID(thirdPartyId string) (*common.Card, error)
	UpdateDefaultAddressForPatient(patientId int64, address *common.Address) error
	DeleteAddress(addressId int64) error
	AddAlertsForPatient(patientId int64, alerts []*common.Alert) error
	GetAlertsForPatient(patientId int64) ([]*common.Alert, error)
	UpdatePatientPCP(pcp *common.PCP) error
	DeletePatientPCP(patientId int64) error
	UpdatePatientEmergencyContacts(patientId int64, emergencyContacts []*common.EmergencyContact) error
	GetPatientPCP(patientId int64) (*common.PCP, error)
	GetPatientEmergencyContacts(patientId int64) ([]*common.EmergencyContact, error)
	GetActiveMembersOfCareTeamForPatient(patientId int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
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
	GetDoctorsAssignedToPatientCase(patientCaseId int64) ([]*common.CareProviderAssignment, error)
	GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error)
	GetActiveMembersOfCareTeamForCase(patientCaseId int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
	AssignDoctorToPatientFileAndCase(doctorId int64, patientCase *common.PatientCase) error
	GetPatientCaseFromPatientVisitId(patientVisitId int64) (*common.PatientCase, error)
	GetPatientCaseFromTreatmentPlanId(treatmentPlanId int64) (*common.PatientCase, error)
	GetPatientCaseFromId(patientCaseId int64) (*common.PatientCase, error)
	DoesActiveTreatmentPlanForCaseExist(patientCaseId int64) (bool, error)
	GetActiveTreatmentPlanForCase(patientCaseId int64) (*common.TreatmentPlan, error)
	GetTreatmentPlansForCase(patientCaseId int64) ([]*common.TreatmentPlan, error)
	DeleteDraftTreatmentPlanByDoctorForCase(doctorId, patientCaseId int64) error
	GetCasesForPatient(patientId int64) ([]*common.PatientCase, error)
	GetVisitsForCase(patientCaseId int64, statuses []string) ([]*common.PatientVisit, error)
	GetNotificationsForCase(patientCaseId int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error)
	GetNotificationCountForCase(patientCaseId int64) (int64, error)
	InsertCaseNotification(caseNotificationItem *common.CaseNotification) error
	DeleteCaseNotification(uid string, patientCaseId int64) error
}

type DoctorNotify struct {
	DoctorID     int64
	LastNotified *time.Time
}

type CaseRouteAPI interface {
	TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorId int64, patientCase *common.PatientCase, duration time.Duration) error
	TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(doctorId int64, patientCase *common.PatientCase) error
	PermanentlyAssignDoctorToCaseAndRouteToQueue(doctorId int64, patientCase *common.PatientCase, queueItem *DoctorQueueItem) error
	ExtendClaimForDoctor(doctorId, patientId, patientCaseId int64, duration time.Duration) error
	GetClaimedItemsInQueue() ([]*DoctorQueueItem, error)
	GetTempClaimedCaseInQueue(patientCaseId, doctorId int64) (*DoctorQueueItem, error)
	GetElligibleItemsInUnclaimedQueue(doctorId int64) ([]*DoctorQueueItem, error)
	GetAllItemsInUnclaimedQueue() ([]*DoctorQueueItem, error)
	OldestUnclaimedItems(maxItems int) ([]*ItemAge, error)
	InsertUnclaimedItemIntoQueue(doctorQueueItem *DoctorQueueItem) error
	RevokeDoctorAccessToCase(patientCaseId, patientId, doctorId int64) error
	CareProvidingStatesWithUnclaimedCases() ([]int64, error)
	DoctorsToNotifyInCareProvidingState(careProvidingStateID int64, avoidDoctorsRegisteredInStates []int64, timeThreshold time.Time) ([]*DoctorNotify, error)
	RecordDoctorNotifiedOfUnclaimedCases(doctorID int64) error
	RecordCareProvidingStateNotified(careProvidingStateID int64) error
	LastNotifiedTimeForCareProvidingState(careProvidingStateID int64) (time.Time, error)
}

type PatientVisitUpdate struct {
	Status          *string
	LayoutVersionID *int64
	SubmittedDate   *time.Time
}

type ItemAge struct {
	ID  int64
	Age time.Duration
}

type PatientVisitAPI interface {
	GetLastCreatedPatientVisit(patientId int64) (*common.PatientVisit, error)
	GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error)
	GetLatestSubmittedPatientVisit() (*common.PatientVisit, error)
	GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error)
	GetPatientVisitFromId(patientVisitId int64) (*common.PatientVisit, error)
	GetPatientVisitForSKU(patientID int64, skuType sku.SKU) (*common.PatientVisit, error)
	GetPatientVisitFromTreatmentPlanId(treatmentPlanId int64) (*common.PatientVisit, error)
	GetPatientCaseIdFromPatientVisitId(patientVisitId int64) (int64, error)
	PendingFollowupVisitForCase(caseID int64) (*common.PatientVisit, error)
	IsFollowupVisit(patientVisitID int64) (bool, error)
	CreatePatientVisit(visit *common.PatientVisit) (int64, error)
	SetMessageForPatientVisit(patientVisitId int64, message string) error
	GetMessageForPatientVisit(patientVisitId int64) (string, error)
	StartNewTreatmentPlan(patientId, patientVisitId, doctorId int64, parent *common.TreatmentPlanParent, contentSource *common.TreatmentPlanContentSource) (int64, error)
	GetAbridgedTreatmentPlan(treatmentPlanId, doctorId int64) (*common.DoctorTreatmentPlan, error)
	UpdateTreatmentPlanStatus(treatmentPlanID int64, status common.TreatmentPlanStatus) error
	GetTreatmentPlan(treatmentPlanId, doctorId int64) (*common.DoctorTreatmentPlan, error)
	GetAbridgedTreatmentPlanList(doctorId, patientId int64, statuses []common.TreatmentPlanStatus) ([]*common.DoctorTreatmentPlan, error)
	GetAbridgedTreatmentPlanListInDraftForDoctor(doctorId, patientId int64) ([]*common.DoctorTreatmentPlan, error)
	DeleteTreatmentPlan(treatmentPlanId int64) error
	GetPatientIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error)
	UpdatePatientVisit(id int64, update *PatientVisitUpdate) error
	UpdatePatientVisits(ids []int64, update *PatientVisitUpdate) error
	ClosePatientVisit(patientVisitId int64, event string) error
	ActivateTreatmentPlan(treatmentPlanId, doctorId int64) error
	SubmitPatientVisitWithId(patientVisitId int64) error
	GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, patientVisitId int64) ([]*common.AnswerIntake, error)
	DeactivatePreviousDiagnosisForPatientVisit(patientCaseID int64, doctorId int64) error
	GetAdvicePointsForTreatmentPlan(treatmentPlanId int64) (advicePoints []*common.DoctorInstructionItem, err error)
	CreateAdviceForTreatmentPlan(advicePoints []*common.DoctorInstructionItem, treatmentPlanId int64) error
	CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan) error
	GetRegimenPlanForTreatmentPlan(treatmentPlanId int64) (regimenPlan *common.RegimenPlan, err error)
	AddTreatmentsForTreatmentPlan(treatments []*common.Treatment, doctorId, treatmentPlanId, patientId int64) error
	GetTreatmentsBasedOnTreatmentPlanId(treatmentPlanId int64) ([]*common.Treatment, error)
	GetTreatmentBasedOnPrescriptionId(erxId int64) (*common.Treatment, error)
	GetTreatmentsForPatient(patientId int64) ([]*common.Treatment, error)
	GetTreatmentFromId(treatmentId int64) (*common.Treatment, error)
	GetActiveTreatmentPlanForPatient(patientId int64) (*common.TreatmentPlan, error)
	GetTreatmentPlanForPatient(patientId, treatmentPlanId int64) (*common.TreatmentPlan, error)
	IsRevisedTreatmentPlan(treatmentPlanID int64) (bool, error)
	StartRXRoutingForTreatmentsAndTreatmentPlan(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, treatmentPlanID, doctorID int64) error
	UpdateTreatmentWithPharmacyAndErxId(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorId int64) error
	AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error
	GetPrescriptionStatusEventsForPatient(erxPatientID int64) ([]common.StatusEvent, error)
	GetPrescriptionStatusEventsForTreatment(treatmentId int64) ([]common.StatusEvent, error)
	MarkTPDeviatedFromContentSource(treatmentPlanId int64) error
	GetOldestVisitsInStatuses(max int, statuses []string) ([]*ItemAge, error)
	UpdateDiagnosisForVisit(id, doctorID int64, diagnosis string) error
	DiagnosisForVisit(visitID int64) (string, error)
}

type RefillRequestDenialReason struct {
	Id           int64  `json:"id,string"`
	DenialCode   string `json:"denial_code"`
	DenialReason string `json:"denial_reason"`
}

type PrescriptionsAPI interface {
	GetPendingRefillRequestStatusEventsForClinic() ([]common.StatusEvent, error)
	GetApprovedOrDeniedRefillRequestsForPatient(patientId int64) ([]common.StatusEvent, error)
	GetRefillStatusEventsForRefillRequest(refillRequestId int64) ([]common.StatusEvent, error)
	CreateRefillRequest(*common.RefillRequestItem) error
	AddRefillRequestStatusEvent(refillRequestStatus common.StatusEvent) error
	GetRefillRequestFromId(refillRequestId int64) (*common.RefillRequestItem, error)
	GetRefillRequestFromPrescriptionId(prescriptionId int64) (*common.RefillRequestItem, error)
	GetRefillRequestsForPatient(patientId int64) ([]*common.RefillRequestItem, error)
	GetRefillRequestDenialReasons() ([]*RefillRequestDenialReason, error)
	MarkRefillRequestAsApproved(prescriptionId, approvedRefillCount, rxRefillRequestId int64, comments string) error
	MarkRefillRequestAsDenied(prescriptionId, denialReasonId, rxRefillRequestId int64, comments string) error
	LinkRequestedPrescriptionToOriginalTreatment(requestedTreatment *common.Treatment, patient *common.Patient) error
	AddUnlinkedTreatmentInEventOfDNTF(treatment *common.Treatment, refillRequestId int64) error
	GetUnlinkedDNTFTreatment(treatmentId int64) (*common.Treatment, error)
	GetUnlinkedDNTFTreatmentsForPatient(patientId int64) ([]*common.Treatment, error)
	GetUnlinkedDNTFTreatmentFromPrescriptionId(prescriptionId int64) (*common.Treatment, error)
	AddTreatmentToTreatmentPlanInEventOfDNTF(treatment *common.Treatment, refillRequestId int64) error
	UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxId(treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorId int64) error
	AddErxStatusEventForDNTFTreatment(statusEvent common.StatusEvent) error
	GetErxStatusEventsForDNTFTreatment(treatmentId int64) ([]common.StatusEvent, error)
	GetErxStatusEventsForDNTFTreatmentBasedOnPatientId(patientId int64) ([]common.StatusEvent, error)
}

type DrugAPI interface {
	DoesDrugDetailsExist(ndc string) (bool, error)
	DrugDetails(ndc string) (*common.DrugDetails, error)
	ListDrugDetails() ([]*common.DrugDetails, error)
	SetDrugDetails(ndcToDrugDetails map[string]*common.DrugDetails) error
}

type Provider struct {
	ProviderID   int64
	ProviderRole string
}

type DoctorManagementAPI interface {
	GetCareProvidingStateId(stateAbbreviation string, healthConditionId int64) (int64, error)
	AddCareProvidingState(stateAbbreviation, fullStateName string, healthConditionId int64) (int64, error)
	MakeDoctorElligibleinCareProvidingState(careProvidingStateId, doctorId int64) error
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
	GetDoctorFromId(doctorId int64) (doctor *common.Doctor, err error)
	Doctor(id int64, basicInfoOnly bool) (doctor *common.Doctor, err error)
	GetDoctorFromAccountId(accountId int64) (doctor *common.Doctor, err error)
	GetDoctorFromDoseSpotClinicianId(clincianId int64) (doctor *common.Doctor, err error)
	GetDoctorIdFromAccountId(accountId int64) (int64, error)
	GetMAInClinic() (*common.Doctor, error)
	GetRegimenStepsForDoctor(doctorId int64) (regimenSteps []*common.DoctorInstructionItem, err error)
	GetRegimenStepForDoctor(regimenStepId, doctorId int64) (*common.DoctorInstructionItem, error)
	AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorId int64) error
	GetAdvicePointsForDoctor(doctorId int64) (advicePoints []*common.DoctorInstructionItem, err error)
	GetAdvicePointForDoctor(advicePointId, doctorId int64) (*common.DoctorInstructionItem, error)
	AddAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	UpdateAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	MarkAdvicePointToBeDeleted(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	MarkAdvicePointsToBeDeleted(advicePoints []*common.DoctorInstructionItem, doctorId int64) error
	MarkPatientVisitAsOngoingInDoctorQueue(doctorId, patientVisitId int64) error
	GetPendingItemsInDoctorQueue(doctorId int64) (doctorQueue []*DoctorQueueItem, err error)
	GetCompletedItemsInDoctorQueue(doctorId int64) (doctorQueue []*DoctorQueueItem, err error)
	GetPendingItemsForClinic() ([]*DoctorQueueItem, error)
	GetCompletedItemsForClinic() ([]*DoctorQueueItem, error)
	GetPendingItemCountForDoctorQueue(doctorId int64) (int64, error)
	GetMedicationDispenseUnits(languageId int64) (dispenseUnitIds []int64, dispenseUnits []string, err error)
	GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute string, doctorId int64) (drugInstructions []*common.DoctorInstructionItem, err error)
	AddOrUpdateDrugInstructionForDoctor(drugName, drugForm, drugRoute string, drugInstructionToAdd *common.DoctorInstructionItem, doctorId int64) error
	DeleteDrugInstructionForDoctor(drugInstructionToDelete *common.DoctorInstructionItem, doctorId int64) error
	AddDrugInstructionsToTreatment(drugName, drugForm, drugRoute string, drugInstructions []*common.DoctorInstructionItem, treatmentId int64, doctorId int64) error
	AddTreatmentTemplates(treatments []*common.DoctorTreatmentTemplate, doctorId, treatmentPlanId int64) error
	GetTreatmentTemplates(doctorId int64) ([]*common.DoctorTreatmentTemplate, error)
	DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorId int64) error
	InsertItemIntoDoctorQueue(doctorQueueItem DoctorQueueItem) error
	ReplaceItemInDoctorQueue(doctorQueueItem DoctorQueueItem, currentState string) error
	DeleteItemFromDoctorQueue(doctorQueueItem DoctorQueueItem) error
	CompleteVisitOnTreatmentPlanGeneration(doctorId, patientVisitId, treatmentPlanId int64, currentState, updatedState string) error
	GetSavedMessageForDoctor(doctorID int64) (string, error)
	GetTreatmentPlanMessageForDoctor(doctorID, treatmentPlanID int64) (string, error)
	SetSavedMessageForDoctor(doctorID int64, message string) error
	SetTreatmentPlanMessage(doctorID, treatmentPlanID int64, message string) error
	DeleteTreatmentPlanMessage(doctorID, treatmentPlanID int64) error
	DoctorAttributes(doctorID int64, names []string) (map[string]string, error)
	UpdateDoctorAttributes(doctorID int64, attributes map[string]string) error
	AddMedicalLicenses([]*common.MedicalLicense) error
	MedicalLicenses(doctorID int64) ([]*common.MedicalLicense, error)
	CareProviderProfile(accountID int64) (*common.CareProviderProfile, error)
	UpdateCareProviderProfile(accountID int64, profile *common.CareProviderProfile) error
	GetFirstDoctorWithAClinicianId() (*common.Doctor, error)
	GetOldestTreatmentPlanInStatuses(max int, statuses []common.TreatmentPlanStatus) ([]*TreatmentPlanAge, error)
	DoctorEligibleToTreatInState(state string, doctorID, healthConditionID int64) (bool, error)
}

type ClinicAPI interface {
	GetAllDoctorsInClinic() ([]*common.Doctor, error)
}

type FavoriteTreatmentPlanAPI interface {
	CreateOrUpdateFavoriteTreatmentPlan(favoriteTreatmentPlan *common.FavoriteTreatmentPlan, treatmentPlanId int64) error
	GetFavoriteTreatmentPlansForDoctor(doctorId int64) ([]*common.FavoriteTreatmentPlan, error)
	GetFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.FavoriteTreatmentPlan, error)
	DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanID, doctorID int64) error
	GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) ([]*common.Treatment, error)
	GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.RegimenPlan, error)
	GetAdviceInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.Advice, error)
}

type ColumnValue struct {
	Column string
	Value  interface{}
}

type IntakeInfo interface {
	TableName() string
	Role() *ColumnValue
	Context() *ColumnValue
	LayoutVersionID() int64
	Answers() map[int64][]*common.AnswerIntake
}

type IntakeAPI interface {
	GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (map[int64][]common.Answer, error)
	GetPatientCreatedPhotoSectionsForQuestionId(questionId, patientId, patientVisitId int64) ([]common.Answer, error)
	GetPatientCreatedPhotoSectionsForQuestionIds(questionIds []int64, patientId, patientVisitId int64) (map[int64][]common.Answer, error)
	AnswersForQuestions(questionIds []int64, info IntakeInfo) (map[int64][]common.Answer, error)
	StoreAnswersForQuestion(info IntakeInfo) error
	StorePhotoSectionsForQuestion(questionId, patientId, patientVisitId int64, photoSections []*common.PhotoIntakeSection) error
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
	LatestAppVersionSupported(healthConditionId int64, skuID *int64, platform common.Platform, role, purpose string) (*common.Version, error)
	LayoutTemplateVersionBeyondVersion(versionInfo *VersionInfo, role, purpose string, healthConditionID int64, skuID *int64) (*LayoutTemplateVersion, error)
	GetActiveDoctorDiagnosisLayout(healthConditionId int64) (*LayoutVersion, error)
	GetPatientLayout(layoutVersionId, languageId int64) (*LayoutVersion, error)
	GetLayoutVersionIdOfActiveDiagnosisLayout(healthConditionId int64) (int64, error)
	GetGlobalSectionIds() ([]int64, error)
	GetSectionIdsForHealthCondition(healthConditionId int64) ([]int64, error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
	GetQuestionType(questionId int64) (questionType string, err error)
	GetQuestionInfo(questionTag string, languageId int64) (*info_intake.Question, error)
	GetQuestionInfoForTags(questionTags []string, languageId int64) ([]*info_intake.Question, error)
	GetAnswerInfo(questionId int64, languageId int64) (answerInfos []*info_intake.PotentialAnswer, err error)
	GetAnswerInfoForTags(answerTags []string, languageId int64) ([]*info_intake.PotentialAnswer, error)
	GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
	GetPhotoSlots(questionId, languageId int64) ([]*info_intake.PhotoSlot, error)
}

type ObjectStorageDBAPI interface {
	CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error)
	UpdateCloudObjectRecordToSayCompleted(id int64) error
}

type PeopleAPI interface {
	GetPeople(ids []int64) (map[int64]*common.Person, error)
	GetPersonIdByRole(roleType string, roleId int64) (int64, error)
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
	DeletePushCommunicationPreferenceForAccount(accountId int64) error
	GetPushConfigDataForAccount(accountId int64) ([]*common.PushConfigData, error)
	SetOrReplacePushConfigData(pConfigData *common.PushConfigData) error
	GetCommunicationPreferencesForAccount(accountId int64) ([]*common.CommunicationPreference, error)
	SetPushPromptStatus(patientId int64, pStatus common.PushPromptStatus) error
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
	GetSignedUrlForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error)
	DeleteObjectAtLocation(bucket, key, region string) error
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error)
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
	DeleteTempTokensForAccount(accountId int64) error
	// Permissions
	AvailableAccountPermissions() ([]string, error)
	AvailableAccountGroups(withPermissions bool) ([]*common.AccountGroup, error)
	PermissionsForAccount(accountID int64) ([]string, error)
	GroupsForAccount(accountID int64) ([]*common.AccountGroup, error)
	UpdateGroupsForAccount(accountID int64, groups map[int64]bool) error
	UpdateAppDevice(accountID int64, appVersion *common.Version, p common.Platform, platformVersion, device, deviceModel string) error
	LatestAppPlatformVersion(accountID int64) (*common.Platform, *common.Version, error)
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

package api

import (
	"errors"
	"net/http"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/pharmacy"

	"github.com/sprucehealth/backend/common"
)

const (
	EN_LANGUAGE_ID                 = 1
	ADMIN_ROLE                     = "ADMIN"
	DOCTOR_ROLE                    = "DOCTOR"
	PRIMARY_DOCTOR_STATUS          = "PRIMARY"
	PATIENT_ROLE                   = "PATIENT"
	MA_ROLE                        = "MA"
	REVIEW_PURPOSE                 = "REVIEW"
	CONDITION_INTAKE_PURPOSE       = "CONDITION_INTAKE"
	DIAGNOSE_PURPOSE               = "DIAGNOSE"
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

	MinimumPasswordLength = 6
)

var (
	NoRowsError                 = errors.New("No rows exist")
	NoElligibileProviderInState = errors.New("There are no providers elligible in the state the patient resides")
	NoDiagnosisResponseErr      = errors.New("No diagnosis response exists to the question queried tag queried with")
)

type PatientAPI interface {
	GetPatientFromId(patientId int64) (patient *common.Patient, err error)
	GetPatientFromAccountId(accountId int64) (patient *common.Patient, err error)
	GetPatientFromErxPatientId(erxPatientId int64) (*common.Patient, error)
	GetPatientFromRefillRequestId(refillRequestId int64) (*common.Patient, error)
	GetPatientFromTreatmentId(treatmentId int64) (*common.Patient, error)
	GetPatientFromCaseId(patientCaseId int64) (*common.Patient, error)
	GetPatientFromUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId int64) (*common.Patient, error)
	GetPatientVisitsForPatient(patientId int64) ([]*common.PatientVisit, error)
	RegisterPatient(patient *common.Patient) error
	UpdateTopLevelPatientInformation(patient *common.Patient) error
	UpdatePatientInformation(patient *common.Patient, updateFromDoctor bool) error
	CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, healthConditionId int64) error
	UpdatePatientWithERxPatientId(patientId, erxPatientId int64) error
	GetPatientIdFromAccountId(accountId int64) (int64, error)
	AddDoctorToCareTeamForPatient(patientId, healthConditionId, doctorId int64) error
	CreateCareTeamForPatientWithPrimaryDoctor(patientId, healthConditionId, doctorId int64) (careTeam *common.PatientCareTeam, err error)
	GetCareTeamForPatient(patientId int64) (careTeam *common.PatientCareTeam, err error)
	IsEligibleToServePatientsInState(shortState string, healthConditionId int64) (bool, error)
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
	AddCardAndMakeDefaultForPatient(patientId int64, card *common.Card) error
	MarkCardInactiveForPatient(patientId int64, card *common.Card) error
	DeleteCardForPatient(patientId int64, card *common.Card) error
	MakeLatestCardDefaultForPatient(patientId int64) (*common.Card, error)
	MakeCardDefaultForPatient(patientId int64, card *common.Card) error
	GetCardsForPatient(patientId int64) ([]*common.Card, error)
	GetCardFromId(cardId int64) (*common.Card, error)
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
	GetActiveMembersOfCareTeamForCase(patientCaseId int64, fillInDetails bool) ([]*common.CareProviderAssignment, error)
	AssignDoctorToPatientFileAndCase(doctorId int64, patientCase *common.PatientCase) error
	GetPatientCaseFromPatientVisitId(patientVisitId int64) (*common.PatientCase, error)
	GetPatientCaseFromTreatmentPlanId(treatmentPlanId int64) (*common.PatientCase, error)
	GetPatientCaseFromId(patientCaseId int64) (*common.PatientCase, error)
	DoesActiveTreatmentPlanForCaseExist(patientCaseId int64) (bool, error)
	GetActiveTreatmentPlanForCase(patientCaseId int64) (*common.TreatmentPlan, error)
	GetAllTreatmentPlansForCase(patientCaseId int64) ([]*common.TreatmentPlan, error)
	DeleteDraftTreatmentPlanByDoctorForCase(doctorId, patientCaseId int64) error
	GetCasesForPatient(patientId int64) ([]*common.PatientCase, error)
	GetVisitsForCase(patientCaseId int64) ([]*common.PatientVisit, error)
	GetNotificationsForCase(patientCaseId int64, notificationTypeRegistry map[string]reflect.Type) ([]*common.CaseNotification, error)
	GetNotificationCountForCase(patientCaseId int64) (int64, error)
	InsertCaseNotification(caseNotificationItem *common.CaseNotification) error
	DeleteCaseNotification(uid string, patientCaseId int64) error
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
	InsertUnclaimedItemIntoQueue(doctorQueueItem *DoctorQueueItem) error
	RevokeDoctorAccessToCase(patientCaseId, patientId, doctorId int64) error
}

type PatientVisitAPI interface {
	GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error)
	GetLastCreatedPatientVisitIdForPatient(patientId int64) (int64, error)
	GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error)
	GetLatestSubmittedPatientVisit() (*common.PatientVisit, error)
	GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error)
	GetLatestClosedPatientVisitForPatient(patientId int64) (*common.PatientVisit, error)
	GetPatientVisitFromId(patientVisitId int64) (*common.PatientVisit, error)
	GetPatientVisitFromTreatmentPlanId(treatmentPlanId int64) (*common.PatientVisit, error)
	GetPatientCaseIdFromPatientVisitId(patientVisitId int64) (int64, error)
	CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (*common.PatientVisit, error)
	SetMessageForPatientVisit(patientVisitId int64, message string) error
	GetMessageForPatientVisit(patientVisitId int64) (string, error)
	UpdateDiagnosisForPatientVisit(patientVisitId int64, diagnosis string) error
	StartNewTreatmentPlan(patientId, patientVisitId, doctorId int64, parent *common.TreatmentPlanParent, contentSource *common.TreatmentPlanContentSource) (int64, error)
	GetAbridgedTreatmentPlan(treatmentPlanId, doctorId int64) (*common.DoctorTreatmentPlan, error)
	GetTreatmentPlan(treatmentPlanId, doctorId int64) (*common.DoctorTreatmentPlan, error)
	GetAbridgedTreatmentPlanList(doctorId, patientId int64, status string) ([]*common.DoctorTreatmentPlan, error)
	GetAbridgedTreatmentPlanListInDraftForDoctor(doctorId, patientId int64) ([]*common.DoctorTreatmentPlan, error)
	DeleteTreatmentPlan(treatmentPlanId int64) error
	GetPatientIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error)
	UpdatePatientVisitStatus(patientVisitId int64, message, event string) error
	ClosePatientVisit(patientVisitId int64, event string) error
	ActivateTreatmentPlan(treatmentPlanId, doctorId int64) error
	SubmitPatientVisitWithId(patientVisitId int64) error
	GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, patientVisitId int64) ([]*common.AnswerIntake, error)
	DeactivatePreviousDiagnosisForPatientVisit(treatmentPlanId int64, doctorId int64) error
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
	UpdateTreatmentWithPharmacyAndErxId(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorId int64) error
	AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error
	GetPrescriptionStatusEventsForPatient(patientId int64) ([]common.StatusEvent, error)
	GetPrescriptionStatusEventsForTreatment(treatmentId int64) ([]common.StatusEvent, error)
	MarkTPDeviatedFromContentSource(treatmentPlanId int64) error
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

type DoctorManagementAPI interface {
	GetCareProvidingStateId(stateAbbreviation string, healthConditionId int64) (int64, error)
	AddCareProvidingState(stateAbbreviation, fullStateName string, healthConditionId int64) (int64, error)
	MakeDoctorElligibleinCareProvidingState(careProvidingStateId, doctorId int64) error
	GetDoctorWithEmail(email string) (*common.Doctor, error)
}

type DoctorAPI interface {
	RegisterDoctor(doctor *common.Doctor) (int64, error)
	UpdateDoctor(doctorID int64, req *DoctorUpdate) error
	GetDoctorFromId(doctorId int64) (doctor *common.Doctor, err error)
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
	MarkGenerationOfTreatmentPlanInVisitQueue(doctorId, patientVisitId, treatmentPlanId int64, currentState, updatedState string) error
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
}

type ClinicAPI interface {
	GetAllDoctorsInClinic() ([]*common.Doctor, error)
}

type FavoriteTreatmentPlanAPI interface {
	CreateOrUpdateFavoriteTreatmentPlan(favoriteTreatmentPlan *common.FavoriteTreatmentPlan, treatmentPlanId int64) error
	GetFavoriteTreatmentPlansForDoctor(doctorId int64) ([]*common.FavoriteTreatmentPlan, error)
	GetFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.FavoriteTreatmentPlan, error)
	DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) error
	GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) ([]*common.Treatment, error)
	GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.RegimenPlan, error)
	GetAdviceInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.Advice, error)
}

type IntakeAPI interface {
	GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (map[int64][]common.Answer, error)
	GetPatientAnswersForQuestions(questionIds []int64, patientId int64, patientVisitId int64) (map[int64][]common.Answer, error)
	GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds []int64, roleId int64, patientVisitId int64) (map[int64][]common.Answer, error)
	GetPatientCreatedPhotoSectionsForQuestionId(questionId, patientId, patientVisitId int64) ([]common.Answer, error)
	GetPatientCreatedPhotoSectionsForQuestionIds(questionIds []int64, patientId, patientVisitId int64) (map[int64][]common.Answer, error)
	StoreAnswersForQuestion(role string, roleId, patientVisitId, layoutVersionId int64, answersToStorePerQuestion map[int64][]*common.AnswerIntake) error
	RejectPatientVisitPhotos(patientVisitId int64) error
	StorePhotoSectionsForQuestion(questionId, patientId, patientVisitId int64, photoSections []*common.PhotoIntakeSection) error
}

type IntakeLayoutAPI interface {
	GetQuestionType(questionId int64) (questionType string, err error)
	GetActiveLayoutForHealthCondition(healthConditionTag, role, purpose string) ([]byte, error)
	GetCurrentActivePatientLayout(languageId, healthConditionId int64) ([]byte, int64, error)
	GetCurrentActiveDoctorLayout(healthConditionId int64) ([]byte, int64, error)
	GetActiveDoctorDiagnosisLayout(healthConditionId int64) ([]byte, int64, error)
	GetPatientLayout(layoutVersionId, languageId int64) ([]byte, error)
	CreateLayoutVersion(layout []byte, syntaxVersion int64, healthConditionId int64, role, purpose, comment string) (int64, error)
	CreatePatientLayout(layout []byte, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	CreateDoctorLayout(layout []byte, layoutVersionId int64, healthConditionId int64) (int64, error)
	GetLayoutVersionIdOfActiveDiagnosisLayout(healthConditionId int64) (int64, error)
	GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error)
	UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error
	UpdateDoctorActiveLayouts(layoutId, doctorLayoutId, healthConditionId int64, purpose string) error
	GetGlobalSectionIds() ([]int64, error)
	GetSectionIdsForHealthCondition(healthConditionId int64) ([]int64, error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
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
}

type NotificationAPI interface {
	GetPushConfigData(deviceToken string) (*common.PushConfigData, error)
	DeletePushCommunicationPreferenceForAccount(accountId int64) error
	GetPushConfigDataForAccount(accountId int64) ([]*common.PushConfigData, error)
	SetOrReplacePushConfigData(pConfigData *common.PushConfigData) error
	GetCommunicationPreferencesForAccount(accountId int64) ([]*common.CommunicationPreference, error)
	SetPushPromptStatus(patientId int64, pStatus common.PushPromptStatus) error
}

type MediaAPI interface {
	AddMedia(uploaderID int64, url, mimetype string) (int64, error)
	GetMedia(mediaID int64) (*common.Media, error)
	ClaimMedia(mediaID int64, claimerType string, claimerID int64) error
}

type ResourceLibraryAPI interface {
	ListResourceGuideSections() ([]*common.ResourceGuideSection, error)
	GetResourceGuide(id int64) (*common.ResourceGuide, error)
	ListResourceGuides() ([]*common.ResourceGuideSection, map[int64][]*common.ResourceGuide, error)
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
	AddBankAccount(userAccountID int64, stripeRecipientID string, defaultAccount bool) (int64, error)
	DeleteBankAccount(id int64) error
	ListBankAccounts(userAccountID int64) ([]*common.BankAccount, error)
	UpdateBankAccountVerficiation(id int64, amount1, amount2 int, transfer1ID, transfer2ID string, expires time.Time, verified bool) error
}

type SearchAPI interface {
	SearchDoctors(query string) ([]*common.DoctorSearchResult, error)
}

type DataAPI interface {
	GeoAPI
	PatientAPI
	DoctorAPI
	ClinicAPI
	DoctorManagementAPI
	PatientVisitAPI
	PatientCaseAPI
	IntakeLayoutAPI
	ObjectStorageDBAPI
	IntakeAPI
	PrescriptionsAPI
	DrugAPI
	PeopleAPI
	CaseMessageAPI
	NotificationAPI
	MediaAPI
	FavoriteTreatmentPlanAPI
	ResourceLibraryAPI
	CaseRouteAPI
	BankingAPI
	SearchAPI
	MedicalRecordAPI
}

type CloudStorageAPI interface {
	GetObjectAtLocation(bucket, key, region string) (rawData []byte, responseHeader http.Header, err error)
	GetSignedUrlForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error)
	DeleteObjectAtLocation(bucket, key, region string) error
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error)
}

const (
	LostPassword     = "lost_password"
	LostPasswordCode = "lost_password_code"
	PasswordReset    = "password_reset"
)

type Platform string

const (
	Mobile Platform = "mobile"
	Web    Platform = "web"
)

type AuthAPI interface {
	CreateAccount(email, password, roleType string) (int64, error)
	Authenticate(email, password string) (*common.Account, error)
	CreateToken(accountID int64, platform Platform) (string, error)
	DeleteToken(token string) error
	ValidateToken(token string, platform Platform) (*common.Account, error)
	GetToken(accountID int64) (string, error)
	SetPassword(accountID int64, password string) error
	UpdateLastOpenedDate(accountID int64) error
	GetAccountForEmail(email string) (*common.Account, error)
	GetAccount(id int64) (*common.Account, error)
	GetPhoneNumbersForAccount(id int64) ([]*common.PhoneNumber, error)
	// Temporary auth tokens
	CreateTempToken(accountId int64, expireSec int, purpose, token string) (string, error)
	ValidateTempToken(purpose, token string) (*common.Account, error)
	DeleteTempToken(purpose, token string) error
	DeleteTempTokensForAccount(accountId int64) error
}
